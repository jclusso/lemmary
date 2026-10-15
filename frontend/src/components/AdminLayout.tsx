import { Outlet, useMatchRoute, useNavigate } from '@tanstack/react-router'
import { useAppMeta } from '../hooks/useAppMeta'
import { adminNavGroups, type NavGroup, type RouteNavItem } from '../lib/nav'
import { pbAdminUrl } from '../lib/pb'
import { t } from '../i18n'
import { selectClassName } from './ui'
import { NavItemLink } from './NavItemLink'

const sidebarItemClass =
  '-ml-px flex items-center gap-2 border-l-2 border-transparent py-1.5 pl-3 pr-2 text-sm font-medium text-ink-muted transition-colors hover:text-oxblood focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-focus data-[status=active]:border-oxblood data-[status=active]:text-oxblood'
const sidebarHeadingClass =
  'mb-1 pl-3 text-[10px] font-semibold uppercase tracking-[0.14em] text-ink-faint'

function ShieldIcon() {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.75"
      strokeLinecap="round"
      strokeLinejoin="round"
      className="h-4 w-4 shrink-0 text-ochre"
      aria-hidden="true"
    >
      <path d="M12 3 4.5 6v5.5c0 4.4 3.1 8.4 7.5 9.5 4.4-1.1 7.5-5.1 7.5-9.5V6L12 3Z" />
      <path d="M9.5 12.5 11.5 14.5 15 11" />
    </svg>
  )
}

function routeItems(groups: NavGroup[]): RouteNavItem[] {
  return groups.flatMap((group) => group.items).filter((item) => item.kind === 'route')
}

/** Below `md` the sidebar would push the page off screen, so it collapses to one select. */
function SectionSelect({ groups, current }: { groups: NavGroup[]; current?: RouteNavItem }) {
  const navigate = useNavigate()
  const items = routeItems(groups)

  return (
    <select
      aria-label={t('admin.sectionsLabel')}
      value={current ? String(current.to) : ''}
      onChange={(event) => {
        const item = items.find((candidate) => String(candidate.to) === event.target.value)
        if (item) void navigate({ to: item.to })
      }}
      className={`${selectClassName} w-full md:hidden`}
    >
      {groups.map((group) => (
        <optgroup key={group.key} label={group.heading}>
          {group.items
            .filter((item) => item.kind === 'route')
            .map((item) => (
              <option key={String(item.to)} value={String(item.to)}>
                {item.label}
              </option>
            ))}
        </optgroup>
      ))}
    </select>
  )
}

/**
 * The frame every admin page renders in: a sidebar holding every admin page,
 * settings sections included, beside the page's own title and intro, so no
 * page says "admin only" or carries a tab bar of its own.
 */
export function AdminLayout() {
  // unknown/failed meta counts as managed; see AppMeta.managed
  const { managed, ingestDir, ingestImap } = useAppMeta()
  const groups = adminNavGroups(pbAdminUrl, {
    hostEditable: managed === false,
    ingest: Boolean(ingestDir || ingestImap),
  })
  const matchRoute = useMatchRoute()
  const current = routeItems(groups).find((item) =>
    Boolean(matchRoute({ to: item.to, fuzzy: !item.exact })),
  )

  return (
    <div className="mx-auto flex max-w-6xl flex-col gap-6 md:flex-row md:gap-10">
      <aside className="flex flex-col gap-4 md:w-48 md:shrink-0 md:gap-6">
        <p className="flex items-center gap-1.5 pt-1 text-xs font-semibold uppercase tracking-[0.14em] text-ochre">
          <ShieldIcon />
          {t('admin.title')}
        </p>
        <SectionSelect groups={groups} current={current} />
        <nav aria-label={t('admin.sectionsLabel')} className="hidden flex-col gap-5 md:flex">
          {groups.map((group) => (
            <div key={group.key} role="group" aria-label={group.heading}>
              <p aria-hidden="true" className={sidebarHeadingClass}>
                {group.heading}
              </p>
              <div className="flex flex-col border-l border-line">
                {group.items.map((item) => (
                  <NavItemLink key={item.label} item={item} className={sidebarItemClass} />
                ))}
              </div>
            </div>
          ))}
        </nav>
      </aside>
      <div className="min-w-0 max-w-3xl flex-1">
        <div className="mb-6">
          <h1 className="font-display text-3xl font-semibold tracking-tight text-ink">{current?.label}</h1>
          {current?.intro && <p className="mt-1 text-sm text-ink-soft">{current.intro}</p>}
        </div>
        <Outlet />
      </div>
    </div>
  )
}
