package pageedit

import (
	"image"
	"image/color"
)

const (
	// blankCoverage is the share of a page's inner area that may be ink before
	// the page stops being suggested as blank. A page holding only a short line
	// of 18pt text sits several times above it.
	blankCoverage = 0.0002
	// inkContrast is how much darker than the paper a pixel must be to count as
	// ink. Text bleeding through from the back of the sheet is a light grey a
	// few dozen levels below the paper, so it falls short of this.
	inkContrast = 100
	// marginShare is trimmed from every edge before measuring: scanner bed
	// edges, shadows and punch holes live there and are not content.
	marginShare = 0.06
)

// IsBlank reports whether a rendered page carries no print beyond bleed-through,
// specks and edge shadows.
func IsBlank(img image.Image) bool {
	return InkCoverage(img) < blankCoverage
}

// InkCoverage is the share of the page's inner area darker than its paper by at
// least inkContrast. The paper is the median level, so tinted paper and a grey
// scanner background are measured against themselves rather than pure white.
func InkCoverage(img image.Image) float64 {
	b := img.Bounds()
	mx := int(float64(b.Dx()) * marginShare)
	my := int(float64(b.Dy()) * marginShare)
	inner := image.Rect(b.Min.X+mx, b.Min.Y+my, b.Max.X-mx, b.Max.Y-my)
	if inner.Empty() {
		return 0
	}

	var histogram [256]int
	for y := inner.Min.Y; y < inner.Max.Y; y++ {
		for x := inner.Min.X; x < inner.Max.X; x++ {
			histogram[luminance(img, x, y)]++
		}
	}

	total := inner.Dx() * inner.Dy()
	paper := median(histogram, total)
	threshold := paper - inkContrast
	if threshold <= 0 {
		return 1
	}
	ink := 0
	for level := range threshold {
		ink += histogram[level]
	}
	return float64(ink) / float64(total)
}

// luminance reads *image.RGBA directly because that is what a pdftoppm PNG
// decodes to, and going through At for every pixel of every page is the
// slowest part of a scan's inspection.
func luminance(img image.Image, x, y int) uint8 {
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
