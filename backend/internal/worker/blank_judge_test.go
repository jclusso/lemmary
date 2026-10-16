package worker

import (
	"context"
	"errors"
	"slices"
	"testing"

	"lemmary/backend/internal/blankpage"
	"lemmary/backend/internal/models"
	"lemmary/backend/internal/ocr"
	"lemmary/backend/internal/pdftool/testpdf"
)

// judgingOCR is an OCR provider that can also look at pages.
type judgingOCR struct {
	answers map[int]bool
	err     error
	asked   []ocr.PageToJudge
}

func (*judgingOCR) Name() string  { return "judge" }
func (*judgingOCR) Model() string { return "judge-model" }
func (*judgingOCR) ExtractText(context.Context, string, string) (string, error) {
	return "text", nil
}
func (j *judgingOCR) JudgeBlankPages(_ context.Context, pages []ocr.PageToJudge) (map[int]bool, error) {
	j.asked = pages
	return j.answers, j.err
}

// Page 2 of 3 is unsure: the local check guessed content, the AI says blank.
var unsureSecond = []blankpage.Verdict{{Blank: false, Sure: true}, {Blank: false, Sure: false}, {Blank: false, Sure: true}}

func TestJudgeUnsurePagesAsksOnlyAboutUnsurePages(t *testing.T) {
	judge := &judgingOCR{answers: map[int]bool{2: true}}
	state := ocrState(t, pagedDocument(models.PageRecord{}), testpdf.Multipage(3))
	state.OCR = judge
	state.Cfg.BlankPageAICheckEnabled = true
	blank := make([]bool, 3)

	if err := judgeUnsurePages(context.Background(), state, unsureSecond, blank); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(blank, []bool{false, true, false}) {
		t.Fatalf("blank = %v, want only page 2", blank)
	}
	if len(judge.asked) != 1 || judge.asked[0].Page != 2 || judge.asked[0].Previous == nil || judge.asked[0].Image == nil {
		t.Fatalf("asked about %+v, want page 2 with page 1 beside it", judge.asked)
	}
	if state.RanWith.Model != "judge-model" {
		t.Fatalf("the step does not name the model it called: %+v", state.RanWith)
	}
}

func TestJudgeUnsurePagesStaysLocal(t *testing.T) {
	for name, tc := range map[string]struct {
		enabled  bool
		verdicts []blankpage.Verdict
	}{
		"with the check off":   {false, unsureSecond},
		"with every page sure": {true, []blankpage.Verdict{{Sure: true}, {Sure: true}, {Sure: true}}},
	} {
		judge := &judgingOCR{answers: map[int]bool{}}
		state := ocrState(t, pagedDocument(models.PageRecord{}), testpdf.Multipage(3))
		state.OCR = judge
		state.Cfg.BlankPageAICheckEnabled = tc.enabled
		if err := judgeUnsurePages(context.Background(), state, tc.verdicts, make([]bool, 3)); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if judge.asked != nil || state.RanWith.Provider != "" {
			t.Errorf("%s: the AI was asked", name)
		}
	}

	// An OCR provider that cannot look at pages is not asked either.
	state := ocrState(t, pagedDocument(models.PageRecord{}), testpdf.Multipage(3))
	state.OCR = stubOCR{}
	state.Cfg.BlankPageAICheckEnabled = true
	if err := judgeUnsurePages(context.Background(), state, unsureSecond, make([]bool, 3)); err != nil {
		t.Fatal(err)
	}
}

func TestJudgeUnsurePagesKeepsTheLocalGuessOnFailure(t *testing.T) {
	judge := &judgingOCR{err: errors.New("provider is down")}
	state := ocrState(t, pagedDocument(models.PageRecord{}), testpdf.Multipage(3))
	state.OCR = judge
	state.Cfg.BlankPageAICheckEnabled = true
	blank := []bool{false, true, false}
	if err := judgeUnsurePages(context.Background(), state, unsureSecond, blank); err == nil {
		t.Fatal("a failed AI check reported success")
	}
	if !slices.Equal(blank, []bool{false, true, false}) {
		t.Fatalf("a failed check changed the local verdicts: %v", blank)
	}
}
