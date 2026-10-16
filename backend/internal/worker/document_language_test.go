package worker

import (
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/core"

	"lemmary/backend/internal/models"
)

func languageTestDocument(source, language string) *core.Record {
	documents := core.NewBaseCollection("documents")
	documents.Fields.Add(
		&core.TextField{Name: "metadata_source"},
		&core.TextField{Name: "language"},
	)
	record := core.NewRecord(documents)
	record.Set("metadata_source", source)
	record.Set("language", language)
	return record
}

func TestDocumentLanguage(t *testing.T) {
	english := strings.Repeat("Your statement is ready and the amount due will be drafted from your account on the due date. ", 6)

	cases := []struct {
		name     string
		document *core.Record
		text     string
		want     string
	}{
		{"measured", languageTestDocument("gpt", ""), english, "en"},
		{"owner's choice outranks the measurement", languageTestDocument(models.MetadataSourceUser, "es"), english, "es"},
		{"a model's earlier answer is measured again", languageTestDocument("gpt", "es"), english, "en"},
		{"too short to measure", languageTestDocument("gpt", ""), "Total 42.00", ""},
	}
	for _, tc := range cases {
		if got := documentLanguage(tc.document, tc.text); got != tc.want {
			t.Fatalf("%s: documentLanguage = %q, want %q", tc.name, got, tc.want)
		}
	}
}
