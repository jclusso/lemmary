import { t } from '../../i18n'
import { apiFetch } from '../apiClient'
import { ensureAuth } from '../auth'
import { pb, pbUrl } from '../pb'
import type { DocumentRecord } from './documents'

export type DocumentPage = {
  page: number
  /** Looks blank, and the owner has not said it has content. */
  blank: boolean
  /** OCR left the page out, so the document's text has nothing from it. */
  unread: boolean
}

export type DocumentPages = {
  pages: DocumentPage[]
  /** True once pages were removed: the file as uploaded can be put back. */
  has_original: boolean
}

/** Whether the edit queued OCR again. */
export type PageEditOutcome = { reread: boolean }

/** The pages selected when the editor opens: every one that looks blank. */
export function suggestedRemovals(pages: DocumentPage[]): Set<number> {
  return new Set(pages.filter((page) => page.blank).map((page) => page.page))
}

/**
 * Unread pages that are not selected: keeping them says they have content, so
 * they are read. Saving needs a page removed or one of these.
 */
export function unreadKept(pages: DocumentPage[], selected: ReadonlySet<number>): number[] {
  return pages.filter((page) => page.unread && !selected.has(page.page)).map((page) => page.page)
}

/**
 * Pages suggested for removal (blank or unread) that are not selected: keeping
 * them confirms they stay, which ends the suggestion.
 */
export function suggestedKept(pages: DocumentPage[], selected: ReadonlySet<number>): number[] {
  return pages
    .filter((page) => (page.blank || page.unread) && !selected.has(page.page))
    .map((page) => page.page)
}

/** A change needs something to do and at least one page left over. */
export function canSave(pages: DocumentPage[], selected: ReadonlySet<number>): boolean {
  if (pages.length === 0 || selected.size >= pages.length) return false
  return selected.size > 0 || suggestedKept(pages, selected).length > 0
}

/**
 * Mirrors the backend's rule: the text still holds when only unread pages go
 * and every unread page goes; otherwise OCR runs again.
 */
export function needsReread(pages: DocumentPage[], selected: ReadonlySet<number>): boolean {
  const removesRead = pages.some((page) => selected.has(page.page) && !page.unread)
  return removesRead || unreadKept(pages, selected).length > 0
}

export type BlankSuggestion = {
  /** Pages of the current file, numbered as the file is now. */
  pages: number[]
  /** Every one of them was left out of the OCR text. */
  unread: boolean
}

/**
 * The pages the detect_blank_pages step found blank that are still in the file
 * and that the owner has not kept.
 */
export function blankSuggestion(
  document: Pick<DocumentRecord, 'page_map' | 'blank_pages' | 'ocr_skipped_pages' | 'confirmed_pages'>,
): BlankSuggestion {
  const blank = new Set(document.blank_pages ?? [])
  const confirmed = new Set(document.confirmed_pages ?? [])
  const skipped = new Set(document.ocr_skipped_pages ?? [])
  // Without a map the file is the upload, so a page's number is its own.
  const pages = document.page_map?.length
    ? document.page_map.map((uploaded, index) => ({ page: index + 1, uploaded }))
    : [...blank].sort((a, b) => a - b).map((uploaded) => ({ page: uploaded, uploaded }))
  const suggested = pages.filter((entry) => blank.has(entry.uploaded) && !confirmed.has(entry.uploaded))
  return {
    pages: suggested.map((entry) => entry.page),
    unread: suggested.length > 0 && suggested.every((entry) => skipped.has(entry.uploaded)),
  }
}

/** One page of the caller's own PDF as a JPEG, returned as an object URL to revoke. */
export async function fetchPageThumbnail(documentId: string, page: number): Promise<string> {
  await ensureAuth()
  const response = await fetch(
    `${pbUrl}/api/app/documents/${encodeURIComponent(documentId)}/pages/${page}`,
    { headers: { Authorization: pb.authStore.token } },
  )
  if (!response.ok) {
    throw new Error(t('pageEditor.loadFailed'))
  }
  return URL.createObjectURL(await response.blob())
}

/** Lists the pages of the caller's own PDF, flagging the ones that look blank. */
export function getDocumentPages(documentId: string, signal?: AbortSignal) {
  return apiFetch<DocumentPages>(`/api/app/documents/${encodeURIComponent(documentId)}/pages`, {
    signal,
    fallbackError: t('pageEditor.loadFailed'),
  })
}

/**
 * Removes these pages; unread pages left in are read from then on. The backend
 * decides whether that needs OCR again.
 */
export function removeDocumentPages(documentId: string, pages: number[]) {
  return apiFetch<PageEditOutcome>(
    `/api/app/documents/${encodeURIComponent(documentId)}/pages/removal`,
    { method: 'POST', body: { pages }, fallbackError: t('pageEditor.changeFailed') },
  )
}

/** Puts back the file as it was before any page was removed. */
export function restoreDocumentOriginal(documentId: string) {
  return apiFetch<PageEditOutcome>(
    `/api/app/documents/${encodeURIComponent(documentId)}/original/restore`,
    { method: 'POST', fallbackError: t('pageEditor.changeFailed') },
  )
}
