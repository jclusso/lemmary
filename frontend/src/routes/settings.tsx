import { Outlet } from '@tanstack/react-router'

import { useAppMeta } from '../hooks/useAppMeta'
import { t } from '../i18n'

/**
 * The Settings shell: whichever section is on the URL, picked from the admin
 * sidebar. Each section saves only its own fields -- the settings PATCH treats
 * every field as optional, so a section naming its own leaves the rest alone.
 */
export function SettingsPage() {
  // unknown/failed meta counts as managed; see AppMeta.managed
  const { managed } = useAppMeta()

  return (
    <>
      {managed !== false && <p className="mb-5 text-sm text-ink-soft">{t('settings.aiManaged')}</p>}
      <Outlet />
    </>
  )
}
