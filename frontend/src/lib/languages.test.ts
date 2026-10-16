import { describe, expect, it } from 'vitest'
import { languageOptions, offersTranslation } from './languages'

describe('languageOptions', () => {
  it('names each code in the given locale', () => {
    const english = languageOptions('en')
    expect(english.find((option) => option.code === 'es')?.name).toBe('Spanish')
    expect(languageOptions('de').find((option) => option.code === 'es')?.name).toBe('Spanisch')
  })

  it('sorts by name rather than by code', () => {
    const names = languageOptions('en').map((option) => option.name)
    expect(names).toEqual([...names].sort((a, b) => a.localeCompare(b, 'en')))
    expect(names.length).toBe(183)
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
