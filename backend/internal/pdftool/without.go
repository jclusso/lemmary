package pdftool

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

type pageRange struct{ from, to int }

// Without writes pdfPath, which has pageCount pages, to outPath minus the
// 1-based pages in remove. At least one page has to stay.
func Without(ctx context.Context, pdfPath string, pageCount int, remove []int, outPath string) error {
	ranges, err := keptRanges(pageCount, remove)
	if err != nil {
		return err
	}
	if len(ranges) == 1 {
		return ExtractRange(ctx, pdfPath, ranges[0].from, ranges[0].to, outPath)
	}

	workDir, err := os.MkdirTemp("", "lemmary-pdfwithout-*")
	if err != nil {
		return fmt.Errorf("pdftool: temp dir: %w", err)
	}
	defer os.RemoveAll(workDir)

	parts := make([]string, 0, len(ranges))
	for i, r := range ranges {
		part := filepath.Join(workDir, fmt.Sprintf("part-%d.pdf", i+1))
		if err := ExtractRange(ctx, pdfPath, r.from, r.to, part); err != nil {
			return err
		}
		parts = append(parts, part)
	}
	return Merge(ctx, outPath, parts...)
}

// keptRanges turns the pages to remove into the runs of pages that stay.
func keptRanges(pageCount int, remove []int) ([]pageRange, error) {
	removed := make(map[int]bool, len(remove))
	for _, page := range remove {
		if page < 1 || page > pageCount {
			return nil, fmt.Errorf("pdftool: page %d is outside 1-%d", page, pageCount)
		}
		removed[page] = true
	}
	if len(removed) == 0 || len(removed) == pageCount {
		return nil, fmt.Errorf("pdftool: removing %d of %d pages", len(removed), pageCount)
	}

	var ranges []pageRange
	for page := 1; page <= pageCount; page++ {
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
