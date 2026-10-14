package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

const (
	usersThemeField = "theme"
	usersModeField  = "mode"
)

// Hidden so the users collection API neither shows nor accepts it: the only
// writer is PATCH /api/app/me, which checks the values against the known themes and modes.
func init() {
	m.Register(func(app core.App) error {
		users, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return err
		}
		for _, name := range []string{usersThemeField, usersModeField} {
			if users.Fields.GetByName(name) == nil {
				users.Fields.Add(&core.TextField{Name: name, Hidden: true, Max: 32})
			}
		}
		return app.Save(users)
	}, func(app core.App) error {
		users, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return err
		}
		for _, name := range []string{usersThemeField, usersModeField} {
			if f := users.Fields.GetByName(name); f != nil {
				users.Fields.RemoveById(f.GetId())
			}
		}
		return app.Save(users)
	})
}
