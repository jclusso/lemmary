package blankpage

import "testing"

func TestDecodePGM(t *testing.T) {
	data := append([]byte("P5\n# from pdftoppm\n3 2\n255\n"), 0, 10, 20, 30, 40, 250)
	img, err := decodePGM(data)
	if err != nil {
		t.Fatal(err)
	}
	if b := img.Bounds(); b.Dx() != 3 || b.Dy() != 2 {
		t.Fatalf("bounds = %v, want 3x2", b)
	}
	if got := img.GrayAt(2, 1).Y; got != 250 {
		t.Fatalf("pixel (2,1) = %d, want 250", got)
	}
}

func TestDecodePGMRefuses(t *testing.T) {
	for name, data := range map[string][]byte{
		"a PNG":           []byte("\x89PNG\r\n"),
		"a short body":    append([]byte("P5 3 2 255\n"), 1, 2),
		"16-bit samples":  append([]byte("P5 1 1 65535\n"), 0, 0),
		"a missing field": []byte("P5 3\n"),
	} {
		if _, err := decodePGM(data); err == nil {
			t.Errorf("%s: decoded", name)
		}
	}
}
