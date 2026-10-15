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

// Verdict is what the local check makes of one page.
type Verdict struct {
	Blank bool
	// Sure is false for a page between nearly no ink and plenty of print,
	// where heavy bleed-through and faint real print look alike. Blank is the
	// best local guess there; a second opinion may overturn it.
	Sure bool
}

// Detect renders every page of pdfPath and judges, in page order, whether
// each looks blank. A page also counts as blank when it only shows the page
// before it through the paper: a duplex scan puts each sheet's back right
// after its front, and heavy bleed-through can carry as much ink as faint
// print, so the front is what tells them apart.
func Detect(ctx context.Context, pdfPath string, pageCount int) ([]Verdict, error) {
	dir, err := os.MkdirTemp("", "lemmary-blankpage-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)

	paths, err := pdftool.RenderPagesAs(ctx, pdfPath, filepath.Join(dir, "pages"), pdftool.Gray, RenderEdge, pageCount)
	if err != nil {
		return nil, err
	}
	verdicts := make([]Verdict, len(paths))
	var previous measuredPage
	for i, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		img, err := decodePGM(data)
		if err != nil {
			return nil, fmt.Errorf("blankpage: page %d: %w", i+1, err)
		}
		current := measurePage(img)
		blank := isBlank(current.ink) || (i > 0 && !verdicts[i-1].Blank && showsThrough(current, previous))
		verdicts[i] = Verdict{
			Blank: blank,
			Sure:  isBlank(current.ink) || current.ink.solid >= contentSolid,
		}
		previous = current
	}
	return verdicts, nil
}
