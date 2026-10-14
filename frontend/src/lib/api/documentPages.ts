import { t } from '../../i18n'
import { apiFetch } from '../apiClient'

export type DocumentPage = {
  page: number
  blank: boolean
  /** A JPEG data URL. */
  thumbnail: string
}

export type DocumentPages = {
  pages: DocumentPage[]
  /** True once pages were removed: the file as uploaded can be put back. */
  has_original: boolean
}

/** The pages selected when the editor opens: every one that looks blank. */
export function suggestedRemovals(pages: DocumentPage[]): Set<number> {
  return new Set(pages.filter((page) => page.blank).map((page) => page.page))
}

/** A removal needs at least one page selected and one left over. */
export function canRemove(selected: ReadonlySet<number>, pageCount: number): boolean {
  return selected.size > 0 && selected.size < pageCount
}

/** Renders every page of the caller's own PDF, flagging the ones that look blank. */
export function getDocumentPages(documentId: string, signal?: AbortSignal) {
  return apiFetch<DocumentPages>(`/api/app/documents/${encodeURIComponent(documentId)}/pages`, {
    signal,
    fallbackError: t('pageEditor.loadFailed'),
  })
}

/** Replaces the PDF with one lacking these pages and queues preview and OCR again. */
export function removeDocumentPages(documentId: string, pages: number[]) {
  return apiFetch<{ status: string }>(
    `/api/app/documents/${encodeURIComponent(documentId)}/pages/removal`,
    { method: 'POST', body: { pages }, fallbackError: t('pageEditor.changeFailed') },
  )
}

/** Puts back the file as it was before any page was removed. */
export function restoreDocumentOriginal(documentId: string) {
  return apiFetch<{ status: string }>(
    `/api/app/documents/${encodeURIComponent(documentId)}/original/restore`,
    { method: 'POST', fallbackError: t('pageEditor.changeFailed') },
  )
}
