// Package langcode reads ISO 639-1 language codes, the form every stored
// language in the app takes so that two of them can be compared.
package langcode

import (
	"strings"

	"golang.org/x/text/language"
	"golang.org/x/text/language/display"
)

// Parse reads raw as a language and returns its lowercase ISO 639-1 code:
// "EN", "en-US" and "eng" all give "en". It reports false for empty input and
// for anything that names no language with a two-letter code.
func Parse(raw string) (string, bool) {
	// Raw, because the default canonicalization turns "tl" into "fil", which
	// has no two-letter code.
	tag, err := language.Raw.Parse(strings.TrimSpace(raw))
	if err != nil {
		return "", false
	}
	base, confidence := tag.Base()
	code := base.String()
	if confidence != language.Exact || len(code) != 2 {
		return "", false
	}
	return code, true
}

// EnglishName is the language's English name, which a model follows more
// reliably than a bare code. An unknown code comes back unchanged.
func EnglishName(code string) string {
	tag, err := language.Parse(code)
	if err != nil {
		return code
	}
	if name := display.English.Languages().Name(tag); name != "" {
		return name
	}
	return code
}
