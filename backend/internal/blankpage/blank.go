package blankpage

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
)

// inkProfile is what isBlank decides from, measured over the page's inner area.
type inkProfile struct {
	// ink and solid are shares of the inner area.
	ink, solid float64
}

// contentSolid is the near-black share past which a page is printed beyond
// doubt. Printed pages measure from 0.3% (a scanner that renders toner as a
// light grey) to over 2%, and the heaviest bleed-through seen so far 0.5%,
// so only pages above every bleed-through are settled by it.
const contentSolid = 0.015

// isBlank reports whether a page is blank on its own: next to no ink at all,
// or a few faint marks. Faint ink across the whole page is not enough, because
// a page printed in light toner looks exactly like heavy bleed-through; only
// the page before it (showsThrough) or a second opinion can tell them apart.
func isBlank(p inkProfile) bool {
	return p.ink < blankInk || (p.ink < faintInk && p.solid < faintSolid)
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

	ink, solid := 0, 0
	for y := inner.Min.Y; y < inner.Max.Y; y++ {
		for x := inner.Min.X; x < inner.Max.X; x++ {
			level := int(luminance(img, x, y))
			if level >= paper-inkContrast {
				continue
			}
			ink++
			if level < paper-solidContrast {
				solid++
			}
		}
	}
	return inkProfile{
		ink:   float64(ink) / float64(total),
		solid: float64(solid) / float64(total),
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
