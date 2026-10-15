package pageedit

import (
	"bytes"
	"errors"
	"image"
	"strconv"
)

var errBadPGM = errors.New("pageedit: not an 8-bit binary PGM")

// decodePGM reads the binary greymap pdftoppm -gray writes. The standard
// library has no decoder for it, and the format is a short text header in
// front of one byte per pixel.
func decodePGM(data []byte) (*image.Gray, error) {
	if !bytes.HasPrefix(data, []byte("P5")) {
		return nil, errBadPGM
	}
	rest := data[2:]
	var fields [3]int
	for i := range fields {
		var token []byte
		token, rest = nextToken(rest)
		n, err := strconv.Atoi(string(token))
		if err != nil || n <= 0 {
			return nil, errBadPGM
		}
		fields[i] = n
	}
	width, height, maxval := fields[0], fields[1], fields[2]
	// One whitespace byte separates the header from the pixels.
	if maxval > 255 || len(rest) < 1+width*height {
		return nil, errBadPGM
	}
	img := image.NewGray(image.Rect(0, 0, width, height))
	copy(img.Pix, rest[1:1+width*height])
	return img, nil
}

// nextToken skips whitespace and # comments and returns the token after them,
// leaving rest at the byte that ended it.
func nextToken(data []byte) (token, rest []byte) {
	i := 0
	for i < len(data) {
		switch {
		case data[i] == '#':
			for i < len(data) && data[i] != '\n' {
				i++
			}
		case isSpace(data[i]):
			i++
		default:
			start := i
			for i < len(data) && !isSpace(data[i]) {
				i++
			}
			return data[start:i], data[i:]
		}
	}
	return nil, nil
}

func isSpace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r'
}
