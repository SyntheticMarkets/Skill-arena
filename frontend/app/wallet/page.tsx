'use client'

import { useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'

const apiBase = process.env.NEXT_PUBLIC_API_BASE_URL ?? 'http://localhost:8080'
type Profile = { liveBalance: number; availableLiveBalance: number; demoBalance: number; availableDemoBalance: number; pendingWithdrawals: number }
type Ledger = { id: string; transactionType: string; amount: number; balanceAfter: number; reference?: string; createdAt: string }
const authHeaders = () => { const token = window.localStorage.getItem('skill-arena-token'); return token ? { Authorization: `Bearer ${token}` } : null }

export default function WalletPage() {
  const router = useRouter()
  const [profile, setProfile] = useState<Profile | null>(null)
  const [ledger, setLedger] = useState<Ledger[]>([])
  useEffect(() => {
    const headers = authHeaders()
    if (!headers) { router.replace('/auth/login'); return }
    Promise.all([fetch(`${apiBase}/api/v1/profile`, { headers }), fetch(`${apiBase}/api/v1/wallet/transactions`, { headers })]).then(async ([p, l]) => { setProfile(await p.json()); setLedger(await l.json()) }).catch(() => undefined)
  }, [router])
  return (
    <main className="page-shell">
      <section className="dashboard-command"><div><span className="eyebrow">Wallet</span><h1>Balances and ledger</h1><p>Server-calculated balances and token movement history.</p></div></section>
      <section className="metric-grid"><article className="metric-card"><span>Available Live</span><strong>{profile?.availableLiveBalance.toFixed(2) ?? '-'}</strong></article><article className="metric-card"><span>Demo</span><strong>{profile?.availableDemoBalance.toFixed(2) ?? '-'}</strong></article><article className="metric-card"><span>Pending Withdrawals</span><strong>{profile?.pendingWithdrawals.toFixed(2) ?? '-'}</strong></article></section>
      <section className="panel-large"><table className="leaderboard-table"><thead><tr><th>Time</th><th>Type</th><th>Amount</th><th>Balance</th><th>Reference</th></tr></thead><tbody>{ledger.map((entry) => <tr key={entry.id}><td>{new Date(entry.createdAt).toLocaleString()}</td><td>{entry.transactionType}</td><td>{entry.amount.toFixed(2)}</td><td>{entry.balanceAfter.toFixed(2)}</td><td>{entry.reference || '-'}</td></tr>)}</tbody></table></section>
    </main>
  )
}
