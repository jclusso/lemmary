package migrations

import (
	"testing"

	"github.com/pocketbase/pocketbase/core"
)

func TestNormalizeResultLanguage(t *testing.T) {
	cases := map[string]string{"en": "en", "EN-us": "en", "english": "", "": ""}
	for raw, want := range cases {
		app := bootMigratedApp(t)
		collection, err := app.FindCollectionByNameOrId("app_settings")
		if err != nil {
			t.Fatalf("app_settings collection: %v", err)
		}
		settings := core.NewRecord(collection)
		settings.Set("processing_result_language", raw)
		if err := app.Save(settings); err != nil {
			t.Fatalf("save app_settings: %v", err)
		}

		if err := normalizeLanguageSettings(app); err != nil {
			t.Fatalf("normalizeLanguageSettings: %v", err)
		}
		saved, err := app.FindRecordById("app_settings", settings.Id)
		if err != nil {
			t.Fatalf("reload app_settings: %v", err)
		}
		if got := saved.GetString("processing_result_language"); got != want {
			t.Fatalf("%q normalized to %q, want %q", raw, got, want)
		}
	}
}

func TestNormalizeDeepSearchLanguages(t *testing.T) {
	app := bootMigratedApp(t)
	collection, err := app.FindCollectionByNameOrId("app_settings")
	if err != nil {
		t.Fatalf("app_settings collection: %v", err)
	}
	settings := core.NewRecord(collection)
	settings.Set("deep_search_languages", "DE, english,uk,de")
	if err := app.Save(settings); err != nil {
		t.Fatalf("save app_settings: %v", err)
	}

	if err := normalizeLanguageSettings(app); err != nil {
		t.Fatalf("normalizeLanguageSettings: %v", err)
	}
	saved, err := app.FindRecordById("app_settings", settings.Id)
	if err != nil {
		t.Fatalf("reload app_settings: %v", err)
	}
	if got := saved.GetString("deep_search_languages"); got != "de,uk" {
		t.Fatalf("deep_search_languages = %q, want de,uk", got)
	}
}

func TestDocumentLanguageMigrationIsIdempotent(t *testing.T) {
	app := bootMigratedApp(t)

	if err := addDocumentLanguageField(app); err != nil {
		t.Fatalf("addDocumentLanguageField on an already-migrated app: %v", err)
	}
	documents, err := app.FindCollectionByNameOrId("documents")
	if err != nil {
		t.Fatalf("documents collection: %v", err)
	}
	field, ok := documents.Fields.GetByName("language").(*core.TextField)
	if !ok {
		t.Fatal("documents has no language text field")
	}
	for _, bad := range []string{"EN", "e1", "eng"} {
		if field.ValidatePlainValue(bad) == nil {
			t.Fatalf("language %q must be refused", bad)
		}
	}
	if err := field.ValidatePlainValue("en"); err != nil {
		t.Fatalf("language en refused: %v", err)
	}
}
