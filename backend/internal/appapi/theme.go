package appapi

import (
	"net/http"
	"slices"

	"github.com/pocketbase/pocketbase/core"

	"lemmary/backend/internal/i18n"
)

const (
	usersThemeField = "theme"
	usersModeField  = "mode"
)

// The same lists as `families` and `modes` in frontend/src/lib/theme.ts.
var (
	themeFamilies = []string{"paper", "neutral", "nord", "catppuccin", "rosepine", "monokai"}
	themeModes    = []string{"system", "light", "dark"}
)

// Pointers, so a request can change one of the two and leave the other alone.
type patchMeRequest struct {
	Theme *string `json:"theme"`
	Mode  *string `json:"mode"`
}

type accountAppearance struct {
	Theme string `json:"theme"`
	Mode  string `json:"mode"`
}

// accountTheme is empty for an account that never chose, and for a superuser
// with no paired users record: the browser keeps its own choice then.
func accountTheme(app core.App, e *core.RequestEvent) accountAppearance {
	userID, err := resolveOwnerUserID(app, e)
	if err != nil {
		return accountAppearance{}
	}
	record, err := app.FindRecordById("users", userID)
	if err != nil {
		return accountAppearance{}
	}
	return accountAppearance{Theme: record.GetString(usersThemeField), Mode: record.GetString(usersModeField)}
}

func (req patchMeRequest) validate() error {
	if req.Theme != nil && !slices.Contains(themeFamilies, *req.Theme) {
		return i18n.Errorf("unknown theme %q", *req.Theme)
	}
	if req.Mode != nil && !slices.Contains(themeModes, *req.Mode) {
		return i18n.Errorf("unknown theme mode %q", *req.Mode)
	}
	return nil
}

func handlePatchMe(app core.App) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		var req patchMeRequest
		if err := e.BindBody(&req); err != nil {
			return writeError(e, http.StatusBadRequest, "Invalid request body.")
		}
		if err := req.validate(); err != nil {
			return writeBadRequest(e, err)
		}
		userID, err := resolveOwnerUserID(app, e)
		if err != nil {
			return writeOwnerError(e, err)
		}
		record, err := app.FindRecordById("users", userID)
		if err != nil {
			app.Logger().Error("account lookup failed", "user", userID, "error", err)
			return writeError(e, http.StatusInternalServerError, "Failed to load the account.")
		}
		if req.Theme != nil {
			record.Set(usersThemeField, *req.Theme)
		}
		if req.Mode != nil {
			record.Set(usersModeField, *req.Mode)
		}
		if err := app.Save(record); err != nil {
			app.Logger().Error("theme save failed", "user", userID, "error", err)
			return writeError(e, http.StatusInternalServerError, "Failed to save the theme.")
		}
		return writeJSON(e, http.StatusOK, accountAppearance{
			Theme: record.GetString(usersThemeField),
			Mode:  record.GetString(usersModeField),
		})
	}
}
