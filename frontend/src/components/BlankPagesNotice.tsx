import { lang, t } from '../i18n'
import type { BlankSuggestion } from '../lib/api/documentPages'
import { Button } from './ui'

const listFormat = new Intl.ListFormat(lang, { style: 'long', type: 'conjunction' })

/**
 * Pages the pipeline found blank, raised the way a likely duplicate is: a
 * notice at the top of the document. It opens the page editor rather than
 * removing anything, because the owner should see the pages first.
 */
export function BlankPagesNotice({ suggestion, onReview }: { suggestion: BlankSuggestion; onReview: () => void }) {
  const count = suggestion.pages.length
  if (count === 0) return null
  const pages = listFormat.format(suggestion.pages.map(String))
  return (
    <div className="flex flex-wrap items-center justify-between gap-3 rounded-xs border border-ochre-line bg-ochre-wash px-4 py-3 text-sm text-ochre">
      <span>
        {suggestion.unread
          ? t('documentPage.unreadPages', { count, pages })
          : t('documentPage.blankPages', { count, pages })}
      </span>
      <Button variant="secondary" size="sm" onClick={onReview}>
        {t('documentPage.reviewBlankPages')}
      </Button>
    </div>
  )
}
