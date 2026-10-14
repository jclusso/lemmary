import { describe, expect, it } from 'vitest'
import { canRemove, suggestedRemovals, type DocumentPage } from './documentPages'

const page = (number: number, blank: boolean): DocumentPage => ({ page: number, blank, thumbnail: '' })

describe('suggestedRemovals', () => {
  it('selects only the pages that look blank', () => {
    expect([...suggestedRemovals([page(1, false), page(2, true), page(3, false), page(4, true)])]).toEqual([2, 4])
  })

  it('selects nothing in a document without blank pages', () => {
    expect(suggestedRemovals([page(1, false), page(2, false)]).size).toBe(0)
  })
})

describe('canRemove', () => {
  it('needs a page selected', () => {
    expect(canRemove(new Set(), 3)).toBe(false)
  })

  it('refuses to remove every page', () => {
    expect(canRemove(new Set([1, 2, 3]), 3)).toBe(false)
  })

  it('allows a removal that leaves a page', () => {
    expect(canRemove(new Set([2]), 3)).toBe(true)
  })

  // Before the pages load there is nothing to remove from.
  it('refuses while no pages are known', () => {
    expect(canRemove(new Set([1]), 0)).toBe(false)
  })
})
