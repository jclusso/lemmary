import type { LinkProps } from '@tanstack/react-router'
import { docsUrl, t } from '../i18n'

export const githubRepoUrl = 'https://github.com/buldezir/lemmary'

/**
 * The header's link set as data: the bar and the mobile panel render it in two
 * shapes, and a link added to only one of them was the recurring failure.
 */

/** Names a count the header hangs off a link -- the key, never the number. */
export type NavBadgeKey = 'inbox' | 'activity'

/**
 * What a badge's digits mean, spelled out for assistive tech -- "Inbox 3" is
 * not a useful accessible name. Rendered after the count.
 */
export const NAV_BADGE_DESCRIPTIONS: Record<NavBadgeKey, string> = {
  inbox: t('nav.badgeInbox'),
  activity: t('nav.badgeActivity'),
}

export type RouteNavItem = {
  kind: 'route'
  label: string
  to: LinkProps['to']
  /** Marks the link active only on an exact path match. */
  exact?: boolean
  /** Which count to show beside the label, if any. */
  badgeKey?: NavBadgeKey
  /** One line under the page title saying what the page is for. */
  intro?: string
}

export type ExternalNavItem = {
  kind: 'external'
  label: string
  href: string
}

export type NavItem = RouteNavItem | ExternalNavItem

/** A run of secondary links, set off by a rule and, after the first, a heading. */
export type NavGroup = {
  key: 'personal' | 'admin' | 'help' | 'settings' | 'instance'
  heading?: string
  items: NavItem[]
}

/**
 * A function because the Inbox is conditional: without always_require_review a
 * document reaches needs_review only by extracting doubtfully, and a permanent
 * link to an almost always empty list is worse than no link.
 */
export function primaryNavItems(reviewRequired: boolean): readonly NavItem[] {
  return [
    { kind: 'route', label: t('nav.documents'), to: '/', exact: true },
    // A path rather than /?status=needs_review: a search-param link would be
    // active whenever Documents was, / being a subset of every search.
    ...(reviewRequired
      ? [{ kind: 'route', label: t('nav.inbox'), to: '/inbox', badgeKey: 'inbox' } as const]
      : []),
    { kind: 'route', label: t('nav.upload'), to: '/upload' },
    { kind: 'route', label: t('nav.activity'), to: '/activity', badgeKey: 'activity' },
    { kind: 'route', label: t('nav.deepResearch'), to: '/rag/research' },
  ]
}

/**
 * The links behind the bar's "More" menu, listed inline on a narrow one. The
 * admin area is one entry here and lists its own pages in a sidebar; the admin
 * routes bounce non-admins anyway, so leaving it out avoids offering a dead end.
 */
export function secondaryNavGroups(admin: boolean): NavGroup[] {
  const groups: NavGroup[] = [
    {
      key: 'personal',
      items: [
        { kind: 'route', label: t('nav.account'), to: '/account' },
        { kind: 'route', label: t('nav.tags'), to: '/tags' },
        { kind: 'route', label: t('nav.bulkActions'), to: '/bulk' },
      ],
    },
  ]
  if (admin) {
    groups.push({ key: 'admin', items: [{ kind: 'route', label: t('nav.adminArea'), to: '/settings' }] })
  }
  groups.push({
    key: 'help',
    heading: t('nav.groupHelp'),
    items: [
      { kind: 'external', label: t('footer.documentation'), href: docsUrl() },
      { kind: 'external', label: 'GitHub', href: githubRepoUrl },
    ],
  })
  return groups
}

/**
 * The admin area's sidebar: every settings section and instance page in one
 * list, so no admin page needs a tab bar of its own. A managed instance
 * (`hostEditable` false) has nothing editable behind AI, Worker, Duplicates,
 * OCR test or the PocketBase dashboard; Ingest exists only once a source is
 * configured.
 */
export function adminNavGroups(
  pbAdminUrl: string,
  { hostEditable, ingest }: { hostEditable: boolean; ingest: boolean },
): NavGroup[] {
  const settings: (NavItem | false)[] = [
    { kind: 'route', label: t('settings.tabAppearance'), to: '/settings', exact: true, intro: t('admin.introAppearance') },
    hostEditable && { kind: 'route', label: t('settings.tabAi'), to: '/settings/ai', intro: t('admin.introAi') },
    { kind: 'route', label: t('settings.tabProcessing'), to: '/settings/processing', intro: t('admin.introProcessing') },
    { kind: 'route', label: t('settings.tabFields'), to: '/settings/fields', intro: t('admin.introFields') },
    hostEditable && { kind: 'route', label: t('settings.tabWorker'), to: '/settings/worker', intro: t('admin.introWorker') },
    hostEditable && { kind: 'route', label: t('settings.tabDuplicates'), to: '/settings/duplicates', intro: t('admin.introDuplicates') },
    ingest && { kind: 'route', label: t('settings.tabIngest'), to: '/settings/ingest', intro: t('admin.introIngest') },
    { kind: 'route', label: t('settings.tabMcp'), to: '/settings/mcp', intro: t('admin.introMcp') },
  ]
  const instance: (NavItem | false)[] = [
    { kind: 'route', label: t('nav.users'), to: '/management', intro: t('admin.introUsers') },
    { kind: 'route', label: t('nav.maintenance'), to: '/maintenance', intro: t('maintenance.intro') },
    hostEditable && { kind: 'route', label: t('nav.ocrTest'), to: '/ocr-test', intro: t('ocrTest.intro') },
    hostEditable && { kind: 'external', label: t('nav.admin'), href: pbAdminUrl },
  ]
  return [
    { key: 'settings', heading: t('nav.settings'), items: settings.filter((item) => item !== false) },
    { key: 'instance', heading: t('nav.groupInstance'), items: instance.filter((item) => item !== false) },
  ]
}
