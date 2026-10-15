package migrations

import (
	"testing"

	"github.com/pocketbase/pocketbase/core"

	"lemmary/backend/internal/models"
)

func TestDocumentsCarryThePageRecord(t *testing.T) {
	app := bootMigratedApp(t)

	documents, err := app.FindCollectionByNameOrId("documents")
	if err != nil {
		t.Fatalf("documents collection: %v", err)
	}
	for _, name := range models.PageRecordFields {
		if _, ok := documents.Fields.GetByName(name).(*core.JSONField); !ok {
			t.Fatalf("documents.%s is missing or not JSON", name)
		}
	}
}
