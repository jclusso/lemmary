package appapi

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"
	"lemmary/backend/internal/i18n"
	"lemmary/backend/internal/pageedit"
)

type removePagesRequest struct {
	Pages []int `json:"pages"`
}

// handleGetDocumentPageThumbnail renders one page of the caller's own PDF.
func handleGetDocumentPageThumbnail(app core.App) func(*core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		page, convErr := strconv.Atoi(e.Request.PathValue("page"))
		if convErr != nil {
			return writeError(e, http.StatusBadRequest, "Page not found.")
		}
		document, handled, err := loadOwnDocument(app, e)
		if handled {
			return err
		}
		data, err := pageedit.Thumbnail(e.Request.Context(), app, document, page)
		if err != nil {
			return writePageEditError(app, e, err)
		}
		e.Response.Header().Set("Content-Type", "image/jpeg")
		// The file name is part of the record, so a page edit gives the
		// editor a fresh list and the old thumbnails are never asked for again.
		e.Response.Header().Set("Cache-Control", "private, max-age=300")
		e.Response.WriteHeader(http.StatusOK)
		_, writeErr := e.Response.Write(data)
		return writeErr
	}
}

// handleGetDocumentPages lists the pages of the caller's own PDF and which
// look blank.
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

func bindPageRecordGuard(app core.App) {
	app.OnRecordCreateRequest("documents").BindFunc(pageedit.GuardPageRecord)
	app.OnRecordUpdateRequest("documents").BindFunc(pageedit.GuardPageRecord)
}

func registerDocumentPageRoutes(g *router.RouterGroup[*core.RequestEvent], app core.App) {
	g.GET("/documents/{documentId}/pages", bindAuth(handleGetDocumentPages(app)))
	g.GET("/documents/{documentId}/pages/{page}", bindAuth(handleGetDocumentPageThumbnail(app)))
	g.POST("/documents/{documentId}/pages/removal", bindAuth(handlePostDocumentPageRemoval(app)))
	g.POST("/documents/{documentId}/original/restore", bindAuth(handlePostDocumentOriginalRestore(app)))
}

// handlePostDocumentPageRemoval takes the listed pages out of the caller's PDF
// and reports whether OCR was queued again.
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
		outcome, err := pageedit.RemovePages(e.Request.Context(), app, document, req.Pages)
		if err != nil {
			return writePageEditError(app, e, err)
		}
		return writeJSON(e, http.StatusOK, outcome)
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
		outcome, err := pageedit.RestoreOriginal(e.Request.Context(), app, document)
		if err != nil {
			return writePageEditError(app, e, err)
		}
		return writeJSON(e, http.StatusOK, outcome)
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
