import { useCallback, useEffect, useRef, useState } from 'react'
import { t } from '../i18n'
import {
  canSave,
  fetchPageThumbnail,
  getDocumentPages,
  needsReread,
  removeDocumentPages,
  restoreDocumentOriginal,
  suggestedRemovals,
  suggestedKept,
  unreadKept,
  type DocumentPages,
  type PageEditOutcome,
} from '../lib/api/documentPages'
import { Button } from './ui'

export type PageChange = 'removed' | 'kept' | 'restored'

/** Thumbnails fetched at once: the server renders each, and may have few CPUs. */
const THUMB_CONCURRENCY = 3

/**
 * Fetches each page's thumbnail once the page list is in, so the dialog opens
 * before the slowest page is rendered. Keyed by the list it was loaded for,
 * because a reopened dialog may describe a file with pages removed.
 */
function usePageThumbnails(documentId: string, list: DocumentPages | null) {
  const [loaded, setLoaded] = useState<{ list: DocumentPages | null; urls: Record<number, string> }>({
    list: null,
    urls: {},
  })

  useEffect(() => {
    if (!list) return
    let cancelled = false
    const urls: string[] = []
    const queue = list.pages.map((page) => page.page)
    const worker = async () => {
      for (;;) {
        const page = queue.shift()
        if (page === undefined || cancelled) return
        try {
          const url = await fetchPageThumbnail(documentId, page)
          if (cancelled) {
            URL.revokeObjectURL(url)
            return
          }
          urls.push(url)
          setLoaded((current) =>
            current.list === list
              ? { list, urls: { ...current.urls, [page]: url } }
              : { list, urls: { [page]: url } },
          )
        } catch {
          // The page can still be chosen by its number.
        }
      }
    }
    void Promise.all(Array.from({ length: THUMB_CONCURRENCY }, worker))
    return () => {
      cancelled = true
      for (const url of urls) URL.revokeObjectURL(url)
    }
  }, [documentId, list])

  return loaded.list === list ? loaded.urls : {}
}

/**
 * Removes pages from the owner's PDF. Pages that look blank arrive selected;
 * the owner confirms or changes the choice, and a page OCR left out that they
 * keep is read. The file as uploaded is kept by the backend, so the dialog
 * also offers to put it back. Whether OCR runs again is the backend's call.
 */
export function PageEditorDialog({
  documentId,
  open,
  onClose,
  onChanged,
}: {
  documentId: string
  open: boolean
  onClose: () => void
  onChanged: (change: PageChange, outcome: PageEditOutcome) => void
}) {
  const ref = useRef<HTMLDialogElement>(null)
  const [data, setData] = useState<DocumentPages | null>(null)
  const [selected, setSelected] = useState<Set<number>>(new Set())
  const [loading, setLoading] = useState(false)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const thumbnails = usePageThumbnails(documentId, open ? data : null)

  const load = useCallback(
    async (signal: AbortSignal) => {
      if (signal.aborted) return
      setData(null)
      setError('')
      setLoading(true)
      try {
        const result = await getDocumentPages(documentId, signal)
        setData(result)
        setSelected(suggestedRemovals(result.pages))
      } catch (err) {
        if (signal.aborted) return
        setError(err instanceof Error ? err.message : t('pageEditor.loadFailed'))
      } finally {
        if (!signal.aborted) setLoading(false)
      }
    },
    [documentId],
  )

  useEffect(() => {
    const dialog = ref.current
    if (!dialog) return
    if (!open) {
      dialog.close()
      return
    }
    dialog.showModal()
    const controller = new AbortController()
    void Promise.resolve().then(() => load(controller.signal))
    return () => controller.abort()
  }, [open, load])

  function toggle(page: number) {
    setSelected((current) => {
      const next = new Set(current)
      if (next.has(page)) next.delete(page)
      else next.add(page)
      return next
    })
  }

  async function run(action: () => Promise<PageEditOutcome>, change: PageChange) {
    setBusy(true)
    setError('')
    try {
      onChanged(change, await action())
      onClose()
    } catch (err) {
      setError(err instanceof Error ? err.message : t('pageEditor.changeFailed'))
    } finally {
      setBusy(false)
    }
  }

  const pages = data?.pages ?? []
  const blankCount = pages.filter((page) => page.blank).length
  const keepsNone = pages.length > 0 && selected.size === pages.length
  const toRead = unreadKept(pages, selected).length
  const toKeep = suggestedKept(pages, selected).length
  const savable = canSave(pages, selected)

  return (
    <dialog
      ref={ref}
      onClose={onClose}
      className="m-auto w-full max-w-4xl border border-line bg-surface p-5 text-ink backdrop:bg-shade/40"
    >
      <h2 className="font-display text-lg font-semibold text-ink">{t('pageEditor.title')}</h2>
      <p className="mt-1 text-sm text-ink-soft">{t('pageEditor.description')}</p>

      {loading && <p className="mt-4 text-sm text-ink-soft">{t('pageEditor.loading')}</p>}
      {error && <p className="mt-4 text-sm text-madder">{error}</p>}

      {data && (
        <>
          <p className="mt-4 text-sm text-ink-muted">
            {blankCount > 0 ? t('pageEditor.blankFound', { count: blankCount }) : t('pageEditor.noBlank')}
          </p>
          <ul className="mt-3 grid max-h-[60vh] grid-cols-2 gap-3 overflow-y-auto sm:grid-cols-3 md:grid-cols-4">
            {pages.map((page) => {
              const isSelected = selected.has(page.page)
              return (
                <li key={page.page}>
                  <button
                    type="button"
                    aria-pressed={isSelected}
                    disabled={busy}
                    onClick={() => toggle(page.page)}
                    className={`flex w-full flex-col gap-1.5 border p-2 text-left transition-colors focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-oxblood ${
                      isSelected
                        ? 'border-madder bg-madder/5'
                        : 'border-line hover:border-ink'
                    }`}
                  >
                    {thumbnails[page.page] ? (
                      <img
                        src={thumbnails[page.page]}
                        alt={t('pageEditor.page', { page: page.page })}
                        className={`w-full border border-line bg-paper ${isSelected ? 'opacity-40' : ''}`}
                      />
                    ) : (
                      <span className="block aspect-[17/22] w-full animate-pulse border border-line bg-paper" />
                    )}
                    <span className="flex items-center justify-between gap-2 text-xs">
                      <span className="font-medium text-ink">{t('pageEditor.page', { page: page.page })}</span>
                      {isSelected ? (
                        <span className="font-semibold text-madder">{t('pageEditor.willRemove')}</span>
                      ) : page.unread ? (
                        <span className="font-semibold text-ink">{t('pageEditor.willRead')}</span>
                      ) : (
                        page.blank && <span className="text-ink-soft">{t('pageEditor.looksBlank')}</span>
                      )}
                    </span>
                  </button>
                </li>
              )
            })}
          </ul>
          {keepsNone && <p className="mt-3 text-sm text-madder">{t('pageEditor.keepOne')}</p>}
          {savable && (
            <p className="mt-3 text-sm text-ink-muted">
              {needsReread(pages, selected) ? t('pageEditor.willReread') : t('pageEditor.textKept')}
            </p>
          )}
        </>
      )}

      <div className="mt-5 flex flex-wrap items-center justify-between gap-3">
        <div>
          {data?.has_original && (
            <Button
              variant="secondary"
              disabled={busy}
              title={t('pageEditor.restoreHint')}
              onClick={() => void run(() => restoreDocumentOriginal(documentId), 'restored')}
            >
              {t('pageEditor.restore')}
            </Button>
          )}
        </div>
        <div className="flex gap-2">
          <Button variant="secondary" onClick={onClose} disabled={busy}>
            {t('common.cancel')}
          </Button>
          <Button
            variant={selected.size > 0 ? 'danger' : 'primary'}
            disabled={busy || !savable}
            onClick={() =>
              void run(
                () => removeDocumentPages(documentId, [...selected].sort((a, b) => a - b)),
                selected.size > 0 ? 'removed' : 'kept',
              )
            }
          >
            {busy
              ? t('pageEditor.working')
              : selected.size > 0 || toKeep === 0
                ? t('pageEditor.remove', { count: selected.size })
                : toRead > 0
                  ? t('pageEditor.read', { count: toRead })
                  : t('pageEditor.keep', { count: toKeep })}
          </Button>
        </div>
      </div>
    </dialog>
  )
}
