package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"

	"lemmary/backend/internal/langcode"
)

// Adds language, the ISO 639-1 code extraction reads off each document, so the
// document page can tell a document already in the result language from one
// worth translating. The result and deep search language settings were free
// text until now; a value that names a language becomes its code, and one that
// does not is dropped, because the settings form now offers only codes and
// would refuse to save it back.
func init() {
	m.Register(func(app core.App) error {
		if err := addDocumentLanguageField(app); err != nil {
			return err
		}
		return normalizeLanguageSettings(app)
	}, func(app core.App) error {
		documents, err := app.FindCollectionByNameOrId("documents")
		if err != nil {
			return nil
		}
		documents.Fields.RemoveByName("language")
		return app.Save(documents)
	})
}

func addDocumentLanguageField(app core.App) error {
	documents, err := app.FindCollectionByNameOrId("documents")
	if err != nil {
		return err
	}
	if documents.Fields.GetByName("language") != nil {
		return nil
	}
	documents.Fields.Add(&core.TextField{Name: "language", Max: 2, Pattern: `^[a-z]{2}$`})
	return app.Save(documents)
}

func normalizeLanguageSettings(app core.App) error {
	records, err := app.FindAllRecords("app_settings")
	if err != nil {
		return err
	}
	for _, record := range records {
		raw := record.GetString("processing_result_language")
		code, _ := langcode.Parse(raw)
		rawList := record.GetString("deep_search_languages")
		codes, _ := langcode.ParseList(rawList)
		if code == raw && codes == rawList {
			continue
		}
		record.Set("processing_result_language", code)
		record.Set("deep_search_languages", codes)
		if err := app.Save(record); err != nil {
			return err
		}
	}
	return nil
}
