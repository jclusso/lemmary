// Package pageedit removes pages from a stored PDF document and puts back the
// file as it was uploaded.
//
// The first removal keeps the uploaded file in original_file. Later removals
// leave it alone, so a restore always returns to what the scanner produced.
// Either change replaces the document's file and queues preview and OCR again;
// metadata is left as it is, because the owner may have corrected it.
package pageedit

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sync"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
	"golang.org/x/image/draw"

	"lemmary/backend/internal/config"
	"lemmary/backend/internal/duplicates"
	"lemmary/backend/internal/i18n"
	"lemmary/backend/internal/models"
	"lemmary/backend/internal/pdftool"
	"lemmary/backend/internal/worker"
)

const (
	// MaxPages bounds an inspection, which renders every page in one request.
	MaxPages = 200
	// renderEdge is large enough that a short line of text is still several
	// pixels of solid ink when the page is measured for blankness.
	renderEdge = 800
	// thumbEdge is what the editor shows; half the measured render.
	thumbEdge    = 400
	thumbQuality = 70
)

var (
	ErrNotPDF        = i18n.Errorf("Only PDF documents have pages to remove.")
	ErrTooManyPages  = i18n.Errorf("Documents of more than %d pages cannot be edited.", MaxPages)
	ErrBusy          = i18n.Errorf("The document is being processed. Try again when it finishes.")
	ErrNoPages       = i18n.Errorf("Choose at least one page to remove.")
	ErrAllPages      = i18n.Errorf("A document must keep at least one page.")
	ErrPageOutOfDoc  = i18n.Errorf("The document has no such page.")
	ErrNoOriginal    = i18n.Errorf("There is no original file to restore.")
	ErrSameAsAnother = i18n.Errorf("Another document already has exactly these pages.")
)

// Page is one page of the editor: its number, whether it looks blank, and a
// JPEG thumbnail as a data URL.
type Page struct {
	Number    int    `json:"page"`
	Blank     bool   `json:"blank"`
	Thumbnail string `json:"thumbnail"`
}

type Overview struct {
	Pages       []Page `json:"pages"`
	HasOriginal bool   `json:"has_original"`
}

// Inspect renders every page of the document's PDF once, measuring each for
// blankness and shrinking it to a thumbnail from the same render.
func Inspect(ctx context.Context, app core.App, document *core.Record) (Overview, error) {
	if !isPDF(document) {
		return Overview{}, ErrNotPDF
	}
	workDir, err := os.MkdirTemp("", "lemmary-pageedit-*")
	if err != nil {
		return Overview{}, err
	}
	defer os.RemoveAll(workDir)

	pdfPath := filepath.Join(workDir, "document.pdf")
	if err := copyStoredFile(app, document, "file", pdfPath); err != nil {
		return Overview{}, err
	}
	count, err := pdftool.PageCount(ctx, pdfPath)
	if err != nil {
		return Overview{}, err
	}
	if count > MaxPages {
		return Overview{}, ErrTooManyPages
	}
	rendered, err := pdftool.RenderPages(ctx, pdfPath, filepath.Join(workDir, "pages"), renderEdge, count)
	if err != nil {
		return Overview{}, err
	}

	pages := make([]Page, 0, count)
	for i, path := range rendered {
		page, err := inspectPage(path)
		if err != nil {
			return Overview{}, fmt.Errorf("page %d: %w", i+1, err)
		}
		page.Number = i + 1
		pages = append(pages, page)
	}
	return Overview{Pages: pages, HasOriginal: document.GetString("original_file") != ""}, nil
}

func inspectPage(path string) (Page, error) {
	file, err := os.Open(path)
	if err != nil {
		return Page{}, err
	}
	defer file.Close()
	img, err := png.Decode(file)
	if err != nil {
		return Page{}, err
	}
	thumb, err := thumbnail(img)
	if err != nil {
		return Page{}, err
	}
	return Page{Blank: IsBlank(img), Thumbnail: thumb}, nil
}

func thumbnail(img image.Image) (string, error) {
	b := img.Bounds()
	scale := float64(thumbEdge) / float64(max(b.Dx(), b.Dy()))
	small := image.NewRGBA(image.Rect(0, 0, max(1, int(float64(b.Dx())*scale)), max(1, int(float64(b.Dy())*scale))))
	draw.ApproxBiLinear.Scale(small, small.Bounds(), img, b, draw.Src, nil)

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, small, &jpeg.Options{Quality: thumbQuality}); err != nil {
		return "", err
	}
	return "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

// RemovePages replaces the document's file with one lacking the given pages.
func RemovePages(ctx context.Context, app core.App, document *core.Record, remove []int) error {
	if !isPDF(document) {
		return ErrNotPDF
	}
	release, err := claim(app, document)
	if err != nil {
		return err
	}
	defer release()
	workDir, err := os.MkdirTemp("", "lemmary-pageedit-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(workDir)

	sourcePath := filepath.Join(workDir, "source.pdf")
	if err := copyStoredFile(app, document, "file", sourcePath); err != nil {
		return err
	}
	count, err := pdftool.PageCount(ctx, sourcePath)
	if err != nil {
		return err
	}
	ranges, err := keptRanges(count, remove)
	if err != nil {
		return err
	}

	outPath := filepath.Join(workDir, "result.pdf")
	if err := writeRanges(ctx, sourcePath, ranges, workDir, outPath); err != nil {
		return err
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		return err
	}

	var backup []byte
	if document.GetString("original_file") == "" {
		if backup, err = os.ReadFile(sourcePath); err != nil {
			return err
		}
	}
	return replaceFile(app, document, data, backup, false)
}

// RestoreOriginal puts the as-uploaded file back and forgets the backup.
func RestoreOriginal(app core.App, document *core.Record) error {
	if document.GetString("original_file") == "" {
		return ErrNoOriginal
	}
	release, err := claim(app, document)
	if err != nil {
		return err
	}
	defer release()
	workDir, err := os.MkdirTemp("", "lemmary-pageedit-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(workDir)

	originalPath := filepath.Join(workDir, "original.pdf")
	if err := copyStoredFile(app, document, "original_file", originalPath); err != nil {
		return err
	}
	data, err := os.ReadFile(originalPath)
	if err != nil {
		return err
	}
	return replaceFile(app, document, data, nil, true)
}

type pageRange struct{ from, to int }

// keptRanges turns the pages to remove into the runs of pages that stay.
func keptRanges(count int, remove []int) ([]pageRange, error) {
	if len(remove) == 0 {
		return nil, ErrNoPages
	}
	removed := make(map[int]bool, len(remove))
	for _, page := range remove {
		if page < 1 || page > count {
			return nil, ErrPageOutOfDoc
		}
		removed[page] = true
	}
	if len(removed) == count {
		return nil, ErrAllPages
	}

	var ranges []pageRange
	for page := 1; page <= count; page++ {
		if removed[page] {
			continue
		}
		if n := len(ranges); n > 0 && ranges[n-1].to == page-1 {
			ranges[n-1].to = page
			continue
		}
		ranges = append(ranges, pageRange{from: page, to: page})
	}
	return ranges, nil
}

func writeRanges(ctx context.Context, sourcePath string, ranges []pageRange, workDir, outPath string) error {
	if len(ranges) == 1 {
		return pdftool.ExtractRange(ctx, sourcePath, ranges[0].from, ranges[0].to, outPath)
	}
	parts := make([]string, 0, len(ranges))
	for i, r := range ranges {
		part := filepath.Join(workDir, fmt.Sprintf("part-%d.pdf", i+1))
		if err := pdftool.ExtractRange(ctx, sourcePath, r.from, r.to, part); err != nil {
			return err
		}
		parts = append(parts, part)
	}
	return pdftool.Merge(ctx, outPath, parts...)
}

// replaceFile saves data as the document's file, optionally keeping backup as
// original_file or dropping the backup, and queues the steps that read the
// file. One transaction for the reason reprocess.queueOne gives: the job must
// not become visible before the pending status lands.
func replaceFile(app core.App, document *core.Record, data, backup []byte, dropBackup bool) error {
	checksum, err := duplicates.SHA256Reader(bytes.NewReader(data))
	if err != nil {
		return err
	}
	existing, err := duplicates.FindByChecksum(app, document.GetString("user"), checksum, document.Id)
	if err != nil {
		return err
	}
	if existing != nil {
		return ErrSameAsAnother
	}

	name := uploadName(document.GetString("file"))
	file, err := filesystem.NewFileFromBytes(data, name)
	if err != nil {
		return err
	}
	document.Set("file", file)
	document.Set("checksum", checksum)
	if backup != nil {
		original, err := filesystem.NewFileFromBytes(backup, name)
		if err != nil {
			return err
		}
		document.Set("original_file", original)
	}
	if dropBackup {
		document.Set("original_file", nil)
	}
	document.Set("processing_status", models.DocStatusPending)

	return app.RunInTransaction(func(txApp core.App) error {
		if err := txApp.Save(document); err != nil {
			return err
		}
		_, err := worker.Enqueue(txApp, document.Id, models.ImportPreserveSteps,
			[]string{models.StepPreview, models.StepOCR}, config.Overrides{})
		return err
	})
}

// editing holds the ids of documents with an edit in flight, so a double click
// cannot run two edits over the same file before either has queued its job.
var editing sync.Map

// claim refuses a document that is being edited or processed, and otherwise
// holds it until release is called.
func claim(app core.App, document *core.Record) (release func(), err error) {
	if _, busy := editing.LoadOrStore(document.Id, struct{}{}); busy {
		return nil, ErrBusy
	}
	release = func() { editing.Delete(document.Id) }

	active, err := app.CountRecords("processing_jobs",
		dbx.NewExp("document = {:doc} AND finished_at = ''", dbx.Params{"doc": document.Id}))
	if err == nil && active > 0 {
		err = ErrBusy
	}
	if err != nil {
		release()
		return nil, err
	}
	return release, nil
}

func isPDF(document *core.Record) bool {
	return pdftool.RequirePDF(document.GetString("file")) == nil
}

// pocketbaseSuffix is the random part PocketBase appends to every stored file
// name. Stripped before the bytes are stored again, or each edit would add one.
var pocketbaseSuffix = regexp.MustCompile(`_[a-zA-Z0-9]{10}(\.[^.]+)$`)

func uploadName(stored string) string {
	return pocketbaseSuffix.ReplaceAllString(stored, "$1")
}

func copyStoredFile(app core.App, document *core.Record, field, dst string) error {
	name := document.GetString(field)
	if name == "" {
		return errors.New("pageedit: document has no " + field)
	}
	fsys, err := app.NewFilesystem()
	if err != nil {
		return err
	}
	defer fsys.Close()

	reader, err := fsys.GetReader(document.BaseFilesPath() + "/" + name)
	if err != nil {
		return fmt.Errorf("pageedit: open %s: %w", field, err)
	}
	defer reader.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, reader); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}
