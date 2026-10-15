import { describe, expect, test } from 'vitest'
import {
  adminNavGroups,
  NAV_BADGE_DESCRIPTIONS,
  primaryNavItems,
  secondaryNavGroups,
  type NavGroup,
} from './nav'

function outline(groups: NavGroup[]) {
  return groups.map((group) => [group.heading ?? null, group.items.map((item) => item.label)])
}

describe('nav items', () => {
  test('leaves the admin entry out for a regular user', () => {
    expect(outline(secondaryNavGroups(false))).toEqual([
      // Tags is not admin-only: the vocabulary is per-user, like the account itself.
      [null, ['Account', 'Tags', 'Bulk Actions']],
      ['Help', ['Documentation', 'GitHub']],
    ])
  })

  test('gives an admin one entry into the admin area', () => {
    expect(outline(secondaryNavGroups(true))).toEqual([
      [null, ['Account', 'Tags', 'Bulk Actions']],
      [null, ['Admin']],
      ['Help', ['Documentation', 'GitHub']],
    ])
  })

  test('lists every settings section and admin page in the admin sidebar', () => {
    expect(outline(adminNavGroups('http://pb.test/_/', { hostEditable: true, ingest: true }))).toEqual([
      ['Settings', ['Appearance', 'AI', 'Processing', 'Fields', 'Worker', 'Duplicates', 'Ingest', 'MCP']],
      ['Instance', ['Users', 'Maintenance', 'OCR test', 'PocketBase']],
    ])
  })

  test('offers only what a managed instance can edit, and Ingest only once configured', () => {
    expect(outline(adminNavGroups('http://pb.test/_/', { hostEditable: false, ingest: false }))).toEqual([
      ['Settings', ['Appearance', 'Processing', 'Fields', 'MCP']],
      ['Instance', ['Users', 'Maintenance']],
    ])
  })

  test('gives every admin page an intro for its header', () => {
    const items = adminNavGroups('http://pb.test/_/', { hostEditable: true, ingest: true }).flatMap(
      (group) => group.items,
    )

    for (const item of items) {
      if (item.kind === 'route') expect(item.intro, item.label).toBeTruthy()
    }
  })

  test('points the PocketBase entry at the dashboard', () => {
    const groups = adminNavGroups('http://pb.test/_/', { hostEditable: true, ingest: false })
    expect(groups.at(-1)?.items.at(-1)).toEqual({
      kind: 'external',
      label: 'PocketBase',
      href: 'http://pb.test/_/',
    })
  })

  test('links the docs and the source repository', () => {
    const help = secondaryNavGroups(false).at(-1)

    expect(help?.items).toEqual([
      { kind: 'external', label: 'Documentation', href: '/docs/' },
      { kind: 'external', label: 'GitHub', href: 'https://github.com/buldezir/lemmary' },
    ])
  })

  test('lists the primary links in order', () => {
    const items = primaryNavItems(true)
    expect(items.map((item) => item.label)).toEqual([
      'Documents',
      'Inbox',
      'Upload',
      'Activity',
      'Deep Research',
    ])
  })

  // Without the setting, needs_review is only ever reached by a doubtful
  // extraction, so a permanent Inbox link would point at an empty list.
  test('offers the Inbox only where review is required', () => {
    expect(primaryNavItems(false).map((item) => item.label)).toEqual([
      'Documents',
      'Upload',
      'Activity',
      'Deep Research',
    ])
  })

  test('gives the Inbox and Activity paths of their own, and the count badges', () => {
    const badged = primaryNavItems(true).filter((item) => item.kind === 'route' && item.badgeKey)

    expect(badged).toEqual([
      { kind: 'route', label: 'Inbox', to: '/inbox', badgeKey: 'inbox' },
      { kind: 'route', label: 'Activity', to: '/activity', badgeKey: 'activity' },
    ])
  })

  // The header reads the digits out as "3 waiting for review", so a badge key
  // added without its sentence would render an undefined one.
  test('describes every badge key for assistive tech', () => {
    for (const item of primaryNavItems(true)) {
      if (item.kind === 'route' && item.badgeKey) {
        expect(NAV_BADGE_DESCRIPTIONS[item.badgeKey]).toBeTruthy()
      }
    }
  })

  // Only the two links that sit on a parent's own path are exact; every other
  // link has children it should stay lit for (/upload/split, /rag/research/<chat>).
  test('marks only the document list and Appearance as exact matches', () => {
    const secondary = [...secondaryNavGroups(true).flatMap((group) => group.items), ...adminNavGroups('/_/', { hostEditable: true, ingest: true }).flatMap((group) => group.items)]
    const exact = [...primaryNavItems(true), ...secondary].filter(
      (item) => item.kind === 'route' && item.exact,
    )

    expect(exact.map((item) => item.label)).toEqual(['Documents', 'Appearance'])
  })
})
