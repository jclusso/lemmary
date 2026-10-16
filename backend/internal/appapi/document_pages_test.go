package appapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"

	"lemmary/backend/internal/models"
	"lemmary/backend/internal/pageedit"
	"lemmary/backend/internal/pdftool/testpdf"
)

func callDocumentPages(t *testing.T, app core.App, handler func(core.App) func(*core.RequestEvent) error,
	method, userID, documentID, body string) (int, string) {
	t.Helper()
	user, err := app.FindRecordById("users", userID)
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	e := &core.RequestEvent{}
	e.Response = rec
	e.Request = httptest.NewRequest(method, "/api/app/documents/"+documentID+"/pages", strings.NewReader(body))
	e.Request.SetPathValue("documentId", documentID)
	e.Auth = user
	if err := handler(app)(e); err != nil {
		t.Fatalf("handler: %v", err)
	}
	return rec.Code, rec.Body.String()
}

func makePDFDocument(t *testing.T, app core.App, ownerID string, data []byte) *core.Record {
	t.Helper()
	document := makeQueueDocument(t, app, ownerID, models.DocStatusCompleted, "text")
	file, err := filesystem.NewFileFromBytes(data, "scan.pdf")
	if err != nil {
		t.Fatal(err)
	}
	document.Set("file", file)
	if err := app.Save(document); err != nil {
		t.Fatal(err)
	}
	return document
}

func TestDocumentPagesRemoveAndRestore(t *testing.T) {
	for _, binary := range []string{"pdfinfo", "pdftoppm", "pdfseparate", "pdfunite"} {
		if _, err := exec.LookPath(binary); err != nil {
			t.Skipf("%s not installed", binary)
		}
	}
	app := bootQueueApp(t)
	owner := makeQueueUser(t, app, "owner@example.com")
	stranger := makeQueueUser(t, app, "stranger@example.com")
	doc := makePDFDocument(t, app, owner, testpdf.MixedText(3, 2))

	if code, _ := callDocumentPages(t, app, handleGetDocumentPages, http.MethodGet, stranger, doc.Id, ""); code != http.StatusNotFound {
		t.Fatalf("stranger GET = %d, want 404", code)
	}
	code, body := callDocumentPages(t, app, handleGetDocumentPages, http.MethodGet, owner, doc.Id, "")
	var overview pageedit.Overview
	if err := json.Unmarshal([]byte(body), &overview); err != nil || code != http.StatusOK {
		t.Fatalf("owner GET = %d %s", code, body)
	}
	if len(overview.Pages) != 3 || !overview.Pages[2].Blank || overview.Pages[0].Blank {
		t.Fatalf("pages = %+v, want three with only the last blank", overview.Pages)
	}

	if code, _ := callDocumentPages(t, app, handlePostDocumentPageRemoval, http.MethodPost, stranger, doc.Id, `{"pages":[3]}`); code != http.StatusNotFound {
		t.Fatalf("stranger removal = %d, want 404", code)
	}
	if code, body := callDocumentPages(t, app, handlePostDocumentPageRemoval, http.MethodPost, owner, doc.Id, `{"pages":[1,2,3]}`); code != http.StatusBadRequest {
		t.Fatalf("removing every page = %d %s, want 400", code, body)
	}
	if code, body := callDocumentPages(t, app, handlePostDocumentPageRemoval, http.MethodPost, owner, doc.Id, `{"pages":[3]}`); code != http.StatusOK {
		t.Fatalf("owner removal = %d %s", code, body)
	}
	if code, _ := callDocumentPages(t, app, handlePostDocumentPageRemoval, http.MethodPost, owner, doc.Id, `{"pages":[2]}`); code != http.StatusConflict {
		t.Fatalf("removal while queued = %d, want 409", code)
	}

	jobs, err := app.FindRecordsByFilter("processing_jobs", "document = {:doc}", "", 0, 0, map[string]any{"doc": doc.Id})
	if err != nil || len(jobs) != 1 {
		t.Fatalf("jobs = %d (%v), want 1", len(jobs), err)
	}
	jobs[0].Set("finished_at", "2026-01-01 00:00:00.000Z")
	if err := app.Save(jobs[0]); err != nil {
		t.Fatal(err)
	}

	code, body = callDocumentPages(t, app, handleGetDocumentPages, http.MethodGet, owner, doc.Id, "")
	if err := json.Unmarshal([]byte(body), &overview); err != nil || code != http.StatusOK {
		t.Fatalf("GET after removal = %d %s", code, body)
	}
	if len(overview.Pages) != 2 || !overview.HasOriginal {
		t.Fatalf("after removal: %d pages, has_original %v; want 2 and true", len(overview.Pages), overview.HasOriginal)
	}

	if code, body := callDocumentPages(t, app, handlePostDocumentOriginalRestore, http.MethodPost, owner, doc.Id, ""); code != http.StatusOK {
		t.Fatalf("restore = %d %s", code, body)
	}
}

func TestDocumentPagesRefusesATextDocument(t *testing.T) {
	app := bootQueueApp(t)
	owner := makeQueueUser(t, app, "owner@example.com")
	doc := makeQueueDocument(t, app, owner, models.DocStatusCompleted, "text")

	if code, body := callDocumentPages(t, app, handleGetDocumentPages, http.MethodGet, owner, doc.Id, ""); code != http.StatusBadRequest {
		t.Fatalf("GET text document = %d %s, want 400", code, body)
	}
	if code, body := callDocumentPages(t, app, handlePostDocumentOriginalRestore, http.MethodPost, owner, doc.Id, ""); code != http.StatusBadRequest {
		t.Fatalf("restore with no original = %d %s, want 400", code, body)
	}
	if code, _ := callDocumentPages(t, app, handlePostDocumentPageRemoval, http.MethodPost, owner, doc.Id, "not json"); code != http.StatusBadRequest {
		t.Fatalf("bad body = %d, want 400", code)
	}
}

// The page record decides what OCR reads and whether an edit costs OCR again,
// so an owner patching it through the records API must not move it.
func TestRecordsAPIKeepsThePageRecord(t *testing.T) {
	app := bootQueueApp(t)
	bindPageRecordGuard(app)
	owner := makeQueueUser(t, app, "owner@example.com")
	doc := makeQueueDocument(t, app, owner, models.DocStatusCompleted, "text")
	models.PageRecord{Skipped: []int{2}}.Store(doc)
	if err := app.Save(doc); err != nil {
		t.Fatal(err)
	}

	record, err := app.FindRecordById("documents", doc.Id)
	if err != nil {
		t.Fatal(err)
	}
	record.Set(models.OCRSkippedPagesField, []int{1, 2, 3})
	e := &core.RecordRequestEvent{RequestEvent: &core.RequestEvent{App: app}, Record: record}
	e.Collection = record.Collection()
	e.Response = httptest.NewRecorder()
	e.Request = httptest.NewRequest(http.MethodPatch, "/api/collections/documents/records/"+doc.Id, nil)
	if err := app.OnRecordUpdateRequest().Trigger(e, func(e *core.RecordRequestEvent) error {
		return e.App.Save(e.Record)
	}); err != nil {
		t.Fatal(err)
	}

	fresh, err := app.FindRecordById("documents", doc.Id)
	if err != nil {
		t.Fatal(err)
	}
	if got := models.LoadPageRecord(fresh).Skipped; len(got) != 1 || got[0] != 2 {
		t.Fatalf("skipped = %v after a records API update, want [2]", got)
	}
}
