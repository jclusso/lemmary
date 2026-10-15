package ocr

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"lemmary/backend/internal/aiprovider"
)

func TestJudgeBlankPagesAsksWithBothSidesAndReadsTheAnswer(t *testing.T) {
	var body string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		body = string(raw)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": "chatcmpl-blank", "object": "chat.completion", "created": 1, "model": "test-model",
			"choices": []map[string]any{{
				"index":         0,
				"message":       map[string]any{"role": "assistant", "content": "```json\n{\"pages\": [{\"page\": 2, \"blank\": true}]}\n```"},
				"finish_reason": "stop",
			}},
		})
	}))
	t.Cleanup(srv.Close)

	provider, err := NewFromAIProvider(aiprovider.Provider{
		SDK: aiprovider.SDKOpenAI, APIKey: "test-key", BaseURL: srv.URL + "/v1",
	}, "test-model", 5*time.Second, slog.Default())
	if err != nil {
		t.Fatal(err)
	}
	judge, ok := AsBlankPageJudge(provider)
	if !ok {
		t.Fatal("an AI OCR provider wrapped for metrics is not a judge")
	}
	if judge.Model() != "test-model" {
		t.Fatalf("model = %q", judge.Model())
	}

	answers, err := judge.JudgeBlankPages(context.Background(), []PageToJudge{
		{Page: 2, Image: []byte("back"), Previous: []byte("front")},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !answers[2] {
		t.Fatalf("answers = %v, want page 2 blank", answers)
	}
	if strings.Count(body, "data:image/jpeg;base64,") != 2 || !strings.Contains(body, "The page before page 2") {
		t.Fatalf("request did not carry both sides of the sheet: %.300s", body)
	}
}

func TestAsBlankPageJudgeRefusesProvidersThatCannotLook(t *testing.T) {
	provider, err := NewFromAIProvider(aiprovider.Provider{SDK: aiprovider.SDKMistral, APIKey: "k"}, "mistral-ocr-latest", time.Second, slog.Default())
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := AsBlankPageJudge(provider); ok {
		t.Fatal("Mistral OCR was taken for a judge")
	}
}

func TestParseBlankVerdictsRefusesAMissingPage(t *testing.T) {
	pages := []PageToJudge{{Page: 2}, {Page: 4}}
	if _, err := parseBlankVerdicts(`{"pages": [{"page": 2, "blank": true}]}`, pages); err == nil {
		t.Fatal("an answer without page 4 was accepted")
	}
	if _, err := parseBlankVerdicts("no idea", pages); err == nil {
		t.Fatal("an answer without JSON was accepted")
	}
	got, err := parseBlankVerdicts(`{"pages": [{"page": 2, "blank": true}, {"page": 4, "blank": false}]}`, pages)
	if err != nil || !got[2] || got[4] {
		t.Fatalf("verdicts = %v, %v", got, err)
	}
}
