package migrations

import (
	"testing"

	"github.com/pocketbase/pocketbase/core"
)

func TestUsersCarryAHiddenThemeAndMode(t *testing.T) {
	app := bootMigratedApp(t)

	users, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		t.Fatalf("users collection: %v", err)
	}
	for _, name := range []string{usersThemeField, usersModeField} {
		field, ok := users.Fields.GetByName(name).(*core.TextField)
		if !ok {
			t.Fatalf("%s field = %T, want a text field", name, users.Fields.GetByName(name))
		}
		if !field.Hidden {
			t.Fatalf("%s field is visible to the collection API", name)
		}
	}
}
