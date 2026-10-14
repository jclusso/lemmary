package appapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/core"
)

func callMe(t *testing.T, app core.App, user *core.Record, method, body string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	e := &core.RequestEvent{}
	e.App = app
	e.Response = rec
	e.Request = httptest.NewRequest(method, "/api/app/me", strings.NewReader(body))
	e.Request.Header.Set("Content-Type", "application/json")
	e.Auth = user
	handler := handleGetMe(app)
	if method == http.MethodPatch {
		handler = handlePatchMe(app)
	}
	if err := handler(e); err != nil {
		t.Fatalf("handler: %v", err)
	}
	return rec
}

func meAppearance(t *testing.T, rec *httptest.ResponseRecorder) accountAppearance {
	t.Helper()
	var body accountAppearance
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode %s: %v", rec.Body, err)
	}
	return body
}

func TestAccountThemeIsSavedAndReturnedByMe(t *testing.T) {
	app := bootQueueApp(t)
	user, err := app.FindRecordById("users", makeQueueUser(t, app, "theme@example.com"))
	if err != nil {
		t.Fatal(err)
	}

	rec := callMe(t, app, user, http.MethodGet, "")
	if rec.Code != http.StatusOK || meAppearance(t, rec) != (accountAppearance{}) {
		t.Fatalf("before choosing = %d %s, want no theme or mode", rec.Code, rec.Body)
	}

	rec = callMe(t, app, user, http.MethodPatch, `{"theme":"nord","mode":"system"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("patch = %d %s", rec.Code, rec.Body)
	}

	rec = callMe(t, app, user, http.MethodPatch, `{"mode":"dark"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("mode-only patch = %d %s", rec.Code, rec.Body)
	}

	rec = callMe(t, app, user, http.MethodGet, "")
	if got := meAppearance(t, rec); got != (accountAppearance{Theme: "nord", Mode: "dark"}) {
		t.Fatalf("after save = %+v, want nord in dark mode", got)
	}
}

func TestAccountThemeRejectsAnUnknownThemeOrMode(t *testing.T) {
	app := bootQueueApp(t)
	user, err := app.FindRecordById("users", makeQueueUser(t, app, "theme@example.com"))
	if err != nil {
		t.Fatal(err)
	}

	for _, body := range []string{`{"theme":"neon"}`, `{"theme":"nord","mode":"dim"}`} {
		rec := callMe(t, app, user, http.MethodPatch, body)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("patch %s = %d %s, want 400", body, rec.Code, rec.Body)
		}
	}
	saved, err := app.FindRecordById("users", user.Id)
	if err != nil {
		t.Fatal(err)
	}
	if got := saved.GetString(usersThemeField); got != "" {
		t.Fatalf("stored theme = %q, want it left empty", got)
	}
}
