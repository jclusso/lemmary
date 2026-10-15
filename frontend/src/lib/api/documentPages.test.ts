import { describe, expect, it } from 'vitest'
import {
  blankSuggestion,
  canSave,
  needsReread,
  suggestedKept,
  suggestedRemovals,
  unreadKept,
  type DocumentPage,
} from './documentPages'

const page = (number: number, blank: boolean, unread = false): DocumentPage => ({
  page: number,
  blank,
  unread,
})

// Page 2 and 4 look blank and OCR left them out; page 3 is blank but was read.
const pages = [page(1, false), page(2, true, true), page(3, true), page(4, true, true)]

describe('suggestedRemovals', () => {
  it('selects only the pages that look blank', () => {
    expect([...suggestedRemovals(pages)]).toEqual([2, 3, 4])
  })

  it('selects nothing in a document without blank pages', () => {
    expect(suggestedRemovals([page(1, false), page(2, false)]).size).toBe(0)
  })
})

describe('needsReread', () => {
  it('keeps the text when only unread pages go', () => {
    expect(needsReread(pages, new Set([2, 4]))).toBe(false)
  })

  it('rereads when a page that was read goes', () => {
    expect(needsReread(pages, new Set([2, 3, 4]))).toBe(true)
  })

  it('rereads when an unread page stays', () => {
    expect(needsReread(pages, new Set([2]))).toBe(true)
  })
})

describe('canSave', () => {
  it('saves a removal that leaves a page', () => {
    expect(canSave(pages, new Set([2]))).toBe(true)
  })

  it('refuses to remove every page', () => {
    expect(canSave(pages, new Set([1, 2, 3, 4]))).toBe(false)
  })

  it('saves keeping unread pages with nothing removed', () => {
    expect(unreadKept(pages, new Set())).toEqual([2, 4])
    expect(canSave(pages, new Set())).toBe(true)
  })

  // Page 2 looks blank but was read: keeping it only ends the suggestion.
  it('saves keeping a suggested page that was read', () => {
    expect(suggestedKept([page(1, false), page(2, true)], new Set())).toEqual([2])
    expect(canSave([page(1, false), page(2, true)], new Set())).toBe(true)
  })

  it('has nothing to save without a removal or a suggested page', () => {
    expect(canSave([page(1, false), page(2, false)], new Set())).toBe(false)
  })

  it('refuses while no pages are known', () => {
    expect(canSave([], new Set([1]))).toBe(false)
  })
})

describe('blankSuggestion', () => {
  it('numbers an unedited file as uploaded', () => {
    expect(blankSuggestion({ page_map: null, blank_pages: [4, 2], ocr_skipped_pages: [2, 4] })).toEqual({
      pages: [2, 4],
      unread: true,
    })
  })

  // The file holds uploaded pages 1, 3 and 4; 2 and 4 were found blank.
  it('renumbers through the page map', () => {
    expect(blankSuggestion({ page_map: [1, 3, 4], blank_pages: [2, 4] })).toEqual({ pages: [3], unread: false })
  })

  it('leaves out pages the owner kept', () => {
    expect(blankSuggestion({ blank_pages: [2, 4], confirmed_pages: [4] }).pages).toEqual([2])
  })

  it('says read when any suggested page is in the text', () => {
    expect(blankSuggestion({ blank_pages: [2, 4], ocr_skipped_pages: [2] }).unread).toBe(false)
  })

  it('is empty when nothing was found', () => {
    expect(blankSuggestion({})).toEqual({ pages: [], unread: false })
  })
})
