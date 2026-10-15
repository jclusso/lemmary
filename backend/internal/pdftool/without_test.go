package pdftool

import (
	"context"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestWithoutDropsThePages(t *testing.T) {
	requirePoppler(t, "pdfinfo", "pdfseparate", "pdfunite", "pdftotext")

	source := writePDF(t, 5)
	out := filepath.Join(t.TempDir(), "out.pdf")
	if err := Without(context.Background(), source, 5, []int{2, 4}, out); err != nil {
		t.Fatal(err)
	}
	texts, err := AllPagesText(context.Background(), out, 3)
	if err != nil {
		t.Fatal(err)
	}
	for i, want := range []string{"Page 1", "Page 3", "Page 5"} {
		if !strings.Contains(texts[i], want) {
			t.Fatalf("page %d = %q, want %s", i+1, texts[i], want)
		}
	}
}

func TestKeptRanges(t *testing.T) {
	got, err := keptRanges(7, []int{1, 4, 5, 7})
	if err != nil {
		t.Fatal(err)
	}
	if want := []pageRange{{2, 3}, {6, 6}}; !slices.Equal(got, want) {
		t.Fatalf("ranges = %v, want %v", got, want)
	}
	for name, remove := range map[string][]int{
		"nothing":     nil,
		"every page":  {1, 2, 2},
		"a page past": {3},
		"page zero":   {0},
	} {
		if _, err := keptRanges(2, remove); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}
