import { describe, expect, it } from 'vitest'
import {
  canSave,
  needsReread,
  suggestedRemovals,
  unreadKept,
  unreadPages,
  type DocumentPage,
} from './documentPages'

const page = (number: number, blank: boolean, unread = false): DocumentPage => ({
  page: number,
  blank,
  unread,
  thumbnail: '',
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

  it('has nothing to save without a removal or an unread page', () => {
    expect(canSave([page(1, false), page(2, true)], new Set())).toBe(false)
  })

  it('refuses while no pages are known', () => {
    expect(canSave([], new Set([1]))).toBe(false)
  })
})

describe('unreadPages', () => {
  it('numbers an unedited file as uploaded', () => {
    expect(unreadPages({ page_map: null, ocr_skipped_pages: [4, 2] })).toEqual([2, 4])
  })

  // The file holds uploaded pages 1, 3 and 4; OCR left out 2 and 4.
  it('renumbers through the page map', () => {
    expect(unreadPages({ page_map: [1, 3, 4], ocr_skipped_pages: [2, 4] })).toEqual([3])
  })

  it('is empty when nothing was left out', () => {
    expect(unreadPages({})).toEqual([])
  })
})
