import { describe, expect, it } from 'vitest'
import { languageName, languageOptions, offersTranslation } from './languages'

describe('languageOptions', () => {
  it('names every code in each interface language', () => {
    for (const locale of ['en', 'de', 'ru'] as const) {
      const options = languageOptions(locale)
      expect(options.length).toBe(183)
      expect(options.filter((option) => !option.name || option.name === option.code)).toEqual([])
    }
    expect(languageOptions('en').find((option) => option.code === 'rn')?.name).toBe('Rundi')
    expect(languageOptions('de').find((option) => option.code === 'es')?.name).toBe('Spanisch')
  })

  it('sorts by name rather than by code', () => {
    const names = languageOptions('en').map((option) => option.name)
    expect(names).toEqual([...names].sort((a, b) => a.localeCompare(b, 'en')))
  })
})

describe('languageName', () => {
  it('names a listed code and passes anything else through', () => {
    expect(languageName('es', 'ru')).toBe('Испанский')
    expect(languageName('xx', 'en')).toBe('xx')
  })
})

describe('offersTranslation', () => {
  it('is off without a result language', () => {
    expect(offersTranslation('es', '')).toBe(false)
    expect(offersTranslation('es', undefined)).toBe(false)
  })

  it('is off for a document already in the result language', () => {
    expect(offersTranslation('en', 'en')).toBe(false)
  })

  it('is on for another language or an unrecorded one', () => {
    expect(offersTranslation('de', 'en')).toBe(true)
    expect(offersTranslation('', 'en')).toBe(true)
    expect(offersTranslation(undefined, 'en')).toBe(true)
  })
})
