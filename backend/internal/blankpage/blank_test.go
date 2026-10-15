package blankpage

import (
	"image"
	"image/color"
	"image/draw"
	"testing"
)

func page(paper uint8) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, 566, 800))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: color.RGBA{R: paper, G: paper, B: paper, A: 0xff}}, image.Point{}, draw.Src)
	return img
}

func fill(img *image.RGBA, r image.Rectangle, level uint8) {
	draw.Draw(img, r, &image.Uniform{C: color.RGBA{R: level, G: level, B: level, A: 0xff}}, image.Point{}, draw.Src)
}

// Lines of text across the page, the way a whole paragraph lays down ink.
func paragraph(img *image.RGBA, level uint8) {
	for y := 100; y < 700; y += 20 {
		fill(img, image.Rect(60, y, 500, y+3), level)
	}
}

func TestIsBlank(t *testing.T) {
	cases := []struct {
		name  string
		build func() *image.RGBA
		blank bool
	}{
		{"white paper", func() *image.RGBA { return page(250) }, true},
		{"text bleeding through", func() *image.RGBA {
			img := page(245)
			paragraph(img, 200)
			return img
		}, true},
		{"bleed-through on tinted paper", func() *image.RGBA {
			img := page(215)
			paragraph(img, 170)
			return img
		}, true},
		{"scanner edges and a punch hole", func() *image.RGBA {
			img := page(250)
			fill(img, image.Rect(0, 0, 566, 20), 0)
			fill(img, image.Rect(0, 0, 15, 800), 0)
			fill(img, image.Rect(10, 380, 30, 400), 0)
			return img
		}, true},
		{"a few specks of dust", func() *image.RGBA {
			img := page(250)
			fill(img, image.Rect(200, 300, 203, 303), 0)
			fill(img, image.Rect(400, 500, 402, 502), 0)
			return img
		}, true},
		{"a stamp pressed through from the front", func() *image.RGBA {
			img := page(250)
			fill(img, image.Rect(200, 150, 240, 152), 130)
			fill(img, image.Rect(200, 180, 240, 182), 130)
			fill(img, image.Rect(200, 150, 202, 182), 130)
			fill(img, image.Rect(238, 150, 240, 182), 130)
			return img
		}, true},
		{"heavy bleed-through across the sheet", func() *image.RGBA {
			img := page(250)
			paragraph(img, 140)
			fill(img, image.Rect(300, 400, 304, 404), 40)
			return img
		}, true},
		{"one dark line on an empty page", func() *image.RGBA {
			img := page(250)
			fill(img, image.Rect(80, 300, 200, 303), 20)
			return img
		}, false},
		{"one short printed word", func() *image.RGBA {
			img := page(250)
			fill(img, image.Rect(70, 70, 130, 82), 20)
			return img
		}, false},
		{"a page of text", func() *image.RGBA {
			img := page(250)
			paragraph(img, 30)
			return img
		}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			img := tc.build()
			if got := IsBlank(img); got != tc.blank {
				t.Fatalf("IsBlank = %v, want %v (%+v)", got, tc.blank, measureInk(img))
			}
		})
	}
}
