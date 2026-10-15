import { t } from '../../i18n'
import { apiFetch } from '../apiClient'
import type { DocumentRecord } from './documents'

export type DocumentPage = {
  page: number
  /** Looks blank, and the owner has not said it has content. */
  blank: boolean
  /** OCR left the page out, so the document's text has nothing from it. */
  unread: boolean
  /** A JPEG data URL. */
  thumbnail: string
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

/** A change needs something to do and at least one page left over. */
export function canSave(pages: DocumentPage[], selected: ReadonlySet<number>): boolean {
  if (pages.length === 0 || selected.size >= pages.length) return false
  return selected.size > 0 || unreadKept(pages, selected).length > 0
}

/**
 * Mirrors the backend's rule: the text still holds when only unread pages go
 * and every unread page goes; otherwise OCR runs again.
 */
export function needsReread(pages: DocumentPage[], selected: ReadonlySet<number>): boolean {
  const removesRead = pages.some((page) => selected.has(page.page) && !page.unread)
  return removesRead || unreadKept(pages, selected).length > 0
}

/** The current file's pages that OCR left out, numbered as the file is now. */
export function unreadPages(document: Pick<DocumentRecord, 'page_map' | 'ocr_skipped_pages'>): number[] {
  const skipped = new Set(document.ocr_skipped_pages ?? [])
  if (skipped.size === 0) return []
  const map = document.page_map
  if (!map || map.length === 0) return [...skipped].sort((a, b) => a - b)
  return map.flatMap((uploaded, index) => (skipped.has(uploaded) ? [index + 1] : []))
}

/** Renders every page of the caller's own PDF, flagging the ones that look blank. */
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
