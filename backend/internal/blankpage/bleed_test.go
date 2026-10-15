package blankpage

import (
	"image"
	"image/color"
	"image/draw"
	"testing"
)

func grayPage(paper uint8) *image.Gray {
	img := image.NewGray(image.Rect(0, 0, 566, 800))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: color.Gray{Y: paper}}, image.Point{}, draw.Src)
	return img
}

func block(img *image.Gray, r image.Rectangle, level uint8) {
	draw.Draw(img, r, &image.Uniform{C: color.Gray{Y: level}}, image.Point{}, draw.Src)
}

// invoice lays out a front page: a header block on the left, lines of text
// and a table rule, deliberately lopsided so its mirror image differs.
func invoice(img *image.Gray, level uint8, mirrored bool) {
	at := func(r image.Rectangle) image.Rectangle {
		if !mirrored {
			return r
		}
		w := img.Bounds().Dx()
		return image.Rect(w-r.Max.X, r.Min.Y, w-r.Min.X, r.Max.Y)
	}
	block(img, at(image.Rect(50, 60, 250, 160)), level)
	for y := 200; y < 420; y += 18 {
		block(img, at(image.Rect(50, y, 330, y+4)), level)
	}
	block(img, at(image.Rect(40, 450, 520, 456)), level)
	for y := 480; y < 700; y += 24 {
		block(img, at(image.Rect(300, y, 510, y+4)), level)
	}
}

func TestShowsThrough(t *testing.T) {
	front := grayPage(250)
	invoice(front, 30, false)
	frontPage := measurePage(front)

	// The front seen through the sheet: mirrored and mostly a light grey, with
	// the table rule pressed through dark enough to pass for print.
	bleed := grayPage(250)
	invoice(bleed, 190, true)
	block(bleed, image.Rect(46, 450, 526, 456), 60)
	if !showsThrough(measurePage(bleed), frontPage) {
		t.Fatalf("heavy bleed-through not recognised: back %+v front %+v", measurePage(bleed).ink, frontPage.ink)
	}

	// The same form printed again, as the next invoice page would be: it
	// matches mirrored too, but it is as dark as the front.
	nextPage := grayPage(250)
	invoice(nextPage, 30, true)
	if showsThrough(measurePage(nextPage), frontPage) {
		t.Fatal("a printed page was taken for bleed-through")
	}

	// Faint print laid out unlike the front.
	notes := grayPage(250)
	for y := 100; y < 300; y += 30 {
		block(notes, image.Rect(120, y, 260, y+3), 120)
	}
	if showsThrough(measurePage(notes), frontPage) {
		t.Fatal("faint notes unlike the front were taken for bleed-through")
	}

	// The front unmirrored and as faint: a pale copy of the same sheet.
	copyPage := grayPage(250)
	invoice(copyPage, 190, false)
	block(copyPage, image.Rect(40, 450, 520, 456), 60)
	if showsThrough(measurePage(copyPage), frontPage) {
		t.Fatal("an unmirrored copy was taken for bleed-through")
	}
}
