package pageedit

import (
	"context"
	"errors"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"

	"lemmary/backend/internal/models"
	"lemmary/backend/internal/pdftool"
	"lemmary/backend/internal/pdftool/testpdf"
	"lemmary/backend/internal/testpb"
	_ "lemmary/backend/migrations"
)

func requirePoppler(t *testing.T) {
	t.Helper()
	for _, binary := range []string{"pdfinfo", "pdftoppm", "pdfseparate", "pdfunite", "pdftotext"} {
		if _, err := exec.LookPath(binary); err != nil {
			t.Skipf("%s not installed", binary)
		}
	}
}

func makeUser(t *testing.T, app core.App) string {
	t.Helper()
	users, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		t.Fatal(err)
	}
	user := core.NewRecord(users)
	user.Set("email", "owner@example.com")
	user.SetPassword("test-password-123")
	if err := app.Save(user); err != nil {
		t.Fatal(err)
	}
	return user.Id
}

func makeDocument(t *testing.T, app core.App, owner, name string, data []byte) *core.Record {
	t.Helper()
	documents, err := app.FindCollectionByNameOrId("documents")
	if err != nil {
		t.Fatal(err)
	}
	file, err := filesystem.NewFileFromBytes(data, name)
	if err != nil {
		t.Fatal(err)
	}
	document := core.NewRecord(documents)
	document.Set("user", owner)
	document.Set("file", file)
	document.Set("title", "Scan")
	document.Set("ocr_text", "old text")
	document.Set("processing_status", models.DocStatusCompleted)
	if err := app.Save(document); err != nil {
		t.Fatal(err)
	}
	return document
}

// storedPages reads the document's field back from storage and returns the
// text of each of its pages.
func storedPages(t *testing.T, app core.App, document *core.Record, field string) []string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "stored.pdf")
	if err := copyStoredFile(app, document, field, path); err != nil {
		t.Fatal(err)
	}
	count, err := pdftool.PageCount(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	texts, err := pdftool.AllPagesText(context.Background(), path, count)
	if err != nil {
		t.Fatal(err)
	}
	return texts
}

func reload(t *testing.T, app core.App, document *core.Record) *core.Record {
	t.Helper()
	fresh, err := app.FindRecordById("documents", document.Id)
	if err != nil {
		t.Fatal(err)
	}
	return fresh
}

func finishJobs(t *testing.T, app core.App, documentID string) {
	t.Helper()
	jobs, err := app.FindRecordsByFilter("processing_jobs", "document = {:doc}", "", 0, 0, map[string]any{"doc": documentID})
	if err != nil {
		t.Fatal(err)
	}
	for _, job := range jobs {
		job.Set("status", models.JobStatusCompleted)
		job.Set("finished_at", "2026-01-01 00:00:00.000Z")
		if err := app.Save(job); err != nil {
			t.Fatal(err)
		}
	}
}

func TestInspectFlagsTheBlankPages(t *testing.T) {
	requirePoppler(t)
	app := testpb.Open(t)
	document := makeDocument(t, app, makeUser(t, app), "scan.pdf", testpdf.MixedText(3, 2))

	overview, err := Inspect(context.Background(), app, document)
	if err != nil {
		t.Fatal(err)
	}
	if len(overview.Pages) != 3 {
		t.Fatalf("pages = %d, want 3", len(overview.Pages))
	}
	for i, page := range overview.Pages {
		if page.Number != i+1 {
			t.Fatalf("page %d numbered %d", i+1, page.Number)
		}
		if !strings.HasPrefix(page.Thumbnail, "data:image/jpeg;base64,") {
			t.Fatalf("page %d thumbnail = %.40q", page.Number, page.Thumbnail)
		}
		if want := page.Number == 3; page.Blank != want {
			t.Fatalf("page %d blank = %v, want %v", page.Number, page.Blank, want)
		}
	}
	if overview.HasOriginal {
		t.Fatal("an unedited document reports an original")
	}
}

// The smallest page of text testpdf makes, "Page 1" alone, still has to read
// as printed.
func TestInspectKeepsAPageWithOneShortLine(t *testing.T) {
	requirePoppler(t)
	app := testpb.Open(t)
	document := makeDocument(t, app, makeUser(t, app), "scan.pdf", testpdf.Multipage(1))

	overview, err := Inspect(context.Background(), app, document)
	if err != nil {
		t.Fatal(err)
	}
	if overview.Pages[0].Blank {
		t.Fatal("a page reading \"Page 1\" was flagged blank")
	}
}

func TestRemovePagesThenRestore(t *testing.T) {
	requirePoppler(t)
	app := testpb.Open(t)
	document := makeDocument(t, app, makeUser(t, app), "scan.pdf", testpdf.Multipage(4))

	if err := RemovePages(context.Background(), app, document, []int{2, 4}); err != nil {
		t.Fatal(err)
	}
	document = reload(t, app, document)
	pages := storedPages(t, app, document, "file")
	if len(pages) != 2 || !strings.Contains(pages[0], "Page 1") || !strings.Contains(pages[1], "Page 3") {
		t.Fatalf("pages after removal = %q, want Page 1 and Page 3", pages)
	}
	if got := len(storedPages(t, app, document, "original_file")); got != 4 {
		t.Fatalf("original has %d pages, want 4", got)
	}
	if !strings.HasPrefix(document.GetString("file"), "scan_") {
		t.Fatalf("file name = %q, want it to keep the upload's name", document.GetString("file"))
	}
	if document.GetString("processing_status") != models.DocStatusPending {
		t.Fatalf("status = %q, want pending", document.GetString("processing_status"))
	}
	job, err := app.FindFirstRecordByFilter("processing_jobs", "document = {:doc}", map[string]any{"doc": document.Id})
	if err != nil {
		t.Fatalf("no job queued: %v", err)
	}
	var steps, force []string
	if err := job.UnmarshalJSONField("steps", &steps); err != nil {
		t.Fatal(err)
	}
	if err := job.UnmarshalJSONField("force_steps", &force); err != nil {
		t.Fatal(err)
	}
	if slices.Contains(steps, models.StepApplyMetadata) {
		t.Fatalf("steps = %v; removing pages must not overwrite metadata", steps)
	}
	if !slices.Contains(force, models.StepPreview) || !slices.Contains(force, models.StepOCR) {
		t.Fatalf("force_steps = %v, want preview and ocr", force)
	}

	if err := RemovePages(context.Background(), app, document, []int{1}); !errors.Is(err, ErrBusy) {
		t.Fatalf("removal during processing = %v, want ErrBusy", err)
	}
	finishJobs(t, app, document.Id)

	// A second removal keeps the first backup: restoring goes back to the scan.
	if err := RemovePages(context.Background(), app, document, []int{1}); err != nil {
		t.Fatal(err)
	}
	document = reload(t, app, document)
	if got := len(storedPages(t, app, document, "original_file")); got != 4 {
		t.Fatalf("original after a second removal has %d pages, want 4", got)
	}
	finishJobs(t, app, document.Id)

	if err := RestoreOriginal(app, document); err != nil {
		t.Fatal(err)
	}
	document = reload(t, app, document)
	if got := len(storedPages(t, app, document, "file")); got != 4 {
		t.Fatalf("restored file has %d pages, want 4", got)
	}
	if document.GetString("original_file") != "" {
		t.Fatal("the backup survived the restore")
	}
	finishJobs(t, app, document.Id)
	if err := RestoreOriginal(app, document); !errors.Is(err, ErrNoOriginal) {
		t.Fatalf("second restore = %v, want ErrNoOriginal", err)
	}
}

func TestRemovePagesRefuses(t *testing.T) {
	requirePoppler(t)
	app := testpb.Open(t)
	owner := makeUser(t, app)
	document := makeDocument(t, app, owner, "scan.pdf", testpdf.Multipage(2))

	for name, tc := range map[string]struct {
		pages []int
		want  error
	}{
		"nothing":      {nil, ErrNoPages},
		"every page":   {[]int{1, 2}, ErrAllPages},
		"a page past":  {[]int{3}, ErrPageOutOfDoc},
		"page zero":    {[]int{0}, ErrPageOutOfDoc},
		"repeated all": {[]int{1, 1, 2}, ErrAllPages},
	} {
		if err := RemovePages(context.Background(), app, document, tc.pages); !errors.Is(err, tc.want) {
			t.Errorf("%s: err = %v, want %v", name, err, tc.want)
		}
	}

	text := makeDocument(t, app, owner, "note.txt", []byte("plain text"))
	if err := RemovePages(context.Background(), app, text, []int{1}); !errors.Is(err, ErrNotPDF) {
		t.Fatalf("text document: err = %v, want ErrNotPDF", err)
	}
	if _, err := Inspect(context.Background(), app, text); !errors.Is(err, ErrNotPDF) {
		t.Fatalf("inspect text document: err = %v, want ErrNotPDF", err)
	}
}

func TestKeptRanges(t *testing.T) {
	got, err := keptRanges(7, []int{1, 4, 5, 7})
	if err != nil {
		t.Fatal(err)
	}
	want := []pageRange{{2, 3}, {6, 6}}
	if !slices.Equal(got, want) {
		t.Fatalf("ranges = %v, want %v", got, want)
	}
}

func TestUploadNameDropsPocketBaseSuffix(t *testing.T) {
	if got := uploadName("scan_a1b2c3d4e5.pdf"); got != "scan.pdf" {
		t.Fatalf("uploadName = %q, want scan.pdf", got)
	}
	if got := uploadName("scan.pdf"); got != "scan.pdf" {
		t.Fatalf("uploadName = %q, want scan.pdf unchanged", got)
	}
}
