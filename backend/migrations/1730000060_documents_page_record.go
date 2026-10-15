package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"

	"lemmary/backend/internal/models"
)

// Adds the three columns of models.PageRecord: which uploaded page each page of
// the current file is, which pages looked blank and were left out of OCR, and
// which the owner kept although they looked blank.
//
// Not hidden, because the document page lists the skipped pages from the
// record it already loads. The records API cannot change them: appapi puts
// back what a request carried, as limits does for page_count.
// Named here rather than read from models.PageRecordFields, which grew a
// column in 1730000061 that this migration must not add or drop.
var pageRecordFields1730000060 = []string{models.PageMapField, models.OCRSkippedPagesField, models.ConfirmedPagesField}

func init() {
	m.Register(func(app core.App) error {
		documents, err := app.FindCollectionByNameOrId("documents")
		if err != nil {
			return err
		}
		for _, name := range pageRecordFields1730000060 {
			if documents.Fields.GetByName(name) == nil {
				documents.Fields.Add(&core.JSONField{Name: name, MaxSize: 64 << 10})
			}
		}
		return app.Save(documents)
	}, func(app core.App) error {
		documents, err := app.FindCollectionByNameOrId("documents")
		if err != nil {
			return nil
		}
		for _, name := range pageRecordFields1730000060 {
			if f := documents.Fields.GetByName(name); f != nil {
				documents.Fields.RemoveById(f.GetId())
			}
		}
		return app.Save(documents)
	})
}
