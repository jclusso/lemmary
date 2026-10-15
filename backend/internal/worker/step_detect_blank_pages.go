package worker

import (
	"context"
	"fmt"

	"lemmary/backend/internal/blankpage"
	"lemmary/backend/internal/models"
	"lemmary/backend/internal/pdftool"
)

// DetectBlankPagesStep records which pages of a PDF look blank, for OCR to
// leave out and the document page to suggest removing. It only ever records:
// the file is changed by the owner, never here.
type DetectBlankPagesStep struct{}

func (s *DetectBlankPagesStep) Name() string { return models.StepDetectBlankPages }

// Runs whenever it is queued, forced or not: it is local, costs about as much
// as the preview, and a stale answer would steer what OCR reads.
func (s *DetectBlankPagesStep) ShouldSkip(state *StepState) (bool, error) {
	return state.MimeType != "application/pdf", nil
}

// Fails softly: without it OCR reads every page, which is what it did before
// the step existed.
func (s *DetectBlankPagesStep) Run(ctx context.Context, state *StepState) error {
	if err := ensureTempFile(state); err != nil {
		return err
	}
	count, err := pdftool.PageCount(ctx, state.TmpPath)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrStepSoft, err)
	}
	blank, err := blankpage.Detect(ctx, state.TmpPath, count)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrStepSoft, err)
	}

	record := models.LoadPageRecord(state.Document)
	current := record.Current(count)
	var found []int
	for i, isBlank := range blank {
		if isBlank {
			found = append(found, current[i])
		}
	}
	record.WithBlank(current, found).Store(state.Document)
	if err := state.App.Save(state.Document); err != nil {
		return fmt.Errorf("save blank pages: %w", err)
	}
	state.Logger.Info("blank pages detected", "pages", found)
	return nil
}
