package langdetect

import (
	"strings"
	"testing"
)

const englishBill = `Your monthly statement is ready. Thank you for being a valued customer of our electric company.
This statement covers the service period from March 1 through March 31. The amount due will be drafted from your account on the due date shown below, so there is nothing more you need to do.

Your usage this month was higher than last month because of colder weather. You can review your daily usage, compare it with previous years, and find ways to save energy by signing in to your online account.

If you have questions about this bill, please call customer service between eight in the morning and six in the evening, Monday through Friday. Payments made after the due date may be subject to a late fee.
`

const spanishNotice = `
Para información en español, llame al número de servicio al cliente.
`

const spanishLetter = `Estimado cliente, le escribimos para informarle que su póliza de seguro se renovará automáticamente el próximo mes.
Las condiciones de la cobertura no cambian, pero la prima anual aumentará ligeramente debido a los ajustes del mercado.

Si desea modificar su póliza o tiene alguna pregunta sobre la renovación, comuníquese con nuestro equipo de atención al cliente de lunes a viernes. Agradecemos su confianza y quedamos a su disposición para cualquier consulta.
`

func TestMainLanguageIgnoresAShortNoticeInAnotherLanguage(t *testing.T) {
	code, ok := MainLanguage(englishBill + spanishNotice)
	if !ok || code != "en" {
		t.Fatalf("MainLanguage = %q,%v, want en", code, ok)
	}
}

func TestMainLanguageReadsANonEnglishDocument(t *testing.T) {
	code, ok := MainLanguage(spanishLetter)
	if !ok || code != "es" {
		t.Fatalf("MainLanguage = %q,%v, want es", code, ok)
	}
}

func TestMainLanguageMeasuresTextWithoutParagraphBreaks(t *testing.T) {
	code, ok := MainLanguage(strings.ReplaceAll(englishBill, "\n\n", "\n") + spanishNotice)
	if !ok || code != "en" {
		t.Fatalf("MainLanguage = %q,%v, want en", code, ok)
	}
}

func TestMainLanguageDeclinesShortText(t *testing.T) {
	if code, ok := MainLanguage("Total 42.00 USD\nThank you"); ok {
		t.Fatalf("MainLanguage = %q, want no answer for a receipt-sized text", code)
	}
}

func TestMainLanguageDeclinesWithoutAClearMajority(t *testing.T) {
	if code, ok := MainLanguage(englishBill + "\n" + spanishLetter); ok {
		t.Fatalf("MainLanguage = %q, want no answer for a letter that is 60/40 English and Spanish", code)
	}
}
