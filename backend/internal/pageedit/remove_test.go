package pageedit

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/core"

	"lemmary/backend/internal/models"
	"lemmary/backend/internal/pdftool/testpdf"
	"lemmary/backend/internal/testpb"
)

// withRecord stores record on the document the way an OCR run would.
func withRecord(t *testing.T, app core.App, document *core.Record, record models.PageRecord) *core.Record {
	t.Helper()
	record.Store(document)
	if err := app.Save(document); err != nil {
		t.Fatal(err)
	}
	return reload(t, app, document)
}

func jobs(t *testing.T, app core.App, documentID string) []*core.Record {
	t.Helper()
	found, err := app.FindRecordsByFilter("processing_jobs", "document = {:doc}", "-created", 0, 0, map[string]any{"doc": documentID})
	if err != nil {
		t.Fatal(err)
	}
	return found
}

// Pages OCR left out put nothing in the text, so removing them and putting
// them back costs no OCR and leaves the document's status alone.
func TestRemovingUnreadBlankPagesKeepsTheText(t *testing.T) {
	requirePoppler(t)
	app := testpb.Open(t)
	document := makeDocument(t, app, makeUser(t, app), "scan.pdf", testpdf.MixedText(3, 2))
	document = withRecord(t, app, document, models.PageRecord{Skipped: []int{3}})

	outcome, err := RemovePages(context.Background(), app, document, []int{3})
	if err != nil {
		t.Fatal(err)
	}
	if outcome.Reread {
		t.Fatal("removing an unread blank page queued OCR")
	}
	document = reload(t, app, document)
	if len(jobs(t, app, document.Id)) != 0 {
		t.Fatal("a job was queued")
	}
	if got := document.GetString("ocr_text"); got != "old text" {
		t.Fatalf("ocr_text = %q, want it untouched", got)
	}
	if got := document.GetString("processing_status"); got != models.DocStatusCompleted {
		t.Fatalf("status = %q, want it untouched", got)
	}
	if got := len(storedPages(t, app, document, "file")); got != 2 {
		t.Fatalf("file has %d pages, want 2", got)
	}
	if document.GetString("preview") == "" {
		t.Fatal("no preview for the new file")
	}
	if got := models.LoadPageRecord(document).Map; !slices.Equal(got, []int{1, 2}) {
		t.Fatalf("page map = %v, want [1 2]", got)
	}

	outcome, err = RestoreOriginal(context.Background(), app, document)
	if err != nil {
		t.Fatal(err)
	}
	if outcome.Reread || len(jobs(t, app, document.Id)) != 0 {
		t.Fatal("restoring an unread blank page queued OCR")
	}
	document = reload(t, app, document)
	if got := len(storedPages(t, app, document, "file")); got != 3 {
		t.Fatalf("restored file has %d pages, want 3", got)
	}
	record := models.LoadPageRecord(document)
	if len(record.Map) != 0 || !slices.Equal(record.Skipped, []int{3}) || document.GetString("original_file") != "" {
		t.Fatalf("after restore: record %+v, original %q", record, document.GetString("original_file"))
	}
}

func TestRemovingAReadPageRereads(t *testing.T) {
	requirePoppler(t)
	app := testpb.Open(t)
	document := makeDocument(t, app, makeUser(t, app), "scan.pdf", testpdf.Multipage(4))

	outcome, err := RemovePages(context.Background(), app, document, []int{2, 4})
	if err != nil {
		t.Fatal(err)
	}
	if !outcome.Reread {
		t.Fatal("removing read pages did not queue OCR")
	}
	document = reload(t, app, document)
	if document.GetString("processing_status") != models.DocStatusPending {
		t.Fatalf("status = %q, want pending", document.GetString("processing_status"))
	}
	queued := jobs(t, app, document.Id)
	var steps, force []string
	if err := queued[0].UnmarshalJSONField("steps", &steps); err != nil {
		t.Fatal(err)
	}
	if err := queued[0].UnmarshalJSONField("force_steps", &force); err != nil {
		t.Fatal(err)
	}
	if slices.Contains(steps, models.StepApplyMetadata) || !slices.Equal(force, []string{models.StepOCR}) {
		t.Fatalf("steps %v forcing %v; want OCR forced and no metadata overwrite", steps, force)
	}
	pages := storedPages(t, app, document, "file")
	if len(pages) != 2 || !strings.Contains(pages[1], "Page 3") {
		t.Fatalf("pages = %q, want Page 1 and Page 3", pages)
	}

	if _, err := RemovePages(context.Background(), app, document, []int{1}); !errors.Is(err, ErrBusy) {
		t.Fatalf("removal during processing = %v, want ErrBusy", err)
	}
	finishJobs(t, app, document.Id)

	// Page 1 of the shorter file is page 1 of the upload, and the second
	// removal keeps the first backup.
	if _, err := RemovePages(context.Background(), app, document, []int{1}); err != nil {
		t.Fatal(err)
	}
	document = reload(t, app, document)
	if got := models.LoadPageRecord(document).Map; !slices.Equal(got, []int{3}) {
		t.Fatalf("page map = %v, want [3]", got)
	}
	if got := len(storedPages(t, app, document, "original_file")); got != 4 {
		t.Fatalf("backup has %d pages, want 4", got)
	}
	finishJobs(t, app, document.Id)

	outcome, err = RestoreOriginal(context.Background(), app, document)
	if err != nil {
		t.Fatal(err)
	}
	if !outcome.Reread {
		t.Fatal("restoring pages that were read did not queue OCR")
	}
	finishJobs(t, app, document.Id)
	if _, err := RestoreOriginal(context.Background(), app, reload(t, app, document)); !errors.Is(err, ErrNoOriginal) {
		t.Fatalf("second restore = %v, want ErrNoOriginal", err)
	}
}

// Keeping a page OCR left out says it has content, so it is read from then on.
func TestKeepingAnUnreadPageConfirmsIt(t *testing.T) {
	requirePoppler(t)
	app := testpb.Open(t)
	document := makeDocument(t, app, makeUser(t, app), "scan.pdf", testpdf.MixedText(3, 2))
	fileName := document.GetString("file")

	if _, err := RemovePages(context.Background(), app, document, nil); !errors.Is(err, ErrNoPages) {
		t.Fatalf("an empty edit with nothing unread = %v, want ErrNoPages", err)
	}

	document = withRecord(t, app, document, models.PageRecord{Skipped: []int{3}})
	outcome, err := RemovePages(context.Background(), app, document, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !outcome.Reread {
		t.Fatal("keeping an unread page did not queue OCR")
	}
	document = reload(t, app, document)
	record := models.LoadPageRecord(document)
	// Skipped stays true to the text until the queued OCR reads page 3.
	if !slices.Equal(record.Confirmed, []int{3}) || !slices.Equal(record.Skipped, []int{3}) {
		t.Fatalf("record = %+v, want page 3 confirmed and still skipped", record)
	}
	if document.GetString("file") != fileName {
		t.Fatal("keeping every page replaced the file")
	}

	finishJobs(t, app, document.Id)
	overview, err := Inspect(context.Background(), app, document)
	if err != nil {
		t.Fatal(err)
	}
	if overview.Pages[2].Blank {
		t.Fatal("a confirmed page is still suggested as blank")
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
		"every page":   {[]int{1, 2}, ErrAllPages},
		"a page past":  {[]int{3}, ErrPageOutOfDoc},
		"page zero":    {[]int{0}, ErrPageOutOfDoc},
		"repeated all": {[]int{1, 1, 2}, ErrAllPages},
	} {
		if _, err := RemovePages(context.Background(), app, document, tc.pages); !errors.Is(err, tc.want) {
			t.Errorf("%s: err = %v, want %v", name, err, tc.want)
		}
	}

	text := makeDocument(t, app, owner, "note.txt", []byte("plain text"))
	if _, err := RemovePages(context.Background(), app, text, []int{1}); !errors.Is(err, ErrNotPDF) {
		t.Fatalf("text document: err = %v, want ErrNotPDF", err)
	}
	if _, err := Inspect(context.Background(), app, text); !errors.Is(err, ErrNotPDF) {
		t.Fatalf("inspect text document: err = %v, want ErrNotPDF", err)
	}
}

func TestPlanRemoval(t *testing.T) {
	// The file holds uploaded pages 1, 2, 4 and 6; OCR left out 2 and 6.
	record := models.PageRecord{Map: []int{1, 2, 4, 6}, Skipped: []int{2, 6}}
	for name, tc := range map[string]struct {
		removed []int
		kept    []int
		confirm []int
		reread  bool
	}{
		"only unread pages":       {[]int{2, 4}, []int{1, 4}, nil, false},
		"a read page":             {[]int{2, 3, 4}, []int{1}, nil, true},
		"keeping an unread page":  {[]int{4}, []int{1, 2, 4}, []int{2}, true},
		"keeping every page":      {nil, []int{1, 2, 4, 6}, []int{2, 6}, true},
		"a read page and the left-outs": {[]int{1, 2, 4}, []int{4}, nil, true},
	} {
		plan := planRemoval(record, 4, tc.removed)
		if !slices.Equal(plan.kept, tc.kept) || !slices.Equal(plan.confirm, tc.confirm) || plan.reread != tc.reread {
			t.Errorf("%s: plan = %+v, want kept %v confirm %v reread %v", name, plan, tc.kept, tc.confirm, tc.reread)
		}
	}

	// Page 3 was read before it was found blank. Keeping it ends the
	// suggestion; the text already has it, so nothing is read again.
	plan := planRemoval(models.PageRecord{Blank: []int{3}}, 3, nil)
	if !slices.Equal(plan.confirm, []int{3}) || plan.reread {
		t.Fatalf("keeping a read blank page: plan = %+v, want it confirmed without rereading", plan)
	}
}

func TestRestoreNeedsReread(t *testing.T) {
	if restoreNeedsReread(models.PageRecord{Map: []int{1, 3}, Skipped: []int{2, 4}}, 4) {
		t.Fatal("bringing back only unread pages asked for OCR")
	}
	if !restoreNeedsReread(models.PageRecord{Map: []int{1, 3}, Skipped: []int{2}}, 4) {
		t.Fatal("bringing back a read page did not ask for OCR")
	}
	// A backup with no page map comes from before pages were numbered.
	if !restoreNeedsReread(models.PageRecord{}, 4) {
		t.Fatal("an unknown page map did not ask for OCR")
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
