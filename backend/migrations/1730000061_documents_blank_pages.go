package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"

	"lemmary/backend/internal/models"
)

// Adds documents.blank_pages, what the detect_blank_pages step found, kept
// apart from ocr_skipped_pages because a document read before the step ran
// has text from pages now known to be blank. Guarded like the rest of the
// page record (see 1730000060).
func init() {
	m.Register(func(app core.App) error {
		documents, err := app.FindCollectionByNameOrId("documents")
		if err != nil {
			return err
		}
		if documents.Fields.GetByName(models.BlankPagesField) == nil {
			documents.Fields.Add(&core.JSONField{Name: models.BlankPagesField, MaxSize: 64 << 10})
		}
		return app.Save(documents)
	}, func(app core.App) error {
		documents, err := app.FindCollectionByNameOrId("documents")
		if err != nil {
			return nil
		}
		if f := documents.Fields.GetByName(models.BlankPagesField); f != nil {
			documents.Fields.RemoveById(f.GetId())
		}
		return app.Save(documents)
	})
}
