// Package langdetect measures which language most of a document's text is
// written in, so extraction can state it rather than leave the model to judge
// from text where a short translated notice weighs as much as the rest.
package langdetect

import (
	"strings"
	"unicode"

	"github.com/abadojack/whatlanggo"

	"lemmary/backend/internal/langcode"
)

const (
	// A chunk is closed at a blank line or once it holds this many letters, so
	// a page without paragraph breaks is still measured piece by piece.
	chunkLetters = 300
	// Shorter chunks say too little for a trigram detector to classify.
	minChunkLetters = 40
	// Below this much classified text there is nothing to measure.
	minTotalLetters = 200
	// The winner's share of the classified letters.
	minShare = 0.6
)

// MainLanguage returns the ISO 639-1 code of the language most of text is
// written in, weighing each chunk by its letters. It reports false when the
// text is too short to measure or no language holds a clear majority.
func MainLanguage(text string) (string, bool) {
	weights := map[string]int{}
	total := 0
	for _, chunk := range chunks(text) {
		letters := countLetters(chunk)
		if letters < minChunkLetters {
			continue
		}
		info := whatlanggo.Detect(chunk)
		if !info.IsReliable() {
			continue
		}
		code, ok := langcode.Parse(info.Lang.Iso6391())
		if !ok {
			continue
		}
		weights[code] += letters
		total += letters
	}
	if total < minTotalLetters {
		return "", false
	}
	best, bestWeight := "", 0
	for code, weight := range weights {
		if weight > bestWeight || (weight == bestWeight && code < best) {
			best, bestWeight = code, weight
		}
	}
	if float64(bestWeight) < minShare*float64(total) {
		return "", false
	}
	return best, true
}

func chunks(text string) []string {
	var out []string
	var current strings.Builder
	letters := 0
	flush := func() {
		if current.Len() > 0 {
			out = append(out, current.String())
		}
		current.Reset()
		letters = 0
	}
	for line := range strings.Lines(text) {
		if strings.TrimSpace(line) == "" {
			flush()
			continue
		}
		current.WriteString(line)
		letters += countLetters(line)
		if letters >= chunkLetters {
			flush()
		}
	}
	flush()
	return out
}

func countLetters(s string) int {
	n := 0
	for _, r := range s {
		if unicode.IsLetter(r) {
			n++
		}
	}
	return n
}
