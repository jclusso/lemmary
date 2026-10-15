// Package blankpage tells which pages of a scan carry no print: nothing at all,
// or only what shows through from the other side of the sheet.
package blankpage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"lemmary/backend/internal/pdftool"
)

// RenderEdge is large enough that a short line of text is still several
// pixels of solid ink when a page is measured.
const RenderEdge = 800

// Detect renders every page of pdfPath and reports, in page order, whether
// each looks blank.
func Detect(ctx context.Context, pdfPath string, pageCount int) ([]bool, error) {
	dir, err := os.MkdirTemp("", "lemmary-blankpage-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)

	paths, err := pdftool.RenderPagesAs(ctx, pdfPath, filepath.Join(dir, "pages"), pdftool.Gray, RenderEdge, pageCount)
	if err != nil {
		return nil, err
	}
	blank := make([]bool, len(paths))
	for i, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		img, err := decodePGM(data)
		if err != nil {
			return nil, fmt.Errorf("blankpage: page %d: %w", i+1, err)
		}
		blank[i] = IsBlank(img)
	}
	return blank, nil
}
