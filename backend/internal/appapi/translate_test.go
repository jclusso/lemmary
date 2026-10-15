package appapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pocketbase/pocketbase/core"

	"lemmary/backend/internal/ai"
	"lemmary/backend/internal/config"
	"lemmary/backend/internal/models"
)

type fakeTranslator struct {
	calls int
	// during runs while the model is "answering", to stage a concurrent edit.
	during func()
}

func (f *fakeTranslator) Name() string  { return "fake" }
func (f *fakeTranslator) Model() string { return "fake-model" }
func (f *fakeTranslator) ExtractMetadata(context.Context, string, ai.ExtractionCatalog) (*models.ExtractedMetadata, error) {
	return nil, nil
}
func (f *fakeTranslator) Translate(_ context.Context, text string) (string, error) {
	f.calls++
	if f.during != nil {
		f.during()
	}
	return "DE: " + text, nil
}

func callTranslation(t *testing.T, app core.App, tr *fakeTranslator, userID, documentID, query string) (int, string) {
	t.Helper()
	user, err := app.FindRecordById("users", userID)
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	e := &core.RequestEvent{}
	e.Response = rec
	e.Request = httptest.NewRequest(http.MethodPost, "/api/app/documents/"+documentID+"/translation"+query, nil)
	e.Request.SetPathValue("documentId", documentID)
	e.Auth = user
	snapshot := func() config.Snapshot {
		return config.Snapshot{Cfg: config.Config{ProcessingResultLanguage: "de"}, AI: tr}
	}
	if err := handleDocumentTranslation(app, snapshot)(e); err != nil {
		t.Fatalf("handler: %v", err)
	}
	var body struct{ Text string }
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	return rec.Code, body.Text
}

func TestDocumentTranslationStoresAndReusesTheTranslation(t *testing.T) {
	app := bootQueueApp(t)
	app.OnRecordUpdate("documents").BindFunc(clearStaleTranslation)
	owner := makeQueueUser(t, app, "owner@example.com")
	stranger := makeQueueUser(t, app, "stranger@example.com")
	doc := makeQueueDocument(t, app, owner, "completed", "Rechnung")
	tr := &fakeTranslator{}

	if code, _ := callTranslation(t, app, tr, stranger, doc.Id, ""); code != http.StatusForbidden {
		t.Fatalf("stranger status = %d, want 403", code)
	}
	if code, text := callTranslation(t, app, tr, owner, doc.Id, ""); code != http.StatusOK || text != "DE: Rechnung" {
		t.Fatalf("first call = %d %q", code, text)
	}
	if _, text := callTranslation(t, app, tr, owner, doc.Id, ""); text != "DE: Rechnung" || tr.calls != 1 {
		t.Fatalf("second call = %q after %d model calls, want the stored text after 1", text, tr.calls)
	}
	if _, _ = callTranslation(t, app, tr, owner, doc.Id, "?force=1"); tr.calls != 2 {
		t.Fatalf("force made %d model calls, want 2", tr.calls)
	}

	fresh, err := app.FindRecordById("documents", doc.Id)
	if err != nil {
		t.Fatal(err)
	}
	fresh.Set("ocr_text", "Mahnung")
	if err := app.Save(fresh); err != nil {
		t.Fatal(err)
	}
	if fresh, _ = app.FindRecordById("documents", doc.Id); fresh.GetString("ocr_text_translated") != "" {
		t.Fatalf("editing ocr_text should clear the translation, got %q", fresh.GetString("ocr_text_translated"))
	}
	if _, text := callTranslation(t, app, tr, owner, doc.Id, ""); text != "DE: Mahnung" {
		t.Fatalf("after edit = %q", text)
	}
}

func TestDocumentTranslationRefusesASensitiveDocument(t *testing.T) {
	app := bootQueueApp(t)
	owner := makeQueueUser(t, app, "owner@example.com")
	doc := makeQueueDocument(t, app, owner, "completed", "Lohnsteuerbescheinigung")
	doc.Set("sensitive", true)
	if err := app.Save(doc); err != nil {
		t.Fatal(err)
	}
	tr := &fakeTranslator{}
	for _, query := range []string{"", "?force=1"} {
		if code, _ := callTranslation(t, app, tr, owner, doc.Id, query); code != http.StatusForbidden {
			t.Fatalf("status %q = %d, want 403", query, code)
		}
	}
	if tr.calls != 0 {
		t.Fatalf("the model was called %d times for a sensitive document", tr.calls)
	}
}

func TestDocumentChatRefusesASensitiveDocument(t *testing.T) {
	app := bootQueueApp(t)
	ownerID := makeQueueUser(t, app, "owner@example.com")
	owner, err := app.FindRecordById("users", ownerID)
	if err != nil {
		t.Fatal(err)
	}
	doc := makeQueueDocument(t, app, ownerID, "completed", "wages")
	doc.Set("sensitive", true)
	if err := app.Save(doc); err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	e := &core.RequestEvent{}
	e.Response = rec
	e.Request = httptest.NewRequest(http.MethodPost, "/api/app/documents/"+doc.Id+"/chat", nil)
	e.Request.SetPathValue("documentId", doc.Id)
	e.Auth = owner
	_, text, handled, err := loadChatDocument(app, e)
	if err != nil {
		t.Fatal(err)
	}
	if !handled || text != "" || rec.Code != http.StatusForbidden {
		t.Fatalf("handled=%v text=%q status=%d, want a 403 and no text for the model", handled, text, rec.Code)
	}
}

func TestDocumentTranslationKeepsAnEditMadeWhileTranslating(t *testing.T) {
	app := bootQueueApp(t)
	app.OnRecordUpdate("documents").BindFunc(clearStaleTranslation)
	owner := makeQueueUser(t, app, "owner@example.com")
	doc := makeQueueDocument(t, app, owner, "completed", "Rechnung")
	tr := &fakeTranslator{during: func() {
		edited, err := app.FindRecordById("documents", doc.Id)
		if err != nil {
			t.Fatal(err)
		}
		edited.Set("ocr_text", "Rechnung korrigiert")
		if err := app.Save(edited); err != nil {
			t.Fatal(err)
		}
	}}

	if code, _ := callTranslation(t, app, tr, owner, doc.Id, ""); code != http.StatusConflict {
		t.Fatalf("status = %d, want 409", code)
	}
	fresh, err := app.FindRecordById("documents", doc.Id)
	if err != nil {
		t.Fatal(err)
	}
	if fresh.GetString("ocr_text") != "Rechnung korrigiert" || fresh.GetString("ocr_text_translated") != "" {
		t.Fatalf("edit lost or stale translation stored: ocr_text=%q translated=%q",
			fresh.GetString("ocr_text"), fresh.GetString("ocr_text_translated"))
	}
}
