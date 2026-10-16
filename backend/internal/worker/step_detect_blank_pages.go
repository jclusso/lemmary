package worker

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"lemmary/backend/internal/blankpage"
	"lemmary/backend/internal/models"
	"lemmary/backend/internal/ocr"
	"lemmary/backend/internal/pdftool"
)

// judgeEdge is the size pages are sent to the AI at: enough to read mirrored
// text through a page, small enough to cost a few hundred tokens each.
const judgeEdge = 1024

// DetectBlankPagesStep records which pages of a PDF look blank, for OCR to
// leave out and the document page to suggest removing. It only ever records:
// the file is changed by the owner, never here.
//
// The local check settles most pages. With the AI check on and an OCR model
// that reads images, the pages it is unsure of go to that model in one
// request, with the page before each, and its answer decides them.
type DetectBlankPagesStep struct{}

func (s *DetectBlankPagesStep) Name() string { return models.StepDetectBlankPages }

// Runs whenever it is queued, forced or not: it is local, costs about as much
// as the preview, and a stale answer would steer what OCR reads.
func (s *DetectBlankPagesStep) ShouldSkip(state *StepState) (bool, error) {
	return state.MimeType != "application/pdf", nil
}

// Fails softly: without it OCR reads every page, which is what it did before
// the step existed. A failed AI check also fails softly, after the local
// verdicts are stored.
func (s *DetectBlankPagesStep) Run(ctx context.Context, state *StepState) error {
	if err := ensureTempFile(state); err != nil {
		return err
	}
	count, err := pdftool.PageCount(ctx, state.TmpPath)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrStepSoft, err)
	}
	verdicts, err := blankpage.Detect(ctx, state.TmpPath, count)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrStepSoft, err)
	}
	blank := make([]bool, count)
	for i, v := range verdicts {
		blank[i] = v.Blank
	}

	judgeErr := judgeUnsurePages(ctx, state, verdicts, blank)

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
	if judgeErr != nil {
		return fmt.Errorf("%w: AI check failed, kept the local result: %w", ErrStepSoft, judgeErr)
	}
	return nil
}

// judgeUnsurePages asks the OCR model about the pages the local check was
// unsure of and writes its answers into blank. It does nothing when the check
// is off, the OCR model cannot look at pages, or every page is settled.
func judgeUnsurePages(ctx context.Context, state *StepState, verdicts []blankpage.Verdict, blank []bool) error {
	if !state.Cfg.BlankPageAICheckEnabled || state.OCR == nil {
		return nil
	}
	judge, ok := ocr.AsBlankPageJudge(state.OCR)
	if !ok {
		return nil
	}
	var unsure []int
	for i, v := range verdicts {
		if !v.Sure {
			unsure = append(unsure, i)
		}
	}
	if len(unsure) == 0 {
		return nil
	}

	dir, err := os.MkdirTemp("", "lemmary-blankjudge-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	images, err := pdftool.RenderPagesAs(ctx, state.TmpPath, filepath.Join(dir, "pages"), pdftool.JPEG, judgeEdge, len(verdicts))
	if err != nil {
		return err
	}
	pages := make([]ocr.PageToJudge, 0, len(unsure))
	for _, i := range unsure {
		page := ocr.PageToJudge{Page: i + 1}
		if page.Image, err = os.ReadFile(images[i]); err != nil {
			return err
		}
		if i > 0 {
			if page.Previous, err = os.ReadFile(images[i-1]); err != nil {
				return err
			}
		}
		pages = append(pages, page)
	}

	state.RanWith.Provider, state.RanWith.Model = judge.Name(), judge.Model()
	judgeCtx, cancel := context.WithTimeout(ctx, state.Cfg.OCRTimeout)
	defer cancel()
	answers, err := judge.JudgeBlankPages(judgeCtx, pages)
	if err != nil {
		return err
	}
	for _, page := range pages {
		blank[page.Page-1] = answers[page.Page]
	}
	state.Logger.Info("blank pages judged by AI", "pages", len(pages), "model", judge.Model())
	return nil
}
