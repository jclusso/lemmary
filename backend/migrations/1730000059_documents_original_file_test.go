package migrations

import (
	"testing"

	"github.com/pocketbase/pocketbase/core"
)

func TestDocumentsOriginalFileIsAHiddenPDFField(t *testing.T) {
	app := bootMigratedApp(t)

	documents, err := app.FindCollectionByNameOrId("documents")
	if err != nil {
		t.Fatalf("documents collection: %v", err)
	}
	field, ok := documents.Fields.GetByName("original_file").(*core.FileField)
	if !ok || field == nil {
		t.Fatal("documents.original_file is missing")
	}
	if !field.Hidden {
		t.Fatal("documents.original_file is not hidden")
	}
	if len(field.MimeTypes) != 1 || field.MimeTypes[0] != "application/pdf" {
		t.Fatalf("mime types = %v, want only application/pdf", field.MimeTypes)
	}
}
