package models

import (
	"slices"

	"github.com/pocketbase/pocketbase/core"
)

// The documents columns PageRecord is stored in.
const (
	PageMapField         = "page_map"
	BlankPagesField      = "blank_pages"
	OCRSkippedPagesField = "ocr_skipped_pages"
	ConfirmedPagesField  = "confirmed_pages"
)

// PageRecordFields lists the columns only the server writes; the records API
// keeps them as they were.
var PageRecordFields = []string{PageMapField, BlankPagesField, OCRSkippedPagesField, ConfirmedPagesField}

// PageRecord is what a document remembers about its PDF's pages. Every number
// is the page's number in the file as first uploaded, so the record stays true
// when pages are removed and when the upload is restored.
type PageRecord struct {
	// Map holds the uploaded page number of each page of the current file, in
	// order. Empty while the file is the upload itself.
	Map []int
	// Blank are pages the detect_blank_pages step found blank.
	Blank []int
	// Skipped are pages left out of ocr_text. Kept apart from Blank because
	// the two can disagree: text read before a page was found blank still
	// holds that page.
	Skipped []int
	// Confirmed are pages the owner kept although they looked blank. OCR reads
	// them, and they are not suggested for removal again.
	Confirmed []int
}

func LoadPageRecord(document *core.Record) PageRecord {
	var p PageRecord
	_ = document.UnmarshalJSONField(PageMapField, &p.Map)
	_ = document.UnmarshalJSONField(BlankPagesField, &p.Blank)
	_ = document.UnmarshalJSONField(OCRSkippedPagesField, &p.Skipped)
	_ = document.UnmarshalJSONField(ConfirmedPagesField, &p.Confirmed)
	return p
}

// Store writes the record onto document; saving it is the caller's.
func (p PageRecord) Store(document *core.Record) {
	document.Set(PageMapField, emptyAsNil(p.Map))
	document.Set(BlankPagesField, emptyAsNil(sorted(p.Blank)))
	document.Set(OCRSkippedPagesField, emptyAsNil(sorted(p.Skipped)))
	document.Set(ConfirmedPagesField, emptyAsNil(sorted(p.Confirmed)))
}

// Uploaded maps a 1-based page of the current file to its uploaded number.
func (p PageRecord) Uploaded(page int) int {
	if len(p.Map) == 0 || page < 1 || page > len(p.Map) {
		return page
	}
	return p.Map[page-1]
}

// Current lists the uploaded numbers of a current file of count pages.
func (p PageRecord) Current(count int) []int {
	pages := make([]int, count)
	for i := range pages {
		pages[i] = p.Uploaded(i + 1)
	}
	return pages
}

func (p PageRecord) IsBlank(uploaded int) bool     { return slices.Contains(p.Blank, uploaded) }
func (p PageRecord) IsSkipped(uploaded int) bool   { return slices.Contains(p.Skipped, uploaded) }
func (p PageRecord) IsConfirmed(uploaded int) bool { return slices.Contains(p.Confirmed, uploaded) }

// WithBlank replaces what is known about the current file's pages, given as
// uploaded numbers in current, with blank. Entries for pages no longer in the
// file stay: a restore brings those pages back as they were.
func (p PageRecord) WithBlank(current, blank []int) PageRecord {
	p.Blank = replaceFor(p.Blank, current, blank)
	return p
}

// WithSkipped is WithBlank for the pages left out of ocr_text.
func (p PageRecord) WithSkipped(current, skipped []int) PageRecord {
	p.Skipped = replaceFor(p.Skipped, current, skipped)
	return p
}

func replaceFor(list, current, values []int) []int {
	out := slices.DeleteFunc(slices.Clone(list), func(page int) bool {
		return slices.Contains(current, page)
	})
	return append(out, values...)
}

func sorted(pages []int) []int {
	out := slices.Clone(pages)
	slices.Sort(out)
	return slices.Compact(out)
}

func emptyAsNil(pages []int) any {
	if len(pages) == 0 {
		return nil
	}
	return pages
}
