// Package pageedit removes pages from a stored PDF document and puts back the
// file as it was uploaded.
//
// The first removal keeps the uploaded file in original_file. Later removals
// leave it alone, so a restore always returns to what the scanner produced.
//
// An edit works out from the document's models.PageRecord whether its OCR text
// still holds. Removing only pages that OCR left out as blank, or restoring
// them, changes no text and queues nothing; removing a page that was read, or
// keeping one that was not, queues OCR again. Metadata is never re-extracted,
// because the owner may have corrected it.
package pageedit

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sync"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"

	"lemmary/backend/internal/blankpage"
	"lemmary/backend/internal/config"
	"lemmary/backend/internal/duplicates"
	"lemmary/backend/internal/i18n"
	"lemmary/backend/internal/models"
	"lemmary/backend/internal/pdftool"
	"lemmary/backend/internal/preview"
	"lemmary/backend/internal/worker"
)

const (
	// MaxPages bounds an inspection, which renders every page in one request.
	MaxPages = 200
	// thumbEdge is what the editor shows.
	thumbEdge = 400
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

// rereadSteps run after an edit that changed which pages the text should come
// from. apply_metadata is left out so corrected metadata stays.
var rereadSteps = []string{models.StepOCR, models.StepDetectDuplicates, models.StepEmbed, models.StepLinkRelated}

// Page is one page of the editor. Blank means it looks blank and the owner has
// not said otherwise; Unread that OCR left it out.
type Page struct {
	Number    int    `json:"page"`
	Blank     bool   `json:"blank"`
	Unread    bool   `json:"unread"`
	Thumbnail string `json:"thumbnail"`
}

type Overview struct {
	Pages       []Page `json:"pages"`
	HasOriginal bool   `json:"has_original"`
}

// Outcome tells the caller whether the edit queued OCR again.
type Outcome struct {
	Reread bool `json:"reread"`
}

// Inspect renders every page of the document's PDF, measuring each for
// blankness and returning a thumbnail of it.
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
	record := models.LoadPageRecord(document)
	blank, err := blankPages(ctx, record, pdfPath, count)
	if err != nil {
		return Overview{}, err
	}
	// A JPEG straight from pdftoppm: compressing a noisy scan as PNG takes it
	// over twenty times longer, and the browser shows a JPEG as well.
	thumbs, err := pdftool.RenderPagesAs(ctx, pdfPath, filepath.Join(workDir, "thumbs"), pdftool.JPEG, thumbEdge, count)
	if err != nil {
		return Overview{}, err
	}

	pages := make([]Page, 0, count)
	for i := range count {
		thumb, err := os.ReadFile(thumbs[i])
		if err != nil {
			return Overview{}, fmt.Errorf("page %d: %w", i+1, err)
		}
		uploaded := record.Uploaded(i + 1)
		pages = append(pages, Page{
			Number:    i + 1,
			Blank:     blank[i] && !record.IsConfirmed(uploaded),
			Unread:    record.IsSkipped(uploaded),
			Thumbnail: "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(thumb),
		})
	}
	return Overview{Pages: pages, HasOriginal: document.GetString("original_file") != ""}, nil
}

// blankPages is what the detect_blank_pages step recorded, which may include
// an AI's answer the editor should agree with, or the local check for a file
// the step has not seen.
func blankPages(ctx context.Context, record models.PageRecord, pdfPath string, count int) ([]bool, error) {
	blank := make([]bool, count)
	if record.Detected() {
		for i := range blank {
			blank[i] = record.IsBlank(record.Uploaded(i + 1))
		}
		return blank, nil
	}
	verdicts, err := blankpage.Detect(ctx, pdfPath, count)
	if err != nil {
		return nil, err
	}
	for i, v := range verdicts {
		blank[i] = v.Blank
	}
	return blank, nil
}

// RemovePages takes the given pages out of the document's PDF. Every page OCR
// left out that stays is taken as the owner saying it has content, which is
// why remove may be empty: keeping such pages is an edit of its own.
func RemovePages(ctx context.Context, app core.App, document *core.Record, remove []int) (Outcome, error) {
	if !isPDF(document) {
		return Outcome{}, ErrNotPDF
	}
	release, err := claim(app, document)
	if err != nil {
		return Outcome{}, err
	}
	defer release()
	workDir, err := os.MkdirTemp("", "lemmary-pageedit-*")
	if err != nil {
		return Outcome{}, err
	}
	defer os.RemoveAll(workDir)

	sourcePath := filepath.Join(workDir, "source.pdf")
	if err := copyStoredFile(app, document, "file", sourcePath); err != nil {
		return Outcome{}, err
	}
	count, err := pdftool.PageCount(ctx, sourcePath)
	if err != nil {
		return Outcome{}, err
	}
	removed, err := checkRemoval(count, remove)
	if err != nil {
		return Outcome{}, err
	}

	record := models.LoadPageRecord(document)
	plan := planRemoval(record, count, removed)
	if len(removed) == 0 && len(plan.confirm) == 0 {
		return Outcome{}, ErrNoPages
	}
	record.Confirmed = append(record.Confirmed, plan.confirm...)

	if len(removed) > 0 {
		outPath := filepath.Join(workDir, "result.pdf")
		if err := pdftool.Without(ctx, sourcePath, count, removed, outPath); err != nil {
			return Outcome{}, err
		}
		var backupPath string
		if document.GetString("original_file") == "" {
			backupPath = sourcePath
		}
		if err := replaceFile(ctx, app, document, outPath, backupPath); err != nil {
			return Outcome{}, err
		}
		record.Map = plan.kept
	}
	record.Store(document)
	return Outcome{Reread: plan.reread}, save(app, document, plan.reread)
}

// RestoreOriginal puts the as-uploaded file back and forgets the backup.
func RestoreOriginal(ctx context.Context, app core.App, document *core.Record) (Outcome, error) {
	if document.GetString("original_file") == "" {
		return Outcome{}, ErrNoOriginal
	}
	release, err := claim(app, document)
	if err != nil {
		return Outcome{}, err
	}
	defer release()
	workDir, err := os.MkdirTemp("", "lemmary-pageedit-*")
	if err != nil {
		return Outcome{}, err
	}
	defer os.RemoveAll(workDir)

	originalPath := filepath.Join(workDir, "original.pdf")
	if err := copyStoredFile(app, document, "original_file", originalPath); err != nil {
		return Outcome{}, err
	}
	count, err := pdftool.PageCount(ctx, originalPath)
	if err != nil {
		return Outcome{}, err
	}

	record := models.LoadPageRecord(document)
	reread := restoreNeedsReread(record, count)
	if err := replaceFile(ctx, app, document, originalPath, ""); err != nil {
		return Outcome{}, err
	}
	document.Set("original_file", nil)
	record.Map = nil
	record.Store(document)
	return Outcome{Reread: reread}, save(app, document, reread)
}

// checkRemoval validates pages against the file and returns them without
// repeats.
func checkRemoval(count int, remove []int) ([]int, error) {
	var removed []int
	for _, page := range remove {
		if page < 1 || page > count {
			return nil, ErrPageOutOfDoc
		}
		if !slices.Contains(removed, page) {
			removed = append(removed, page)
		}
	}
	if len(removed) == count {
		return nil, ErrAllPages
	}
	return removed, nil
}

type removalPlan struct {
	// kept holds the uploaded numbers of the pages that stay, in order.
	kept []int
	// confirm holds the uploaded numbers of pages found blank or left out by
	// OCR that stay.
	confirm []int
	// reread is whether the OCR text no longer matches the pages that stay.
	reread bool
}

// planRemoval works out what removing the given pages of the current file
// does to its OCR text. The text covers every page OCR did not leave out, so
// it still holds when only left-out pages go; a removed page that was read
// put text in it that has to come out, and a left-out page that stays has to
// be read. Any page found blank that stays is confirmed, which ends the
// suggestion to remove it.
func planRemoval(record models.PageRecord, count int, removed []int) removalPlan {
	var plan removalPlan
	for page := 1; page <= count; page++ {
		uploaded := record.Uploaded(page)
		if slices.Contains(removed, page) {
			if !record.IsSkipped(uploaded) {
				plan.reread = true
			}
			continue
		}
		plan.kept = append(plan.kept, uploaded)
		if record.IsSkipped(uploaded) {
			plan.reread = true
		}
		if (record.IsSkipped(uploaded) || record.IsBlank(uploaded)) && !record.IsConfirmed(uploaded) {
			plan.confirm = append(plan.confirm, uploaded)
		}
	}
	return plan
}

// restoreNeedsReread reports whether bringing back the upload's count pages
// adds any page the OCR text does not already account for: one that was read
// before it was removed, or one never measured at all.
func restoreNeedsReread(record models.PageRecord, count int) bool {
	if len(record.Map) == 0 {
		return true
	}
	for page := 1; page <= count; page++ {
		if !slices.Contains(record.Map, page) && !record.IsSkipped(page) {
			return true
		}
	}
	return false
}

// replaceFile sets path as the document's file, with a fresh checksum and
// first-page preview, and backupPath, when given, as original_file. Saving is
// the caller's.
func replaceFile(ctx context.Context, app core.App, document *core.Record, path, backupPath string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
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
	// Made here rather than by a queued preview step: an edit that leaves the
	// text alone queues nothing, and a job would also settle the document's
	// status, taking it out of the Inbox.
	previewFile, err := preview.GenerateFirstPagePNG(ctx, path)
	if err != nil {
		return err
	}
	if backupPath != "" {
		backup, err := os.ReadFile(backupPath)
		if err != nil {
			return err
		}
		original, err := filesystem.NewFileFromBytes(backup, name)
		if err != nil {
			return err
		}
		document.Set("original_file", original)
	}
	document.Set("file", file)
	document.Set("checksum", checksum)
	document.Set("preview", previewFile)
	return nil
}

// save stores the document and, when its text has to be read again, queues
// that. One transaction for the reason reprocess.queueOne gives: the job must
// not become visible before the pending status lands.
func save(app core.App, document *core.Record, reread bool) error {
	if !reread {
		return app.Save(document)
	}
	document.Set("processing_status", models.DocStatusPending)
	return app.RunInTransaction(func(txApp core.App) error {
		if err := txApp.Save(document); err != nil {
			return err
		}
		_, err := worker.Enqueue(txApp, document.Id, rereadSteps, []string{models.StepOCR}, config.Overrides{})
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

// GuardPageRecord puts back whatever page record a records API request
// carried: the server alone keeps it in step with the file. A create, or an
// update that replaces the file, starts an empty record, because there is no
// earlier file for it to number.
func GuardPageRecord(e *core.RecordRequestEvent) error {
	fresh := e.Record.IsNew() || len(e.Record.GetUnsavedFiles("file")) > 0
	for _, field := range models.PageRecordFields {
		if fresh {
			e.Record.Set(field, nil)
		} else {
			e.Record.Set(field, e.Record.Original().Get(field))
		}
	}
	return e.Next()
}
