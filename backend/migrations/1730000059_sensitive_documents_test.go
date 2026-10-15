package migrations

import (
	"testing"

	"github.com/pocketbase/pocketbase/core"

	"lemmary/backend/internal/config"
)

func TestSensitiveDocumentsFieldIsOwnerWritable(t *testing.T) {
	app := bootMigratedApp(t)
	for range 2 {
		if err := addSensitiveDocuments(app); err != nil {
			t.Fatalf("add on a migrated app: %v", err)
		}
	}

	documents, err := app.FindCollectionByNameOrId("documents")
	if err != nil {
		t.Fatal(err)
	}
	field, ok := documents.Fields.GetByName("sensitive").(*core.BoolField)
	if !ok || field.Hidden {
		t.Fatalf("documents.sensitive = %#v, want a visible bool the upload form can set", field)
	}

	userID := makeUser(t, app, "sensitive@example.test")
	doc, err := app.FindRecordById("documents", makeDocument(t, app, userID, "W-2"))
	if err != nil {
		t.Fatal(err)
	}
	if doc.GetBool("sensitive") {
		t.Fatal("a document must not start out sensitive")
	}

	for range 2 {
		if err := dropSensitiveDocuments(app); err != nil {
			t.Fatalf("drop: %v", err)
		}
	}
	if err := addSensitiveDocuments(app); err != nil {
		t.Fatalf("re-add: %v", err)
	}
}

// The default reaches an install that predates the field; an admin who later
// empties the list keeps it empty, because the seed only runs with the field.
func TestSensitiveFoldersSeedOnlyWhenTheFieldIsAdded(t *testing.T) {
	app := bootMigratedApp(t)
	if err := dropSensitiveDocuments(app); err != nil {
		t.Fatal(err)
	}
	collection, err := app.FindCollectionByNameOrId("app_settings")
	if err != nil {
		t.Fatal(err)
	}
	settings := core.NewRecord(collection)
	settings.Id = config.SingletonID
	settings.MarkAsNew()
	if err := app.Save(settings); err != nil {
		t.Fatal(err)
	}

	if err := addSensitiveDocuments(app); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(app)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.SensitiveFolders) != 1 || cfg.SensitiveFolders[0] != config.DefaultSensitiveFolder {
		t.Fatalf("SensitiveFolders = %q, want the default on an upgraded install", cfg.SensitiveFolders)
	}

	settings, err = app.FindRecordById("app_settings", config.SingletonID)
	if err != nil {
		t.Fatal(err)
	}
	settings.Set("sensitive_folders", "")
	if err := app.Save(settings); err != nil {
		t.Fatal(err)
	}
	if err := addSensitiveDocuments(app); err != nil {
		t.Fatal(err)
	}
	if cfg, _ := config.Load(app); len(cfg.SensitiveFolders) != 0 {
		t.Fatalf("SensitiveFolders = %q, want the admin's empty list kept", cfg.SensitiveFolders)
	}
}
