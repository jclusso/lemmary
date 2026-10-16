package langcode

import "testing"

func TestParse(t *testing.T) {
	cases := []struct {
		in   string
		want string
		ok   bool
	}{
		{"en", "en", true},
		{" EN ", "en", true},
		{"en-US", "en", true},
		{"eng", "en", true},
		{"deu", "de", true},
		{"", "", false},
		{"english", "", false},
		{"zz", "", false},
		{"und", "", false},
		{"fil", "", false},
	}
	for _, tc := range cases {
		got, ok := Parse(tc.in)
		if got != tc.want || ok != tc.ok {
			t.Fatalf("Parse(%q)=%q,%v, want %q,%v", tc.in, got, ok, tc.want, tc.ok)
		}
	}
}

func TestParseList(t *testing.T) {
	codes, invalid := ParseList(" de, EN ,en-US,, english, uk ")
	if codes != "de,en,uk" {
		t.Fatalf("codes = %q, want de,en,uk", codes)
	}
	if len(invalid) != 1 || invalid[0] != "english" {
		t.Fatalf("invalid = %v, want [english]", invalid)
	}
	if codes, invalid := ParseList(""); codes != "" || invalid != nil {
		t.Fatalf("empty list = %q %v", codes, invalid)
	}
}

func TestEnglishName(t *testing.T) {
	cases := map[string]string{"en": "English", "de": "German", "es": "Spanish", "qq": "qq"}
	for code, want := range cases {
		if got := EnglishName(code); got != want {
			t.Fatalf("EnglishName(%q)=%q, want %q", code, got, want)
		}
	}
}
