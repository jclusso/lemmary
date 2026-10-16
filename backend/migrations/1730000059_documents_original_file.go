package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"

	"lemmary/backend/internal/models"
)

// Adds documents.original_file, the as-uploaded PDF kept when pages are removed
// so the removal can be undone.
//
// Hidden so a regular account cannot write it through the records API: only
// internal/pageedit fills it, from the file the document already had, and only
// pageedit puts it back.
func init() {
	m.Register(func(app core.App) error {
		documents, err := app.FindCollectionByNameOrId("documents")
		if err != nil {
			return err
		}
		if documents.Fields.GetByName("original_file") == nil {
			documents.Fields.Add(&core.FileField{
				Name:      "original_file",
				MaxSelect: 1,
				MaxSize:   models.MaxFileBytes,
				MimeTypes: []string{"application/pdf"},
				Hidden:    true,
			})
		}
		return app.Save(documents)
	}, func(app core.App) error {
		documents, err := app.FindCollectionByNameOrId("documents")
		if err != nil {
			return nil
		}
		if f := documents.Fields.GetByName("original_file"); f != nil {
			documents.Fields.RemoveById(f.GetId())
		}
		return app.Save(documents)
	})
}
