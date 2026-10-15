package worker

import (
	"context"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"lemmary/backend/internal/config"
	"lemmary/backend/internal/models"
	"lemmary/backend/internal/pdftool"
	"lemmary/backend/internal/pdftool/testpdf"
)

// pageCountingOCR answers with how many pages the file it was handed holds.
type pageCountingOCR struct{ pages int }

func (*pageCountingOCR) Name() string { return "counting" }

func (p *pageCountingOCR) ExtractText(ctx context.Context, path, _ string) (string, error) {
	count, err := pdftool.PageCount(ctx, path)
	p.pages = count
	return "text", err
}

func pagedDocument(record models.PageRecord) *core.Record {
	docs := coreTestDocumentsCollection()
	for _, name := range models.PageRecordFields {
		docs.Fields.Add(&core.JSONField{Name: name})
	}
	doc := core.NewRecord(docs)
	record.Store(doc)
	return doc
}

func ocrState(t *testing.T, doc *core.Record, pdf []byte) *StepState {
	t.Helper()
	for _, binary := range []string{"pdfinfo", "pdftoppm", "pdfseparate", "pdfunite"} {
		if _, err := exec.LookPath(binary); err != nil {
			t.Skipf("%s not installed", binary)
		}
	}
	path := filepath.Join(t.TempDir(), "scan.pdf")
	if err := os.WriteFile(path, pdf, 0o600); err != nil {
		t.Fatal(err)
	}
	return &StepState{
		Document: doc,
		TmpPath:  path,
		MimeType: "application/pdf",
		Cfg:      config.Config{OCRTimeout: time.Minute},
		Logger:   slog.New(slog.DiscardHandler),
	}
}

func TestOCRLeavesOutBlankPages(t *testing.T) {
	doc := pagedDocument(models.PageRecord{})
	state := ocrState(t, doc, testpdf.MixedText(4, 2))
	provider := &pageCountingOCR{}

	if _, _, err := resolveOCRText(context.Background(), state, provider); err != nil {
		t.Fatal(err)
	}
	if provider.pages != 2 {
		t.Fatalf("OCR read %d pages, want the 2 with text", provider.pages)
	}
	if got := models.LoadPageRecord(doc).Skipped; !slices.Equal(got, []int{3, 4}) {
		t.Fatalf("skipped = %v, want [3 4]", got)
	}
}

// Numbers are the upload's: the file holds uploaded pages 1, 2 and 5, and the
// owner said page 5 has content although it looks blank.
func TestOCRReadsConfirmedPagesAndNumbersByUpload(t *testing.T) {
	doc := pagedDocument(models.PageRecord{Map: []int{1, 2, 5}, Confirmed: []int{5}, Skipped: []int{3, 4, 5}})
	state := ocrState(t, doc, testpdf.MixedText(3, 2))
	provider := &pageCountingOCR{}

	if _, _, err := resolveOCRText(context.Background(), state, provider); err != nil {
		t.Fatal(err)
	}
	if provider.pages != 3 {
		t.Fatalf("OCR read %d pages, want all 3", provider.pages)
	}
	// 3 and 4 are no longer in the file, so what was known about them stays.
	if got := models.LoadPageRecord(doc).Skipped; !slices.Equal(got, []int{3, 4}) {
		t.Fatalf("skipped = %v, want [3 4]", got)
	}
}

func TestOCRReadsADocumentThatIsBlankThroughout(t *testing.T) {
	doc := pagedDocument(models.PageRecord{})
	state := ocrState(t, doc, testpdf.Blank(2))
	provider := &pageCountingOCR{}

	if _, _, err := resolveOCRText(context.Background(), state, provider); err != nil {
		t.Fatal(err)
	}
	if provider.pages != 2 || len(models.LoadPageRecord(doc).Skipped) != 0 {
		t.Fatalf("read %d pages, skipped %v; want both read", provider.pages, models.LoadPageRecord(doc).Skipped)
	}
}
