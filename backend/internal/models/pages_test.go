package models

import (
	"slices"
	"testing"

	"github.com/pocketbase/pocketbase/core"
)

func TestPageRecordUploaded(t *testing.T) {
	if got := (PageRecord{}).Uploaded(3); got != 3 {
		t.Fatalf("an unedited file maps page 3 to %d", got)
	}
	if got := (PageRecord{Map: []int{1, 3, 5}}).Uploaded(2); got != 3 {
		t.Fatalf("page 2 of 1,3,5 maps to %d, want 3", got)
	}
}

// What is known about pages no longer in the file survives a new OCR run over
// the pages that are.
func TestPageRecordWithSkipped(t *testing.T) {
	p := PageRecord{Map: []int{1, 3, 5}, Skipped: []int{2, 3, 6}}
	got := p.WithSkipped([]int{1, 3, 5}, []int{5}).Skipped
	slices.Sort(got)
	if want := []int{2, 5, 6}; !slices.Equal(got, want) {
		t.Fatalf("skipped = %v, want %v", got, want)
	}
}

// A detection that found nothing is still a detection: stored as an empty
// list, not as no value, so the editor trusts it instead of guessing again.
func TestPageRecordKnowsAnEmptyDetection(t *testing.T) {
	docs := core.NewBaseCollection("documents")
	for _, name := range PageRecordFields {
		docs.Fields.Add(&core.JSONField{Name: name})
	}
	doc := core.NewRecord(docs)

	PageRecord{}.Store(doc)
	if LoadPageRecord(doc).Detected() {
		t.Fatal("a record never detected reads as detected")
	}
	PageRecord{}.WithBlank([]int{1, 2}, nil).Store(doc)
	if !LoadPageRecord(doc).Detected() {
		t.Fatal("a detection that found nothing reads as never run")
	}
}
