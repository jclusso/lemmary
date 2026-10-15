package blankpage

import (
	"image"
	"math"
)

const (
	// gridEdge is the longest edge of the darkness grid two pages are
	// compared on: coarse enough that a sheet scanned a few millimetres off
	// on each side still lines up within maxShift cells.
	gridEdge = 120
	maxShift = 8
	// A back showing its front through the paper matches the mirrored front
	// at least this well...
	mirroredMatch = 0.45
	// ...and clearly better than the front as printed. The margin is what
	// tells bleed-through from two pages of the same form, which match
	// each other whichever way round they are laid.
	mirrorMargin = 0.2
	// Bleed-through is the front's print seen through paper, so it carries
	// a fraction of the front's ink. A real page that happens to line up
	// with its mirrored neighbour is as dark as that neighbour.
	bleedInkShare   = 0.35
	bleedSolidShare = 0.25
)

// measuredPage is what Detect keeps of a rendered page.
type measuredPage struct {
	ink  inkProfile
	grid darknessGrid
}

func measurePage(img *image.Gray) measuredPage {
	return measuredPage{ink: measureInk(img), grid: newDarknessGrid(img)}
}

// showsThrough reports whether back carries nothing but front seen through the
// sheet: a mirror image of it, and a faint one.
func showsThrough(back, front measuredPage) bool {
	if front.ink.ink == 0 || front.ink.solid == 0 {
		return false
	}
	if back.ink.ink > bleedInkShare*front.ink.ink || back.ink.solid > bleedSolidShare*front.ink.solid {
		return false
	}
	mirrored := back.grid.match(front.grid, true)
	return mirrored >= mirroredMatch && mirrored-back.grid.match(front.grid, false) >= mirrorMargin
}

// darknessGrid is a page shrunk to cells holding how far below the paper the
// page is on average there.
type darknessGrid struct {
	w, h  int
	cells []float64
}

func newDarknessGrid(img *image.Gray) darknessGrid {
	b := img.Bounds()
	var histogram [256]int
	for _, v := range img.Pix {
		histogram[v]++
	}
	paper := median(histogram, len(img.Pix))

	scale := float64(gridEdge) / float64(max(b.Dx(), b.Dy()))
	g := darknessGrid{w: max(1, int(float64(b.Dx())*scale)), h: max(1, int(float64(b.Dy())*scale))}
	g.cells = make([]float64, g.w*g.h)
	counts := make([]float64, len(g.cells))
	for y := b.Min.Y; y < b.Max.Y; y++ {
		cy := min(g.h-1, int(float64(y-b.Min.Y)*scale))
		for x := b.Min.X; x < b.Max.X; x++ {
			cell := cy*g.w + min(g.w-1, int(float64(x-b.Min.X)*scale))
			g.cells[cell] += float64(max(0, paper-int(img.Pix[img.PixOffset(x, y)])))
			counts[cell]++
		}
	}
	for i := range g.cells {
		if counts[i] > 0 {
			g.cells[i] /= counts[i]
		}
	}
	return g
}

// match is the best correlation of g with other, laid left-to-right reversed
// when mirrored, over shifts of up to maxShift cells either way.
func (g darknessGrid) match(other darknessGrid, mirrored bool) float64 {
	best := -1.0
	for dy := -maxShift; dy <= maxShift; dy++ {
		for dx := -maxShift; dx <= maxShift; dx++ {
			if r, ok := g.correlate(other, mirrored, dx, dy); ok && r > best {
				best = r
			}
		}
	}
	return best
}

func (g darknessGrid) correlate(other darknessGrid, mirrored bool, dx, dy int) (float64, bool) {
	var n, sumA, sumB, sumAA, sumBB, sumAB float64
	for y := range g.h {
		oy := y + dy
		if oy < 0 || oy >= other.h {
			continue
		}
		for x := range g.w {
			ox := x + dx
			if mirrored {
				ox = other.w - 1 - x + dx
			}
			if ox < 0 || ox >= other.w {
				continue
			}
			a, b := g.cells[y*g.w+x], other.cells[oy*other.w+ox]
			n++
			sumA += a
			sumB += b
			sumAA += a * a
			sumBB += b * b
			sumAB += a * b
		}
	}
	if n == 0 {
		return 0, false
	}
	varA := sumAA/n - (sumA/n)*(sumA/n)
	varB := sumBB/n - (sumB/n)*(sumB/n)
	if varA <= 0 || varB <= 0 {
		return 0, false
	}
	return (sumAB/n - (sumA/n)*(sumB/n)) / math.Sqrt(varA*varB), true
}
