package ocr

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/shared"

	"lemmary/backend/internal/ai"
	"lemmary/backend/internal/aiprovider"
	"lemmary/backend/internal/metrics"
)

// BlankPageJudge looks at pages the local blank check could not settle. Only
// providers that read documents as images implement it.
type BlankPageJudge interface {
	Name() string
	Model() string
	// JudgeBlankPages answers, for each page asked about, whether it is blank.
	JudgeBlankPages(ctx context.Context, pages []PageToJudge) (map[int]bool, error)
}

// PageToJudge is one page and, when there is one, the page before it, both as
// JPEG. The page before is what may show through.
type PageToJudge struct {
	Page     int
	Image    []byte
	Previous []byte
}

// AsBlankPageJudge finds the judge behind p, looking through the metrics
// wrapper withMetrics puts around every provider.
func AsBlankPageJudge(p Provider) (BlankPageJudge, bool) {
	t, wrapped := unwrapTimed(p)
	if wrapped {
		p = t.Provider
	}
	judge, ok := p.(BlankPageJudge)
	if !ok {
		return nil, false
	}
	if wrapped {
		return timedJudge{BlankPageJudge: judge, sdk: t.sdk, model: t.model}, true
	}
	return judge, true
}

func unwrapTimed(p Provider) (timed, bool) {
	switch v := p.(type) {
	case timed:
		return v, true
	case timedLimited:
		return v.timed, true
	}
	return timed{}, false
}

type timedJudge struct {
	BlankPageJudge
	sdk, model string
}

func (t timedJudge) JudgeBlankPages(ctx context.Context, pages []PageToJudge) (_ map[int]bool, err error) {
	defer metrics.TimeAICall(ctx, "blank_pages", t.sdk, t.model)(&err)
	return t.BlankPageJudge.JudgeBlankPages(ctx, pages)
}

const blankJudgeSystemPrompt = `You check scanned pages for a document archive and say which ones are blank.

A page is blank when it has nothing of its own on it: empty paper, specks, scanner shadows, or only what shows through from the other side of the sheet. Show-through looks like faint, mirrored text or lines; when the page before is given, it is usually that page reversed left to right.

A page is not blank when anything was printed, written or stamped on it, however little: a single line, a signature, a stamp, a handwritten note, a page number. Faint or faded print is still print. When you cannot tell, answer not blank.

Return ONLY valid JSON of this shape:
{"pages": [{"page": 2, "blank": true}]}
with one entry for every page you were asked about. Do not include markdown or explanation.`

func (p *LLMProvider) Model() string { return p.model }

func (p *LLMProvider) JudgeBlankPages(ctx context.Context, pages []PageToJudge) (map[int]bool, error) {
	if len(pages) == 0 {
		return map[int]bool{}, nil
	}
	ctx = aiprovider.EnsureSession(ctx, "blank_pages")

	parts := []openai.ChatCompletionContentPartUnionParam{
		openai.TextContentPart(fmt.Sprintf("Say whether each of these %d pages is blank.", len(pages))),
	}
	for _, page := range pages {
		if page.Previous != nil {
			parts = append(parts,
				openai.TextContentPart(fmt.Sprintf("The page before page %d:", page.Page)),
				jpegPart(page.Previous))
		}
		parts = append(parts, openai.TextContentPart(fmt.Sprintf("Page %d:", page.Page)), jpegPart(page.Image))
	}

	resp, err := p.llm.Complete(ctx, openai.ChatCompletionNewParams{
		Model: shared.ChatModel(p.model),
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(blankJudgeSystemPrompt),
			openai.UserMessage(parts),
		},
		ResponseFormat: openai.ChatCompletionNewParamsResponseFormatUnion{
			OfJSONObject: &shared.ResponseFormatJSONObjectParam{},
		},
		Temperature: ai.CompletionTemperature(p.model, 0),
	}, "purpose", "blank_pages", "messages", 2)
	if err != nil {
		return nil, fmt.Errorf("blank page check: %w", err)
	}
	if len(resp.Choices) == 0 {
		return nil, errors.New("blank page check returned no choices")
	}
	return parseBlankVerdicts(resp.Choices[0].Message.Content, pages)
}

func jpegPart(data []byte) openai.ChatCompletionContentPartUnionParam {
	return openai.ImageContentPart(openai.ChatCompletionContentPartImageImageURLParam{
		URL: "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(data),
	})
}

// parseBlankVerdicts reads the model's answer, refusing one that leaves out a
// page it was asked about rather than guessing for it.
func parseBlankVerdicts(content string, pages []PageToJudge) (map[int]bool, error) {
	start, end := strings.Index(content, "{"), strings.LastIndex(content, "}")
	if start < 0 || end < start {
		return nil, fmt.Errorf("blank page check: no JSON in the answer")
	}
	var answer struct {
		Pages []struct {
			Page  int  `json:"page"`
			Blank bool `json:"blank"`
		} `json:"pages"`
	}
	if err := json.Unmarshal([]byte(content[start:end+1]), &answer); err != nil {
		return nil, fmt.Errorf("blank page check: %w", err)
	}
	verdicts := make(map[int]bool, len(answer.Pages))
	for _, v := range answer.Pages {
		verdicts[v.Page] = v.Blank
	}
	for _, page := range pages {
		if _, ok := verdicts[page.Page]; !ok {
			return nil, fmt.Errorf("blank page check: no answer for page %d", page.Page)
		}
	}
	return verdicts, nil
}
