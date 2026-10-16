import type { Language } from '../i18n'
import { LANGUAGE_NAMES } from './languageNames'

export type LanguageOption = { code: string; name: string }

/** Every ISO 639-1 language, named and sorted in the interface language. */
export function languageOptions(locale: Language): LanguageOption[] {
  const collator = new Intl.Collator(locale)
  return Object.entries(LANGUAGE_NAMES[locale])
    .map(([code, name]) => ({ code, name }))
    .sort((a, b) => collator.compare(a.name, b.name))
}

/** The language's name in the interface language, or the code when it is not one we list. */
export function languageName(code: string, locale: Language) {
  return LANGUAGE_NAMES[locale][code] ?? code
}

/**
 * Whether a document's OCR text is worth offering in the result language. A
 * document extracted before languages were recorded has none, so it is offered.
 */
export function offersTranslation(documentLanguage: string | undefined, resultLanguage: string | undefined) {
  return Boolean(resultLanguage) && documentLanguage !== resultLanguage
}
