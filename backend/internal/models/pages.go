package models

import (
	"slices"

	"github.com/pocketbase/pocketbase/core"
)

// The documents columns PageRecord is stored in.
const (
	PageMapField         = "page_map"
	OCRSkippedPagesField = "ocr_skipped_pages"
	ConfirmedPagesField  = "confirmed_pages"
)

// PageRecordFields lists the columns only the server writes; the records API
// keeps them as they were.
var PageRecordFields = []string{PageMapField, OCRSkippedPagesField, ConfirmedPagesField}

// PageRecord is what a document remembers about its PDF's pages. Every number
// is the page's number in the file as first uploaded, so the record stays true
// when pages are removed and when the upload is restored.
type PageRecord struct {
	// Map holds the uploaded page number of each page of the current file, in
	// order. Empty while the file is the upload itself.
	Map []int
	// Skipped are pages that looked blank and were left out of ocr_text.
	Skipped []int
	// Confirmed are pages the owner kept although they looked blank. OCR reads
	// them whatever they look like.
	Confirmed []int
}

func LoadPageRecord(document *core.Record) PageRecord {
	var p PageRecord
	_ = document.UnmarshalJSONField(PageMapField, &p.Map)
	_ = document.UnmarshalJSONField(OCRSkippedPagesField, &p.Skipped)
	_ = document.UnmarshalJSONField(ConfirmedPagesField, &p.Confirmed)
	return p
}

// Store writes the record onto document; saving it is the caller's.
func (p PageRecord) Store(document *core.Record) {
	document.Set(PageMapField, emptyAsNil(p.Map))
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

func (p PageRecord) IsSkipped(uploaded int) bool   { return slices.Contains(p.Skipped, uploaded) }
func (p PageRecord) IsConfirmed(uploaded int) bool { return slices.Contains(p.Confirmed, uploaded) }

// WithSkipped replaces what is known about the current file's pages, given
// as uploaded numbers in current, with skipped. Entries for pages no longer in
// the file stay: a restore brings those pages back as they were.
func (p PageRecord) WithSkipped(current, skipped []int) PageRecord {
	p.Skipped = slices.DeleteFunc(slices.Clone(p.Skipped), func(page int) bool {
		return slices.Contains(current, page)
	})
	p.Skipped = append(p.Skipped, skipped...)
	return p
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
