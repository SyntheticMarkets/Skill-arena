'use client'

import Link from 'next/link'
import { usePathname, useRouter } from 'next/navigation'
import { Bell, LogOut, Menu, UserRound, X } from 'lucide-react'
import { useEffect, useState } from 'react'
import { useAuth } from './auth-context'
import { t } from './i18n'

const navItems = [
  [t.navDashboard, '/dashboard'],
  [t.navGames, '/games'],
  [t.navChallenges, '/challenges'],
  [t.navTournaments, '/tournaments'],
  [t.navLeaderboards, '/leaderboards'],
  [t.navWallet, '/wallet'],
  [t.navReplays, '/replays'],
  [t.navProfile, '/profile'],
  [t.navSettings, '/settings'],
]

const publicPath = (pathname: string) => pathname === '/' || pathname === '/arena' || pathname.startsWith('/auth/')

export function AppShell({ children }: { children: React.ReactNode }) {
  const pathname = usePathname()
  const router = useRouter()
  const { status, user, mfaEnrollmentRequired, logout } = useAuth()
  const [menuOpen, setMenuOpen] = useState(false)
  const isPublic = publicPath(pathname)

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

  const isAdmin = user?.role === 'admin' || user?.role === 'super_admin'

  return (
    <div className="arena-shell">
      <aside className={`arena-sidebar ${menuOpen ? 'open' : ''}`}>
        <div className="sidebar-heading">
          <Link href="/dashboard" className="sa-brand" aria-label={`${t.brandName} ${t.navDashboard}`}>
            <span className="sa-shield">SA</span>
            <span><strong>{t.brandName}</strong><small>{t.brandSubtitle}</small></span>
          </Link>
          <button className="icon-button sidebar-close" type="button" aria-label="Close navigation" onClick={() => setMenuOpen(false)}><X /></button>
        </div>
        <nav className="arena-nav" aria-label={`${t.brandName} navigation`}>
          {navItems.map(([label, href]) => <Link key={href} href={href} aria-current={pathname === href ? 'page' : undefined} onClick={() => setMenuOpen(false)}>{label}</Link>)}
          {isAdmin ? <Link href="/admin" onClick={() => setMenuOpen(false)}>{t.navAdmin}</Link> : null}
        </nav>
      </aside>
      <div className="arena-main">
        <header className="arena-topbar">
          <button className="icon-button mobile-menu" type="button" aria-label="Open navigation" onClick={() => setMenuOpen(true)}><Menu /></button>
          <div><strong>{t.topbarSeason}</strong><span>{t.topbarArena}</span></div>
          <nav aria-label="Account controls">
            <button className="icon-button" type="button" aria-label="Notifications"><Bell /></button>
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
