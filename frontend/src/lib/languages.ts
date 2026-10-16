const ISO_639_1 = (
  'aa ab ae af ak am an ar as av ay az ba be bg bi bm bn bo br bs ca ce ch co cr cs cu cv cy ' +
  'da de dv dz ee el en eo es et eu fa ff fi fj fo fr fy ga gd gl gn gu gv ha he hi ho hr ht ' +
  'hu hy hz ia id ie ig ii ik io is it iu ja jv ka kg ki kj kk kl km kn ko kr ks ku kv kw ky ' +
  'la lb lg li ln lo lt lu lv mg mh mi mk ml mn mr ms mt my na nb nd ne ng nl nn no nr nv ny ' +
  'oc oj om or os pa pi pl ps pt qu rm rn ro ru rw sa sc sd se sg si sk sl sm sn so sq sr ss ' +
  'st su sv sw ta te tg th ti tk tl tn to tr ts tt tw ty ug uk ur uz ve vi vo wa wo xh yi yo ' +
  'za zh zu'
).split(' ')

export type LanguageOption = { code: string; name: string }

/** Every ISO 639-1 language, named and sorted in the reader's locale. */
export function languageOptions(locale: string): LanguageOption[] {
  const names = new Intl.DisplayNames([locale], { type: 'language', fallback: 'code' })
  const collator = new Intl.Collator(locale)
  return ISO_639_1.map((code) => ({ code, name: names.of(code) ?? code })).sort((a, b) =>
    collator.compare(a.name, b.name),
  )
}

/**
 * Whether a document's OCR text is worth offering in the result language. A
 * document extracted before languages were recorded has none, so it is offered.
 */
export function offersTranslation(documentLanguage: string | undefined, resultLanguage: string | undefined) {
  return Boolean(resultLanguage) && documentLanguage !== resultLanguage
}
