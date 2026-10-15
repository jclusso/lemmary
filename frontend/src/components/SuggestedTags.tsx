import { t } from '../i18n'

/** The look every suggestion on a document shares: a label, then dashed chips. */
export const suggestionLabelClassName = 'text-[11px] uppercase tracking-[0.08em] text-amber-800'
export const suggestionChipClassName =
  'border border-dashed border-amber-800 px-1.5 py-0.5 text-[11px] text-amber-800 transition-colors hover:bg-amber-800 hover:text-paper disabled:opacity-50'

/**
 * The AI's proposed new tags on a document awaiting review, each a button that
 * accepts it. Dashed, unlike a real tag chip: nothing here exists yet.
 */
export function SuggestedTags({
  names,
  disabled,
  onAccept,
  className = '',
}: {
  names: string[]
  disabled?: boolean
  onAccept: (name: string) => void
  className?: string
}) {
  if (names.length === 0) return null
  return (
    <div className={`flex flex-wrap items-center gap-1.5 ${className}`}>
      <span className={suggestionLabelClassName}>{t('suggestedTags.suggested')}</span>
      {names.map((name) => (
        <button
          key={name}
          type="button"
          disabled={disabled}
          aria-label={t('suggestedTags.add', { name })}
          title={t('suggestedTags.addTitle')}
          className={suggestionChipClassName}
          onClick={() => onAccept(name)}
        >
          + {name}
        </button>
      ))}
    </div>
  )
}
