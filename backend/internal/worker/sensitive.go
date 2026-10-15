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
// format it reads is never blocked; anything else needs a local engine the
// admin has not also ruled out.
func sensitiveBlocksOCR(state *StepState, provider ocr.Provider) bool {
	if !models.IsSensitive(state.Document) || textextract.Supports(state.MimeType) {
		return false
	}
	local := provider != nil && aiprovider.IsLocalOCR(provider.Name())
	return !local || state.Cfg.SensitiveBlockLocal
}

// sensitiveBlocksEmbedding reports whether a sensitive document's text may not
// go to the embedder.
func sensitiveBlocksEmbedding(cfg config.Config, document *core.Record) bool {
	return models.IsSensitive(document) && !config.EmbedsSensitive(cfg)
}
