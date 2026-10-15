package config

import (
	"slices"
	"testing"
)

func TestParseSensitiveFoldersSplitsTrimsAndDedupes(t *testing.T) {
	got := ParseSensitiveFolders(" sensitive, Medical\n/Tax/ ,SENSITIVE,, a/b ")
	want := []string{"sensitive", "Medical", "Tax"}
	if !slices.Equal(got, want) {
		t.Fatalf("ParseSensitiveFolders = %q, want %q", got, want)
	}
	if got := ParseSensitiveFolders("   "); got != nil {
		t.Fatalf("blank list = %q, want none", got)
	}
}

func TestInSensitiveFolderMatchesDirectoriesAtAnyDepth(t *testing.T) {
	folders := []string{"sensitive", "Medical"}
	cases := map[string]bool{
		"sensitive/w2.pdf":            true,
		"Sensitive/w2.pdf":            true,
		"Taxes/2024/medical/bill.pdf": true,
		"Taxes/2024/invoice.pdf":      false,
		"sensitive.pdf":               false,
		"insensitive/note.pdf":        false,
		`sensitive\w2.pdf`:            true,
	}
	for rel, want := range cases {
		if got := InSensitiveFolder(rel, folders); got != want {
			t.Errorf("InSensitiveFolder(%q) = %v, want %v", rel, got, want)
		}
	}
	if InSensitiveFolder("sensitive/w2.pdf", nil) {
		t.Fatal("an empty folder list must mark nothing")
	}
}
