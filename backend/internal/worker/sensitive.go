package worker

import (
	"github.com/pocketbase/pocketbase/core"

	"lemmary/backend/internal/aiprovider"
	"lemmary/backend/internal/config"
	"lemmary/backend/internal/models"
	"lemmary/backend/internal/ocr"
	"lemmary/backend/internal/textextract"
)

// sensitiveBlocksOCR reports whether a sensitive document's file may not go to
// the OCR provider. Native text extraction never leaves the process, so a
// format it reads is never blocked; anything else needs the local engine.
func sensitiveBlocksOCR(state *StepState, provider ocr.Provider) bool {
	if !models.IsSensitive(state.Document) || textextract.Supports(state.MimeType) {
		return false
	}
	return provider == nil || !aiprovider.IsLocalOCR(provider.Name())
}

// sensitiveBlocksEmbedding reports whether a sensitive document's text may not
// go to the embedder.
func sensitiveBlocksEmbedding(cfg config.Config, document *core.Record) bool {
	return models.IsSensitive(document) && !config.EmbedsSensitive(cfg)
}
