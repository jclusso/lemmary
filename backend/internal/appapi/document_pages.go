package appapi

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/pocketbase/pocketbase/core"
	"lemmary/backend/internal/i18n"
	"lemmary/backend/internal/pageedit"
)

type removePagesRequest struct {
	Pages []int `json:"pages"`
}

// handleGetDocumentPages renders the pages of the caller's own PDF, each with a
// thumbnail and whether it looks blank.
func handleGetDocumentPages(app core.App) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		document, handled, err := loadOwnDocument(app, e)
		if handled {
			return err
		}
		overview, err := pageedit.Inspect(e.Request.Context(), app, document)
		if err != nil {
			return writePageEditError(app, e, err)
		}
		return writeJSON(e, http.StatusOK, overview)
	}
}

// handlePostDocumentPageRemoval replaces the caller's PDF with one lacking the
// listed pages and queues preview and OCR for it.
func handlePostDocumentPageRemoval(app core.App) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		var req removePagesRequest
		if err := json.NewDecoder(e.Request.Body).Decode(&req); err != nil {
			return writeError(e, http.StatusBadRequest, "Invalid request body.")
		}
		document, handled, err := loadOwnDocument(app, e)
		if handled {
			return err
		}
		if err := pageedit.RemovePages(e.Request.Context(), app, document, req.Pages); err != nil {
			return writePageEditError(app, e, err)
		}
		return writeJSON(e, http.StatusOK, map[string]string{"status": "queued"})
	}
}

// handlePostDocumentOriginalRestore puts back the file as it was before any
// page was removed.
func handlePostDocumentOriginalRestore(app core.App) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		document, handled, err := loadOwnDocument(app, e)
		if handled {
			return err
		}
		if err := pageedit.RestoreOriginal(app, document); err != nil {
			return writePageEditError(app, e, err)
		}
		return writeJSON(e, http.StatusOK, map[string]string{"status": "queued"})
	}
}

// loadOwnDocument loads the path's document for its owner only: someone it is
// shared with may read it but not change its pages.
func loadOwnDocument(app core.App, e *core.RequestEvent) (*core.Record, bool, error) {
	ownerID, err := resolveOwnerUserID(app, e)
	if err != nil {
		return nil, true, writeOwnerError(e, err)
	}
	document, err := app.FindRecordById("documents", strings.TrimSpace(e.Request.PathValue("documentId")))
	if err != nil || document.GetString("user") != ownerID {
		return nil, true, writeError(e, http.StatusNotFound, "Document not found.")
	}
	return document, false, nil
}

func writePageEditError(app core.App, e *core.RequestEvent, err error) error {
	if errors.Is(err, pageedit.ErrBusy) || errors.Is(err, pageedit.ErrSameAsAnother) {
		return writeJSON(e, http.StatusConflict, map[string]string{"detail": i18n.Of(i18n.FromRequest(e.Request), err)})
	}
	if _, ok := errors.AsType[*i18n.Error](err); ok {
		return writeBadRequest(e, err)
	}
	app.Logger().Error("document page edit failed", "document", e.Request.PathValue("documentId"), slog.Any("error", err))
	return writeError(e, http.StatusInternalServerError, "Could not change the document's pages.")
}
