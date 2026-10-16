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

func TestEnglishName(t *testing.T) {
	cases := map[string]string{"en": "English", "de": "German", "es": "Spanish", "qq": "qq"}
	for code, want := range cases {
		if got := EnglishName(code); got != want {
			t.Fatalf("EnglishName(%q)=%q, want %q", code, got, want)
		}
	}
}
