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

		if err := normalizeResultLanguage(app); err != nil {
			t.Fatalf("normalizeResultLanguage: %v", err)
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

func TestDocumentLanguageMigrationIsIdempotent(t *testing.T) {
	app := bootMigratedApp(t)

	if err := addDocumentLanguageField(app); err != nil {
		t.Fatalf("addDocumentLanguageField on an already-migrated app: %v", err)
	}
	documents, err := app.FindCollectionByNameOrId("documents")
	if err != nil {
		t.Fatalf("documents collection: %v", err)
	}
	if documents.Fields.GetByName("language") == nil {
		t.Fatal("documents has no language field")
	}
}
