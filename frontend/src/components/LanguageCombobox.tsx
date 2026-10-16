import { useMemo } from 'react'

import { lang, t } from '../i18n'
import { languageName, languageOptions } from '../lib/languages'
import { Combobox } from './Combobox'

type Props = {
  value: string
  onChange: (code: string) => void
  /** The choice that stores no language. */
  emptyLabel: string
  id?: string
  ariaLabel?: string
  disabled?: boolean
}

/** A searchable pick of an ISO 639-1 language, by name or by code. */
export function LanguageCombobox({ value, onChange, emptyLabel, id, ariaLabel, disabled }: Props) {
  const options = useMemo(
    () => [
      { value: '', label: emptyLabel, pinned: true },
      ...languageOptions(lang).map((option) => ({ value: option.code, label: option.name })),
    ],
    [emptyLabel],
  )
  return (
    <Combobox
      value={value}
      options={options}
      onChange={onChange}
      placeholder={emptyLabel}
      id={id}
      ariaLabel={ariaLabel}
      disabled={disabled}
    />
  )
}

type MultiProps = {
  value: string[]
  onChange: (codes: string[]) => void
  placeholder: string
  id?: string
}

/** Several ISO 639-1 languages, chosen ones shown as removable pills in the box. */
export function LanguageMultiCombobox({ value, onChange, placeholder, id }: MultiProps) {
  const options = useMemo(
    () =>
      languageOptions(lang)
        .filter((option) => !value.includes(option.code))
        .map((option) => ({ value: option.code, label: option.name })),
    [value],
  )
  const pills = value.map((code) => {
    const name = languageName(code, lang)
    return (
      <span
        key={code}
        className="flex max-w-full items-center gap-1 rounded-xs border border-line-strong bg-wash px-2 py-0.5 text-xs text-ink"
      >
        <span className="truncate">{name}</span>
        <button
          type="button"
          aria-label={t('tagFilter.remove', { name })}
          className="shrink-0 text-ink-faint transition-colors hover:text-madder"
          onClick={() => onChange(value.filter((other) => other !== code))}
        >
          &times;
        </button>
      </span>
    )
  })
  return (
    <Combobox
      value=""
      options={options}
      onChange={(code) => onChange([...value, code])}
      placeholder={placeholder}
      id={id}
      leading={pills.length > 0 ? pills : undefined}
    />
  )
}
