'use client'

import { ArrowDownLeft, ArrowUpRight, Clock3, ShieldCheck, WalletCards } from 'lucide-react'
import { useEffect, useState } from 'react'
import { useHub } from '../hub-context'
import { apiFetch } from '../lib/api'

type Transaction = {
  id: string
  transactionType: string
  amount: number
  balanceAfter: number
  currency: string
  reference?: string
  createdAt: string
}

function money(value: number, currency: string) {
  return new Intl.NumberFormat(undefined, { style: 'currency', currency }).format(value)
}

export default function WalletPage() {
  const { data, status: hubStatus } = useHub()
  const [transactions, setTransactions] = useState<Transaction[]>([])
  const [status, setStatus] = useState<'loading' | 'ready' | 'error'>('loading')

  useEffect(() => {
    const timer = window.setTimeout(() => {
      void apiFetch<Transaction[]>('/api/v1/wallet/transactions')
        .then((response) => { setTransactions(response); setStatus('ready') })
        .catch(() => setStatus('error'))
    }, 0)
    return () => window.clearTimeout(timer)
  }, [])

  if (hubStatus === 'loading' || !data) return <main className="hub-page"><div className="inline-loading">Loading wallet state...</div></main>

  return (
    <main className="hub-page">
      <section className="subpage-heading">
        <div><span className="eyebrow">Wallet overview</span><h1>Every balance has a state.</h1><p>Review available funds, pending movement, verification status, and the transaction trail recorded by the ledger.</p></div>
        <WalletCards aria-hidden="true" />
      </section>

      <section className="wallet-overview">
        <article className="wallet-primary">
          <span>Available balance</span>
          <strong>{money(data.wallet.availableBalance, data.wallet.currency)}</strong>
          <small>{data.wallet.accountStatus} account / {data.wallet.verificationStatus} verification</small>
        </article>
        <article><ArrowDownLeft /><span>Pending deposits</span><strong>{money(data.wallet.pendingDeposits, data.wallet.currency)}</strong></article>
        <article><ArrowUpRight /><span>Pending withdrawals</span><strong>{money(data.wallet.pendingWithdrawals, data.wallet.currency)}</strong></article>
        <article><ShieldCheck /><span>Live eligibility</span><strong>{data.eligibility.liveEligible ? 'Eligible' : 'Locked'}</strong></article>
      </section>

      <section className="hub-section" aria-labelledby="wallet-history-title">
        <div className="hub-section-heading"><div><span className="eyebrow">Transaction history</span><h2 id="wallet-history-title">Ledger activity</h2></div></div>
        {status === 'loading' ? <div className="inline-loading">Loading transactions...</div> : null}
        {status === 'error' ? <div className="form-message error">Transaction history is temporarily unavailable.</div> : null}
        {status === 'ready' && transactions.length === 0 ? <p className="hub-empty">No wallet transaction has been recorded.</p> : null}
        <div className="transaction-list">
          {transactions.map((transaction) => (
            <article key={transaction.id}>
              <span className="transaction-icon"><Clock3 /></span>
              <span><strong>{transaction.transactionType.replace(/_/g, ' ')}</strong><small>{transaction.reference || transaction.id} / {new Date(transaction.createdAt).toLocaleString()}</small></span>
              <span><strong>{money(transaction.amount, transaction.currency)}</strong><small>Balance {money(transaction.balanceAfter, transaction.currency)}</small></span>
            </article>
          ))}
        </div>
      </section>
    </main>
  )
}
