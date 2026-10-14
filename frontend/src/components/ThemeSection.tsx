import { useState } from 'react'
import { t } from '../i18n'
import { saveAccountTheme } from '../lib/auth'
import {
  families,
  modes,
  resolvePalette,
  setChoice,
  storedChoice,
  type Choice,
  type Palette,
} from '../lib/theme'
import { sectionClassName, sectionTitleClassName } from './ui'

const familyLabels = {
  paper: 'account.themePaper',
  neutral: 'account.themeNeutral',
  nord: 'account.themeNord',
  catppuccin: 'account.themeCatppuccin',
  rosepine: 'account.themeRosePine',
  monokai: 'account.themeMonokai',
} as const

const modeLabels = {
  system: 'account.modeSystem',
  light: 'account.modeLight',
  dark: 'account.modeDark',
} as const

function ThemeSwatch({ palette }: { palette: Palette }) {
  return (
    <div data-theme={palette} className="flex h-full gap-2 bg-paper p-2.5">
      <div className="flex w-1/4 flex-col gap-1.5 pt-1">
        <span className="h-1.5 w-3/4 rounded-full bg-ink" />
        <span className="h-1 w-full rounded-full bg-line-strong" />
        <span className="h-1 w-2/3 rounded-full bg-oxblood" />
        <span className="h-1 w-5/6 rounded-full bg-line-strong" />
      </div>
      <div className="flex flex-1 flex-col gap-1.5 border border-line bg-surface p-2">
        <span className="h-1.5 w-2/3 rounded-full bg-ink" />
        <span className="h-1 w-full rounded-full bg-ink-faint" />
        <span className="h-1 w-5/6 rounded-full bg-ink-faint" />
        <div className="mt-auto flex items-center gap-1">
          <span className="h-3 w-9 bg-oxblood" />
          <span className="h-3 w-6 border border-line-strong bg-bright" />
          <span className="ml-auto h-2 w-2 rounded-full bg-forest" />
          <span className="h-2 w-2 rounded-full bg-ochre" />
          <span className="h-2 w-2 rounded-full bg-madder" />
        </div>
      </div>
    </div>
  )
}

function FamilyPreview({ family, mode }: Choice) {
  if (mode !== 'system') return <ThemeSwatch palette={resolvePalette(family, mode, false)} />
  return (
    <div className="relative h-full">
      <ThemeSwatch palette={resolvePalette(family, 'light', false)} />
      <div className="absolute inset-0 [clip-path:polygon(100%_0,100%_100%,0_100%)]">
        <ThemeSwatch palette={resolvePalette(family, 'dark', true)} />
      </div>
    </div>
  )
}

export function ThemeSection() {
  const [current, setCurrent] = useState(storedChoice)
  const [error, setError] = useState('')

  function choose(next: Choice) {
    setChoice(next)
    setCurrent(next)
    setError('')
    saveAccountTheme(next).catch((err: unknown) => {
      setError(err instanceof Error ? err.message : t('account.themeSaveFailed'))
    })
  }

  return (
    <section className={sectionClassName}>
      <h2 className={sectionTitleClassName}>{t('account.theme')}</h2>
      <p className="text-xs text-ink-soft">{t('account.themeHint')}</p>
      <fieldset className="mt-4 inline-flex border border-line-strong">
        <legend className="sr-only">{t('account.mode')}</legend>
        {modes.map((mode) => (
          <label
            key={mode}
            className="cursor-pointer border-r border-line-strong px-4 py-1.5 text-sm text-ink-soft transition-colors last:border-r-0 hover:text-ink has-checked:bg-ink has-checked:text-paper has-focus-visible:outline-2 has-focus-visible:outline-offset-2 has-focus-visible:outline-focus"
          >
            <input
              type="radio"
              name="theme-mode"
              value={mode}
              checked={current.mode === mode}
              onChange={() => choose({ ...current, mode })}
              className="sr-only"
            />
            {t(modeLabels[mode])}
          </label>
        ))}
      </fieldset>
      <fieldset className="mt-4 grid grid-cols-2 gap-3 sm:grid-cols-3">
        <legend className="sr-only">{t('account.theme')}</legend>
        {families.map((family) => (
          <label
            key={family}
            className="cursor-pointer overflow-hidden border border-line bg-bright transition-colors hover:border-line-strong has-checked:border-oxblood has-checked:ring-1 has-checked:ring-oxblood has-focus-visible:outline-2 has-focus-visible:outline-offset-2 has-focus-visible:outline-focus"
          >
            <input
              type="radio"
              name="theme-family"
              value={family}
              checked={current.family === family}
              onChange={() => choose({ ...current, family })}
              className="sr-only"
            />
            <div className="h-24 border-b border-line">
              <FamilyPreview family={family} mode={current.mode} />
            </div>
            <span className="block px-3 py-2 text-sm font-medium text-ink">{t(familyLabels[family])}</span>
          </label>
        ))}
      </fieldset>
      {error && <p className="mt-3 text-sm text-madder">{error}</p>}
    </section>
  )
}
