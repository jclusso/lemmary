package worker

import (
	"context"
	"testing"

	"github.com/pocketbase/pocketbase/core"

	"lemmary/backend/internal/aiprovider"
	"lemmary/backend/internal/config"
	"lemmary/backend/internal/models"
	"lemmary/backend/internal/textextract"
)

type namedOCR struct{ name string }

func (p namedOCR) Name() string { return p.name }
func (namedOCR) ExtractText(context.Context, string, string) (string, error) {
	return "text", nil
}

func sensitiveTestDocument(t *testing.T, sensitive bool) *core.Record {
	t.Helper()
	collection := core.NewBaseCollection("documents")
	collection.Fields.Add(
		&core.TextField{Name: "ocr_text", Max: models.MaxOCRTextRunes},
		&core.TextField{Name: "duplicate_of", Max: 15},
		&core.BoolField{Name: models.FieldSensitive},
	)
	record := core.NewRecord(collection)
	record.Set(models.FieldSensitive, sensitive)
	return record
}

func TestOCRSkipsASensitiveScanUnlessALocalEngineIsAllowed(t *testing.T) {
	t.Parallel()
	external := namedOCR{name: "mistral"}
	docling := namedOCR{name: aiprovider.SDKDocling}
	forced := map[string]bool{models.StepOCR: true}

	cases := []struct {
		name      string
		sensitive bool
		provider  namedOCR
		mime      string
		wantSkip  bool
	}{
		{"an ordinary scan goes to the provider", false, external, "application/pdf", false},
		{"a sensitive scan never reaches an external provider", true, external, "application/pdf", true},
		{"a sensitive scan may use local OCR", true, docling, "application/pdf", false},
		{"native extraction never leaves the host", true, external, textextract.MIMEPlainText, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			step := &OCRStep{Provider: tc.provider}
			state := &StepState{
				Document:   sensitiveTestDocument(t, tc.sensitive),
				MimeType:   tc.mime,
				ForceSteps: forced,
			}
			skip, err := step.ShouldSkip(state)
			if err != nil {
				t.Fatal(err)
			}
			if skip != tc.wantSkip {
				t.Fatalf("skip = %v, want %v (a forced reprocess must not get around it)", skip, tc.wantSkip)
			}
		})
	}
}

func TestExtractionAndApplySkipSensitiveDocumentsEvenWhenForced(t *testing.T) {
	t.Parallel()
	state := &StepState{
		Document:   sensitiveTestDocument(t, true),
		ForceSteps: map[string]bool{models.StepExtractMetadata: true},
	}
	for _, step := range []Step{&ExtractMetadataStep{}, &ApplyMetadataStep{}} {
		skip, err := step.ShouldSkip(state)
		if err != nil {
			t.Fatal(err)
		}
		if !skip {
			t.Fatalf("%s must skip a sensitive document", step.Name())
		}
	}
}

func TestEmbedSkipsASensitiveDocumentUnlessTheLocalEmbedderIsAllowed(t *testing.T) {
	t.Parallel()
	local := &aiprovider.Provider{SDK: aiprovider.SDKLocalEmbeddings}
	hosted := &aiprovider.Provider{SDK: aiprovider.SDKOpenAI}
	cases := []struct {
		name     string
		provider *aiprovider.Provider
		wantSkip bool
	}{
		{"a hosted embedder never sees it", hosted, true},
		{"the local sidecar may embed it", local, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			document := sensitiveTestDocument(t, true)
			document.Set("ocr_text", "wages")
			step := &EmbedStep{Embedder: stubEmbedder{}}
			state := &StepState{
				Document: document,
				Cfg:      config.Config{EmbeddingProvider: tc.provider},
			}
			skip, err := step.ShouldSkip(state)
			if err != nil {
				t.Fatal(err)
			}
			if skip != tc.wantSkip {
				t.Fatalf("skip = %v, want %v", skip, tc.wantSkip)
			}
		})
	}
}

func TestASensitiveDocumentFinishesInReviewWithoutExtraction(t *testing.T) {
	app := bootAppForEnqueue(t)
	document, err := app.FindRecordById("documents", makeDocumentForEnqueue(t, app))
	if err != nil {
		t.Fatal(err)
	}
	document.Set(models.FieldSensitive, true)
	document.Set("processing_status", models.DocStatusProcessing)
	if err := app.Save(document); err != nil {
		t.Fatal(err)
	}
	if err := finalizeDocumentWithoutApply(app, document, models.FullPipelineSteps); err != nil {
		t.Fatal(err)
	}
	if got := document.GetString("processing_status"); got != models.DocStatusNeedsReview {
		t.Fatalf("processing_status = %q, want %q: no model named it, so a person has to", got, models.DocStatusNeedsReview)
	}
}
