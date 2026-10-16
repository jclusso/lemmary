import { useMemo } from 'react'

import { lang } from '../i18n'
import { languageOptions } from '../lib/languages'
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
