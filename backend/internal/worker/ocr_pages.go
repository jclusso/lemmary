package worker

import (
	"context"
	"os"
	"path/filepath"
	"slices"

	"lemmary/backend/internal/models"
	"lemmary/backend/internal/pdftool"
)

// ocrInput is the file an OCR provider reads: the document's PDF without the
// pages detect_blank_pages found blank, so they cost nothing and their
// bleed-through is not read as text. A page the owner confirmed is always read.
type ocrInput struct {
	path string
	// pages is how many pages path holds.
	pages int
	// current holds the uploaded page number of each page of the document's
	// file, and skipped those left out. current is nil when the file is not a
	// PDF poppler can read, which leaves the page record as it was.
	current []int
	skipped []int
	cleanup func()
}

// Any failure here sends the whole file: leaving out blank pages saves tokens,
// and is not worth failing an OCR run over.
func prepareOCRInput(ctx context.Context, state *StepState) ocrInput {
	in := ocrInput{
		path:    state.TmpPath,
		pages:   max(1, int(state.Document.GetFloat("page_count"))),
		cleanup: func() {},
	}
	if state.MimeType != "application/pdf" {
		return in
	}
	count, err := pdftool.PageCount(ctx, state.TmpPath)
	if err != nil {
		state.Logger.Warn("ocr: page count failed; reading every page", "error", err)
		return in
	}
	record := models.LoadPageRecord(state.Document)
	in.pages = count
	in.current = record.Current(count)

	var remove, skipped []int
	for i, uploaded := range in.current {
		if record.IsBlank(uploaded) && !record.IsConfirmed(uploaded) {
			remove = append(remove, i+1)
			skipped = append(skipped, uploaded)
		}
	}
	// A document that is blank throughout is read whole: what the provider
	// makes of it is a better answer than no text at all.
	if len(remove) == 0 || len(remove) == count {
		return in
	}

	dir, err := os.MkdirTemp("", "lemmary-ocr-*")
	if err != nil {
		return in
	}
	out := filepath.Join(dir, "document.pdf")
	if err := pdftool.Without(ctx, state.TmpPath, count, remove, out); err != nil {
		os.RemoveAll(dir)
		state.Logger.Warn("ocr: leaving out blank pages failed; reading every page", "error", err)
		return in
	}
	state.Logger.Info("ocr: leaving out blank pages", "pages", remove)
	in.path = out
	in.pages = count - len(remove)
	in.skipped = skipped
	in.cleanup = func() { os.RemoveAll(dir) }
	return in
}

// record notes on the document which of its pages the text came from.
func (in ocrInput) record(state *StepState) {
	if in.current == nil {
		return
	}
	models.LoadPageRecord(state.Document).WithSkipped(in.current, slices.Clone(in.skipped)).Store(state.Document)
}
