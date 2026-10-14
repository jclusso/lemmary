import { describe, expect, it } from 'vitest'
import { parseFamily, parseMode, resolvePalette } from './theme'

describe('parseFamily', () => {
  it('accepts every known family', () => {
    expect(parseFamily('neutral')).toBe('neutral')
    expect(parseFamily('rosepine')).toBe('rosepine')
  })

  it('falls back to paper for a missing or unknown value', () => {
    expect(parseFamily(null)).toBe('paper')
    expect(parseFamily('mocha')).toBe('paper')
  })
})

describe('parseMode', () => {
  it('accepts every known mode', () => {
    expect(parseMode('system')).toBe('system')
    expect(parseMode('dark')).toBe('dark')
  })

  it('falls back to light, the look before modes existed', () => {
    expect(parseMode(null)).toBe('light')
    expect(parseMode('dim')).toBe('light')
  })
})

describe('resolvePalette', () => {
  it('keeps an explicit mode regardless of the system setting', () => {
    expect(resolvePalette('nord', 'light', true)).toBe('snowstorm')
    expect(resolvePalette('catppuccin', 'dark', false)).toBe('mocha')
  })

  it('follows the system setting within the family', () => {
    expect(resolvePalette('paper', 'system', true)).toBe('ink')
    expect(resolvePalette('paper', 'system', false)).toBe('paper')
    expect(resolvePalette('rosepine', 'system', true)).toBe('rosepine')
    expect(resolvePalette('rosepine', 'system', false)).toBe('dawn')
    expect(resolvePalette('monokai', 'system', false)).toBe('monokai-light')
  })
})
