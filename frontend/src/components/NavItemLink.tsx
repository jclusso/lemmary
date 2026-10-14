import { Link } from '@tanstack/react-router'
import { NAV_BADGE_DESCRIPTIONS, type NavItem } from '../lib/nav'

function ExternalIcon() {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.75"
      strokeLinecap="round"
      strokeLinejoin="round"
      className="ml-auto h-3.5 w-3.5 shrink-0 text-ink-faint"
      aria-hidden="true"
    >
      <path d="M7 17 17 7" />
      <path d="M8 7h9v9" />
    </svg>
  )
}

/**
 * The digits are hidden from assistive tech and replaced with a sentence,
 * because "Inbox 3" is not a useful accessible name.
 */
function NavBadge({ count, description }: { count: number; description: string }) {
  return (
    <>
      <span
        aria-hidden="true"
        className="ml-1.5 inline-flex px-1 py-0.5 align-middle text-[10px] font-semibold tabular-nums text-amber-800 ring-1 ring-inset ring-amber-800/40"
      >
        {count > 99 ? '99+' : count}
      </span>
      <span className="sr-only">{`, ${count} ${description}`}</span>
    </>
  )
}

/**
 * One nav link, in either shape. The router handles internal paths so a tap
 * does not reload the app; the PocketBase dashboard is a real page elsewhere.
 */
export function NavItemLink({
  item,
  className,
  role,
  onNavigate,
  badge,
}: {
  item: NavItem
  className: string
  role?: string
  onNavigate?: () => void
  badge?: number | null
}) {
  const label = (
    <>
      <span>{item.label}</span>
      {typeof badge === 'number' && badge > 0 && item.kind === 'route' && item.badgeKey && (
        <NavBadge count={badge} description={NAV_BADGE_DESCRIPTIONS[item.badgeKey]} />
      )}
    </>
  )

  if (item.kind === 'external') {
    return (
      <a
        href={item.href}
        target="_blank"
        rel="noopener noreferrer"
        role={role}
        className={className}
        onClick={onNavigate}
      >
        {label}
        <ExternalIcon />
      </a>
    )
  }

  return (
    <Link
      to={item.to}
      role={role}
      className={className}
      activeOptions={item.exact ? { exact: true } : undefined}
      onClick={onNavigate}
    >
      {label}
    </Link>
  )
}
