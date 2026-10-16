package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Adds documents.sensitive, the owner's mark that a document must never reach
// an external AI provider, and sensitive_folders, the consume-folder names whose
// files arrive marked. The flag is owner-writable on purpose: the upload form
// sets it in the same request that creates the document, before any job exists.
func init() {
	m.Register(addSensitiveDocuments, dropSensitiveDocuments)
}

const defaultSensitiveFolders = "sensitive"

func addSensitiveDocuments(app core.App) error {
	documents, err := app.FindCollectionByNameOrId("documents")
	if err != nil {
		return err
	}
	if documents.Fields.GetByName("sensitive") == nil {
		documents.Fields.Add(&core.BoolField{Name: "sensitive"})
		if err := app.Save(documents); err != nil {
			return err
		}
	}

	settings, err := app.FindCollectionByNameOrId("app_settings")
	if err != nil {
		return err
	}
	seedFolders := settings.Fields.GetByName("sensitive_folders") == nil
	if seedFolders {
		settings.Fields.Add(&core.TextField{Name: "sensitive_folders", Max: 500})
	}
	if err := app.Save(settings); err != nil {
		return err
	}
	if !seedFolders {
		return nil
	}
	// An install that predates the field gets the same default a fresh one is
	// seeded with; once the field exists an admin's empty value is kept.
	record, err := app.FindRecordById("app_settings", "appsettings0001")
	if err != nil {
		return nil
	}
	record.Set("sensitive_folders", defaultSensitiveFolders)
	return app.Save(record)
}

func dropSensitiveDocuments(app core.App) error {
	if documents, err := app.FindCollectionByNameOrId("documents"); err == nil && documents.Fields.GetByName("sensitive") != nil {
		documents.Fields.RemoveByName("sensitive")
		if err := app.Save(documents); err != nil {
			return err
		}
	}
	settings, err := app.FindCollectionByNameOrId("app_settings")
	if err != nil {
		return nil
	}
	settings.Fields.RemoveByName("sensitive_folders")
	return app.Save(settings)
}
