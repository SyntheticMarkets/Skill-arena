'use client'

import Link from 'next/link'
import { usePathname, useRouter } from 'next/navigation'
import { useEffect, useState } from 'react'
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

function tokenRole(token: string | null) {
  if (!token) return ''
  try {
    const payload = token.split('.')[1]
    const decoded = JSON.parse(window.atob(payload.replace(/-/g, '+').replace(/_/g, '/')))
    return typeof decoded.role === 'string' ? decoded.role : ''
  } catch {
    return ''
  }
}

function roleRank(role: string) {
  switch (role) {
    case 'super_admin':
      return 100
    case 'admin':
      return 90
    case 'treasury_manager':
      return 70
    case 'fraud_analyst':
      return 60
    case 'support':
      return 50
    case 'moderator':
      return 40
    default:
      return 10
  }
}

export function AppShell({ children }: { children: React.ReactNode }) {
  const pathname = usePathname()
  const router = useRouter()
  const isMarketing = pathname === '/'
  const [checkedAuth, setCheckedAuth] = useState(!isMarketing)
  const [role, setRole] = useState('')
  const isAdmin = roleRank(role) >= roleRank('admin')

  useEffect(() => {
    const token = window.localStorage.getItem('skill-arena-token')
    const currentRole = tokenRole(token)
    setRole(currentRole)
    if (!isMarketing) {
      if (pathname === '/admin' && roleRank(currentRole) < roleRank('admin')) {
        router.replace('/dashboard')
      }
      setCheckedAuth(true)
      return
    }
    if (token) {
      router.replace('/dashboard')
      return
    }
    setCheckedAuth(true)
  }, [isMarketing, router])

  if (isMarketing) {
    return checkedAuth ? <>{children}</> : <main className="marketing-loading" aria-label={t.loading} />
  }

  return (
    <div className="arena-shell">
      <aside className="arena-sidebar">
        <Link href="/dashboard" className="sa-brand" aria-label={`${t.brandName} ${t.navDashboard}`}>
          <span className="sa-shield">SA</span>
          <span>
            <strong>{t.brandName}</strong>
            <small>{t.brandSubtitle}</small>
          </span>
        </Link>
        <nav className="arena-nav" aria-label={`${t.brandName} navigation`}>
          {navItems.map(([label, href]) => (
            <Link key={href} href={href}>
              {label}
            </Link>
          ))}
          {isAdmin ? <Link href="/admin">{t.navAdmin}</Link> : null}
        </nav>
      </aside>
      <div className="arena-main">
        <header className="arena-topbar">
          <div>
            <strong>{t.topbarSeason}</strong>
            <span>{t.topbarArena}</span>
          </div>
          <nav>
            {role ? null : <Link href="/auth/login">{t.navLogin}</Link>}
            {role ? null : <Link href="/auth/register">{t.navRegister}</Link>}
          </nav>
        </header>
        {children}
      </div>
    </div>
  )
}
