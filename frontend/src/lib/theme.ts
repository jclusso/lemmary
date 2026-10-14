export const families = ['paper', 'neutral', 'nord', 'catppuccin', 'rosepine', 'monokai'] as const
export type Family = (typeof families)[number]

export const modes = ['system', 'light', 'dark'] as const
export type Mode = (typeof modes)[number]

export type Choice = { family: Family; mode: Mode }

const palettes = {
  paper: { light: 'paper', dark: 'ink' },
  neutral: { light: 'daylight', dark: 'carbon' },
  nord: { light: 'snowstorm', dark: 'nord' },
  catppuccin: { light: 'latte', dark: 'mocha' },
  rosepine: { light: 'dawn', dark: 'rosepine' },
  monokai: { light: 'monokai-light', dark: 'monokai' },
} as const satisfies Record<Family, { light: string; dark: string }>

export type Palette = (typeof palettes)[Family][keyof (typeof palettes)[Family]]

const familyKey = 'lemmary_theme'
const modeKey = 'lemmary_mode'
const darkQuery = '(prefers-color-scheme: dark)'
const defaultChoice: Choice = { family: 'paper', mode: 'light' }

export function parseFamily(stored: string | null): Family {
  return families.find((family) => family === stored) ?? defaultChoice.family
}

export function parseMode(stored: string | null): Mode {
  return modes.find((mode) => mode === stored) ?? defaultChoice.mode
}

export function resolvePalette(family: Family, mode: Mode, prefersDark: boolean): Palette {
  const dark = mode === 'dark' || (mode === 'system' && prefersDark)
  return palettes[family][dark ? 'dark' : 'light']
}

export function storedChoice(): Choice {
  try {
    return {
      family: parseFamily(window.localStorage.getItem(familyKey)),
      mode: parseMode(window.localStorage.getItem(modeKey)),
    }
  } catch {
    return defaultChoice
  }
}

function applyChoice({ family, mode }: Choice) {
  const root = document.documentElement
  root.dataset.theme = resolvePalette(family, mode, window.matchMedia(darkQuery).matches)
  const ground = getComputedStyle(root).getPropertyValue('--color-paper').trim()
  if (ground) document.querySelector('meta[name="theme-color"]')?.setAttribute('content', ground)
}

export function setChoice(choice: Choice) {
  try {
    window.localStorage.setItem(familyKey, choice.family)
    window.localStorage.setItem(modeKey, choice.mode)
  } catch {
    // Applied for this session, just not remembered.
  }
  applyChoice(choice)
}

/** The account's saved choice wins over this browser's, once signed in. Empty: never chosen. */
export function adoptAccountChoice(family: string, mode: string) {
  if (!family && !mode) return
  const current = storedChoice()
  const next = {
    family: family ? parseFamily(family) : current.family,
    mode: mode ? parseMode(mode) : current.mode,
  }
  if (next.family !== current.family || next.mode !== current.mode) setChoice(next)
}

export function initTheme() {
  applyChoice(storedChoice())
  window.matchMedia(darkQuery).addEventListener('change', () => applyChoice(storedChoice()))
}
