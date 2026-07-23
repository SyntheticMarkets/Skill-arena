'use client'

import Link from 'next/link'
import { usePathname, useRouter } from 'next/navigation'
import {
  Bell,
  CircleHelp,
  Gamepad2,
  Goal,
  Home,
  LogOut,
  Menu,
  PlaySquare,
  Settings,
  Trophy,
  UserRound,
  WalletCards,
  X,
} from 'lucide-react'
import { useEffect, useState } from 'react'
import { useAuth } from './auth-context'
import { useHub } from './hub-context'
import { t } from './i18n'
import { PlayerAvatar } from './player-avatar'

const navItems = [
  { label: 'Home', href: '/dashboard', icon: Home },
  { label: 'Games', href: '/games', icon: Gamepad2 },
  { label: 'Challenges', href: '/challenges', icon: Goal },
  { label: 'Tournaments', href: '/tournaments', icon: Trophy },
  { label: 'Leaderboards', href: '/leaderboards', icon: Trophy },
  { label: 'Wallet', href: '/wallet', icon: WalletCards },
  { label: 'Replay Center', href: '/replays', icon: PlaySquare },
  { label: 'Notifications', href: '/notifications', icon: Bell },
  { label: 'Profile', href: '/profile', icon: UserRound },
  { label: 'Settings', href: '/settings', icon: Settings },
  { label: 'Support', href: '/support', icon: CircleHelp },
]

const alwaysPublicPath = (pathname: string) => pathname === '/' || pathname === '/arena' || pathname.startsWith('/auth/')

export function AppShell({ children }: { children: React.ReactNode }) {
  const pathname = usePathname()
  const router = useRouter()
  const { status, mfaEnrollmentRequired, logout } = useAuth()
  const { data: hub } = useHub()
  const [menuOpen, setMenuOpen] = useState(false)
  const isPublic = alwaysPublicPath(pathname) || (status === 'guest' && pathname === '/leaderboards')

  useEffect(() => {
    if (status === 'guest' && !isPublic) {
      router.replace(`/auth/login?next=${encodeURIComponent(pathname)}`)
    }
    if (status === 'authenticated' && mfaEnrollmentRequired && pathname !== '/auth/mfa/setup') {
      router.replace('/auth/mfa/setup')
    }
  }, [isPublic, mfaEnrollmentRequired, pathname, router, status])

  if (status === 'loading') {
    return (
      <main className="boot-experience" aria-live="polite" aria-label="Skill Arena is loading">
        <div className="boot-mark" aria-hidden="true"><i /><i /><i /></div>
        <strong>SKILL ARENA</strong>
        <span>Preparing your arena</span>
        <div className="boot-progress"><i /></div>
      </main>
    )
  }

  if (isPublic) return <>{children}</>
  if (status !== 'authenticated') return null

  const displayName = hub?.profile.displayName || hub?.profile.username || 'Competitor'
  const playerInitial = displayName.slice(0, 1).toUpperCase()

  return (
    <div className="arena-shell">
      <aside className={`arena-sidebar ${menuOpen ? 'open' : ''}`}>
        <div className="sidebar-heading">
          <Link href="/dashboard" className="sa-brand" aria-label={`${t.brandName} home`}>
            <span className="sa-shield">SA</span>
            <span><strong>{t.brandName}</strong><small>Player Arena</small></span>
          </Link>
          <button className="icon-button sidebar-close" type="button" aria-label="Close navigation" onClick={() => setMenuOpen(false)}><X /></button>
        </div>
        <nav className="arena-nav" aria-label={`${t.brandName} player navigation`}>
          {navItems.map(({ label, href, icon: Icon }) => (
            <Link key={href} href={href} aria-current={pathname === href ? 'page' : undefined} onClick={() => setMenuOpen(false)}>
              <Icon aria-hidden="true" />
              <span>{label}</span>
              {href === '/notifications' && hub?.notifications.unread ? <strong className="nav-count">{hub.notifications.unread}</strong> : null}
            </Link>
          ))}
        </nav>
        <div className="sidebar-player">
          <span className="player-avatar" aria-hidden="true">
            <PlayerAvatar avatarKey={hub?.profile.avatarUrl} fallback={playerInitial} />
          </span>
          <span><strong>{displayName}</strong><small>{hub ? `${hub.progression.leagueTier} / Level ${hub.progression.level}` : 'Loading profile'}</small></span>
        </div>
      </aside>
      <div className="arena-main">
        <header className="arena-topbar">
          <div className="topbar-identity">
            <button className="icon-button mobile-menu" type="button" aria-label="Open navigation" onClick={() => setMenuOpen(true)}><Menu /></button>
            <div><strong>{hub ? `${hub.progression.leagueTier} League` : 'Arena Hub'}</strong><span>{hub ? `${hub.progression.xp.toLocaleString()} XP / Trust ${hub.progression.trustScore.toFixed(0)}` : 'Synchronizing player state'}</span></div>
          </div>
          <nav aria-label="Account controls">
            <Link className="icon-button notification-control" href="/notifications" aria-label={`${hub?.notifications.unread ?? 0} unread notifications`}>
              <Bell />
              {hub?.notifications.unread ? <span>{hub.notifications.unread}</span> : null}
            </Link>
            <Link className="icon-button" href="/profile" aria-label="Profile"><UserRound /></Link>
            <button className="icon-button" type="button" aria-label="Log out" onClick={() => void logout().then(() => router.push('/'))}><LogOut /></button>
          </nav>
        </header>
        {children}
      </div>
      {menuOpen ? <button className="sidebar-scrim" aria-label="Close navigation" onClick={() => setMenuOpen(false)} /> : null}
    </div>
  )
}
