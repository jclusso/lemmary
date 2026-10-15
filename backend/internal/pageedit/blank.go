package pageedit

import (
	"image"
	"image/color"
)

const (
	// inkContrast is how much darker than the paper a pixel must be to count as
	// ink. Light bleed-through is a grey a few dozen levels below the paper and
	// falls short of it.
	inkContrast = 100
	// solidContrast marks near-black ink. Print reaches it; text seen through
	// the sheet rarely does.
	solidContrast = 160
	// marginShare is trimmed from every edge before measuring: scanner bed
	// edges, shadows and punch holes live there and are not content.
	marginShare = 0.06

	// A page with less ink than this is blank. A page holding only a short line
	// of 18pt text sits several times above it.
	blankInk = 0.0002
	// A page with little ink, almost none of it near-black, is blank: a stamp
	// or pen mark pressed through from the front.
	faintInk   = 0.001
	faintSolid = 0.0001
	// A page whose ink reaches over bleedSpread of the grid while staying this
	// short of near-black is heavy bleed-through: the whole of the other side
	// showing faintly. Real print on a mostly empty page is the reverse,
	// near-black and confined to a few cells.
	bleedSolid  = 0.0025
	bleedSpread = 0.4
	gridCols    = 12
	gridRows    = 16
	// cellInk is the ink, in pixels, that makes a grid cell count as inked, so
	// a speck or two does not.
	cellInk = 3
)

// inkProfile is what IsBlank decides from, measured over the page's inner area.
type inkProfile struct {
	// ink and solid are shares of the inner area.
	ink, solid float64
	// spread is the share of grid cells holding ink.
	spread float64
}

// IsBlank reports whether a rendered page carries no print beyond bleed-through,
// specks and edge shadows.
func IsBlank(img image.Image) bool {
	p := measureInk(img)
	switch {
	case p.ink < blankInk:
		return true
	case p.ink < faintInk && p.solid < faintSolid:
		return true
	default:
		return p.solid < bleedSolid && p.spread >= bleedSpread
	}
}

// measureInk compares every pixel with the paper, which is the median level,
// so tinted paper and a grey scanner background are measured against
// themselves rather than pure white.
func measureInk(img image.Image) inkProfile {
	b := img.Bounds()
	mx := int(float64(b.Dx()) * marginShare)
	my := int(float64(b.Dy()) * marginShare)
	inner := image.Rect(b.Min.X+mx, b.Min.Y+my, b.Max.X-mx, b.Max.Y-my)
	if inner.Empty() {
		return inkProfile{}
	}

	var histogram [256]int
	for y := inner.Min.Y; y < inner.Max.Y; y++ {
		for x := inner.Min.X; x < inner.Max.X; x++ {
			histogram[luminance(img, x, y)]++
		}
	}
	total := inner.Dx() * inner.Dy()
	paper := median(histogram, total)

	var cells [gridRows][gridCols]int
	ink, solid := 0, 0
	for y := inner.Min.Y; y < inner.Max.Y; y++ {
		row := (y - inner.Min.Y) * gridRows / inner.Dy()
		for x := inner.Min.X; x < inner.Max.X; x++ {
			level := int(luminance(img, x, y))
			if level >= paper-inkContrast {
				continue
			}
			ink++
			cells[row][(x-inner.Min.X)*gridCols/inner.Dx()]++
			if level < paper-solidContrast {
				solid++
			}
		}
	}

	inked := 0
	for _, row := range cells {
		for _, count := range row {
			if count >= cellInk {
				inked++
			}
		}
	}
	return inkProfile{
		ink:    float64(ink) / float64(total),
		solid:  float64(solid) / float64(total),
		spread: float64(inked) / float64(gridRows*gridCols),
	}
}

// luminance reads *image.Gray and *image.RGBA directly, because going through
// At for every pixel of every page is the slowest part of a scan's inspection.
func luminance(img image.Image, x, y int) uint8 {
	if gray, ok := img.(*image.Gray); ok {
		return gray.Pix[gray.PixOffset(x, y)]
	}
	if rgba, ok := img.(*image.RGBA); ok {
		i := rgba.PixOffset(x, y)
		p := rgba.Pix[i : i+3 : i+3]
		return uint8((299*int(p[0]) + 587*int(p[1]) + 114*int(p[2])) / 1000)
	}
	return color.GrayModel.Convert(img.At(x, y)).(color.Gray).Y
}

func median(histogram [256]int, total int) int {
	seen := 0
	for level, count := range histogram {
		seen += count
		if seen*2 >= total {
			return level
		}
	}
	return 255
}
