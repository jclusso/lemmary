package pageedit

import (
	"context"
	"os/exec"
	"path/filepath"
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
