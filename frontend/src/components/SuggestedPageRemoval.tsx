import { lang, t } from '../i18n'
import { suggestionChipClassName, suggestionLabelClassName } from './SuggestedTags'

const listFormat = new Intl.ListFormat(lang, { style: 'long', type: 'conjunction' })

/**
 * Pages OCR left out as blank, offered for removal in the style of the other
 * suggestions. The chip opens the page editor rather than removing anything,
 * because the owner should see the pages first.
 */
export function SuggestedPageRemoval({ pages, onReview }: { pages: number[]; onReview: () => void }) {
  if (pages.length === 0) return null
  const list = listFormat.format(pages.map(String))
  return (
    <div className="flex flex-wrap items-center gap-1.5">
      <span className={suggestionLabelClassName}>{t('suggestedTags.suggested')}</span>
      <button
        type="button"
        title={t('documentPage.unreadPages', { count: pages.length, pages: list })}
        className={suggestionChipClassName}
        onClick={onReview}
      >
        − {t('documentPage.removeBlankPages', { count: pages.length, pages: list })}
      </button>
    </div>
  )
}
