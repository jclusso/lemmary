package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Adds app_settings.blank_page_ai_check_enabled, off unless
// BLANK_PAGE_AI_CHECK_ENABLED seeds it, because it spends tokens.
func init() {
	m.Register(func(app core.App) error {
		settings, err := app.FindCollectionByNameOrId("app_settings")
		if err != nil {
			return err
		}
		if settings.Fields.GetByName("blank_page_ai_check_enabled") == nil {
			settings.Fields.Add(&core.BoolField{Name: "blank_page_ai_check_enabled"})
		}
		return app.Save(settings)
	}, func(app core.App) error {
		settings, err := app.FindCollectionByNameOrId("app_settings")
		if err != nil {
			return nil
		}
		if f := settings.Fields.GetByName("blank_page_ai_check_enabled"); f != nil {
			settings.Fields.RemoveById(f.GetId())
		}
		return app.Save(settings)
	})
}
