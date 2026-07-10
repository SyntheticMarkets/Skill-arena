'use client'

import { useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'

const apiBase = process.env.NEXT_PUBLIC_API_BASE_URL ?? 'http://localhost:8080'
type Profile = { email: string; level: number; prestige: number; leagueTier: string; eloRating: number; wins: number; losses: number; trustScore: number; kycStatus: string }
const authHeaders = () => { const token = window.localStorage.getItem('skill-arena-token'); return token ? { Authorization: `Bearer ${token}` } : null }

export default function ProfilePage() {
  const router = useRouter()
  const [profile, setProfile] = useState<Profile | null>(null)
  useEffect(() => { const headers = authHeaders(); if (!headers) { router.replace('/auth/login'); return }; fetch(`${apiBase}/api/v1/profile`, { headers }).then((r) => r.json()).then(setProfile).catch(() => undefined) }, [router])
  return <main className="page-shell"><section className="dashboard-command"><div><span className="eyebrow">Profile</span><h1>{profile?.email ?? 'Player Profile'}</h1><p>Identity, progression, league, record, and account status.</p></div></section><section className="metric-grid"><article className="metric-card"><span>Level</span><strong>{profile?.level ?? '-'}</strong><small>Prestige {profile?.prestige ?? '-'}</small></article><article className="metric-card"><span>League</span><strong>{profile?.leagueTier ?? '-'}</strong><small>ELO {profile?.eloRating ?? '-'}</small></article><article className="metric-card"><span>Record</span><strong>{profile ? `${profile.wins}W / ${profile.losses}L` : '-'}</strong></article><article className="metric-card"><span>Trust</span><strong>{profile?.trustScore.toFixed(1) ?? '-'}</strong><small>{profile?.kycStatus ?? ''}</small></article></section></main>
}
