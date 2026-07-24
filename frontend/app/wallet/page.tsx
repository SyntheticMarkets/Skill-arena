'use client'

import {
  ArrowDownLeft, ArrowUpRight, BadgeCheck, Ban, ChevronRight, Clock3,
  Download, FileCheck2, Landmark, RefreshCw, ShieldCheck, SlidersHorizontal, Upload, WalletCards,
} from 'lucide-react'
import { FormEvent, useCallback, useEffect, useMemo, useState } from 'react'
import { apiBase, apiFetch, ApiError } from '../lib/api'

type Minor = number
type PaymentMethod = { id: string; type: string; displayName: string; currency: string; available: boolean; reason?: string }
type Deposit = {
  id: string; amountMinor: Minor; currency: string; method: string; status: string
  checkoutUrl?: string; requestedAt: string; updatedAt: string; completedAt?: string
}
type Withdrawal = {
  id: string; amountMinor: Minor; feeMinor: Minor; currency: string; method: string; status: string
  requestedAt: string; updatedAt: string; completedAt?: string
}
type Limits = {
  currency: string
  dailyDepositMinor: Minor; monthlyDepositMinor: Minor
  dailyWithdrawalMinor: Minor; monthlyWithdrawalMinor: Minor
  depositUsedTodayMinor: Minor; depositUsedMonthMinor: Minor
  withdrawalUsedTodayMinor: Minor; withdrawalUsedMonthMinor: Minor
  coolingOffUntil?: string; selfExcludedUntil?: string
}
type Assessment = {
  status: string; country: string; occupation: string; sourceOfFunds: string
  riskClassification: string; verificationStatus: string; responsibleGamingStatus: string
}
type Overview = {
  wallet: {
    currency: string; availableMinor: Minor; pendingDepositMinor: Minor; pendingWithdrawalMinor: Minor
    lockedMinor: Minor; lifetimeDepositMinor: Minor; lifetimeWithdrawalMinor: Minor; updatedAt: string
  }
  assessment: Assessment
  limits: Limits
  verificationStatus: string
  paymentMethods: PaymentMethod[]
  deposits: Deposit[]
  withdrawals: Withdrawal[]
}
type Entry = {
  id: string; account: string; direction: string; amountMinor: Minor; currency: string
  balanceAfterMinor: Minor; referenceType: string; referenceId: string; description: string; createdAt: string
}
type Statement = {
  id: string; periodStart: string; periodEnd: string; currency: string
  openingMinor: Minor; closingMinor: Minor; totalCreditMinor: Minor; totalDebitMinor: Minor; entries: Entry[]
}
type Evidence = {
  id: string; type: string; contentType: string; sizeBytes: number; sha256: string; status: string; createdAt: string
}
type Artifact = {
  id: string; type: string; contentType: string; sizeBytes: number; sha256: string; createdAt: string
}
type View = 'overview' | 'deposit' | 'withdraw' | 'activity' | 'limits' | 'assessment'

function money(minor: Minor, currency = 'ZAR') {
  return new Intl.NumberFormat(undefined, { style: 'currency', currency }).format(minor / 100)
}

function label(value: string) {
  return value.replace(/_/g, ' ').replace(/\b\w/g, (letter) => letter.toUpperCase())
}

function idempotencyKey() {
  return `wallet-${crypto.randomUUID()}`
}

function statusTone(status: string) {
  if (['completed', 'complete', 'approved', 'verified', 'active'].includes(status)) return 'success'
  if (['failed', 'rejected', 'restricted'].includes(status)) return 'danger'
  return 'pending'
}

export default function WalletPage() {
  const [view, setView] = useState<View>('overview')
  const [overview, setOverview] = useState<Overview | null>(null)
  const [entries, setEntries] = useState<Entry[]>([])
  const [statement, setStatement] = useState<Statement | null>(null)
  const [evidence, setEvidence] = useState<Evidence[]>([])
  const [status, setStatus] = useState<'loading' | 'ready' | 'error'>('loading')
  const [message, setMessage] = useState('')

  const load = useCallback(async () => {
    setStatus('loading')
    try {
      const [snapshot, journal, evidenceItems] = await Promise.all([
        apiFetch<Overview>('/api/v1/financial/overview'),
        apiFetch<Entry[]>('/api/v1/financial/transactions'),
        apiFetch<Evidence[]>('/api/v1/financial/evidence'),
      ])
      setOverview(snapshot)
      setEntries(journal)
      setEvidence(evidenceItems)
      setStatus('ready')
    } catch (cause) {
      setMessage(cause instanceof Error ? cause.message : 'Your financial account could not be loaded.')
      setStatus('error')
    }
  }, [])

  useEffect(() => {
    const timer = window.setTimeout(() => void load(), 0)
    return () => window.clearTimeout(timer)
  }, [load])

  if (status === 'loading') return <main className="hub-page"><div className="wallet-loading"><RefreshCw /> Securing your financial account...</div></main>
  if (status === 'error' || !overview) {
    return <main className="hub-page"><div className="wallet-error" role="alert"><ShieldCheck /><h1>Wallet unavailable</h1><p>{message}</p><button className="button secondary" onClick={() => { setMessage(''); void load() }}><RefreshCw /> Retry</button></div></main>
  }

  return (
    <main className="hub-page financial-page">
      <section className="subpage-heading financial-heading">
        <div><span className="eyebrow">Financial platform</span><h1>Your money, with every stage visible.</h1><p>Available funds, pending movement, limits, verification, and immutable ledger activity stay in one controlled account.</p></div>
        <WalletCards aria-hidden="true" />
      </section>

      <nav className="financial-tabs" aria-label="Wallet sections">
        {([
          ['overview', 'Overview'], ['deposit', 'Deposit'], ['withdraw', 'Withdraw'],
          ['activity', 'Activity'], ['limits', 'Limits'], ['assessment', 'Assessment'],
        ] as Array<[View, string]>).map(([id, text]) => (
          <button key={id} className={view === id ? 'active' : ''} onClick={() => { setMessage(''); setView(id) }}>{text}</button>
        ))}
      </nav>

      {message ? <div className="financial-message" role="status">{message}</div> : null}
      {view === 'overview' ? <OverviewView data={overview} onNavigate={setView} /> : null}
      {view === 'deposit' ? <MoneyIntent kind="deposit" data={overview} onComplete={load} setMessage={setMessage} /> : null}
      {view === 'withdraw' ? <MoneyIntent kind="withdraw" data={overview} onComplete={load} setMessage={setMessage} /> : null}
      {view === 'activity' ? <ActivityView entries={entries} statement={statement} onStatement={setStatement} setMessage={setMessage} /> : null}
      {view === 'limits' ? <LimitsView limits={overview.limits} onComplete={load} setMessage={setMessage} /> : null}
      {view === 'assessment' ? <AssessmentView assessment={overview.assessment} evidence={evidence} onComplete={load} setMessage={setMessage} /> : null}
    </main>
  )
}

function OverviewView({ data, onNavigate }: { data: Overview; onNavigate: (view: View) => void }) {
  const latest = useMemo(() => [
    ...data.deposits.map((item) => ({ ...item, kind: 'deposit' as const })),
    ...data.withdrawals.map((item) => ({ ...item, kind: 'withdrawal' as const })),
  ].sort((a, b) => Date.parse(b.requestedAt) - Date.parse(a.requestedAt)).slice(0, 4), [data])

  return (
    <>
      <section className="wallet-overview financial-balances">
        <article className="wallet-primary"><span>Available balance</span><strong>{money(data.wallet.availableMinor, data.wallet.currency)}</strong><small>Settled funds available for eligible competition</small></article>
        <article><ArrowDownLeft /><span>Pending deposits</span><strong>{money(data.wallet.pendingDepositMinor, data.wallet.currency)}</strong><small>Awaiting provider settlement</small></article>
        <article><ArrowUpRight /><span>Pending withdrawals</span><strong>{money(data.wallet.pendingWithdrawalMinor, data.wallet.currency)}</strong><small>Reserved while the request progresses</small></article>
        <article><ShieldCheck /><span>Account protection</span><strong>{label(data.assessment.status)}</strong><small>{label(data.verificationStatus)} verification</small></article>
      </section>

      <section className="financial-grid">
        <div className="financial-panel">
          <div className="hub-section-heading"><div><span className="eyebrow">Next financial action</span><h2>Keep your account ready</h2></div></div>
          {data.assessment.status !== 'complete' ? (
            <button className="financial-action" onClick={() => onNavigate('assessment')}><span><ShieldCheck /><strong>Complete financial assessment</strong><small>Required before live deposits or withdrawals.</small></span><ChevronRight /></button>
          ) : (
            <button className="financial-action" onClick={() => onNavigate('deposit')}><span><ArrowDownLeft /><strong>Deposit securely</strong><small>Payment Core selects an approved provider.</small></span><ChevronRight /></button>
          )}
          <button className="financial-action" onClick={() => onNavigate('limits')}><span><SlidersHorizontal /><strong>Review your limits</strong><small>Lower limits or start a cooling-off period.</small></span><ChevronRight /></button>
        </div>

        <div className="financial-panel">
          <div className="hub-section-heading"><div><span className="eyebrow">Account status</span><h2>Eligibility controls</h2></div></div>
          <dl className="financial-status-list">
            <div><dt>Financial assessment</dt><dd className={statusTone(data.assessment.status)}>{label(data.assessment.status)}</dd></div>
            <div><dt>Identity verification</dt><dd className={statusTone(data.verificationStatus)}>{label(data.verificationStatus)}</dd></div>
            <div><dt>Responsible gaming</dt><dd className={statusTone(data.assessment.responsibleGamingStatus)}>{label(data.assessment.responsibleGamingStatus)}</dd></div>
            <div><dt>Risk classification</dt><dd>{label(data.assessment.riskClassification)}</dd></div>
          </dl>
        </div>
      </section>

      <section className="financial-panel financial-recent">
        <div className="hub-section-heading"><div><span className="eyebrow">Movement</span><h2>Recent requests</h2></div><button className="text-button" onClick={() => onNavigate('activity')}>View ledger <ChevronRight /></button></div>
        {latest.length === 0 ? <div className="financial-empty"><Clock3 /><p>No money movement has been requested. Your immutable ledger will appear here after settlement.</p></div> : (
          <div className="financial-request-list">{latest.map((item) => <RequestRow key={`${item.kind}-${item.id}`} item={item} />)}</div>
        )}
      </section>
    </>
  )
}

function MoneyIntent({ kind, data, onComplete, setMessage }: {
  kind: 'deposit' | 'withdraw'; data: Overview; onComplete: () => Promise<void>; setMessage: (value: string) => void
}) {
  const methods = data.paymentMethods.filter((method) => method.available)
  const [amount, setAmount] = useState('')
  const [method, setMethod] = useState(methods[0]?.id ?? '')
  const [submitting, setSubmitting] = useState(false)
  const records = kind === 'deposit' ? data.deposits : data.withdrawals

  async function submit(event: FormEvent) {
    event.preventDefault()
    const amountMinor = Math.round(Number(amount) * 100)
    if (!Number.isSafeInteger(amountMinor) || amountMinor <= 0) {
      setMessage('Enter a valid amount with no more than two decimal places.')
      return
    }
    setSubmitting(true)
    setMessage('')
    try {
      const response = await apiFetch<Deposit | Withdrawal>(`/api/v1/financial/${kind === 'deposit' ? 'deposits' : 'withdrawals'}`, {
        method: 'POST',
        headers: { 'Idempotency-Key': idempotencyKey() },
        body: JSON.stringify({ amountMinor, currency: data.wallet.currency, method }),
      })
      if (kind === 'deposit' && 'checkoutUrl' in response && response.checkoutUrl) {
        window.location.assign(response.checkoutUrl)
        return
      }
      setMessage(kind === 'deposit' ? 'Deposit session created. Provider confirmation is pending.' : 'Withdrawal submitted. Manual review is pending.')
      setAmount('')
      await onComplete()
    } catch (cause) {
      setMessage(cause instanceof ApiError ? cause.message : 'The financial request could not be completed.')
    } finally {
      setSubmitting(false)
    }
  }

  const disabledReason = data.assessment.status !== 'complete'
    ? 'Complete the financial assessment before moving live funds.'
    : methods.length === 0 ? 'No approved payment provider is configured for this environment.' : ''

  return (
    <section className="financial-workspace">
      <div className="financial-form-panel">
        <span className="eyebrow">{kind === 'deposit' ? 'Add live funds' : 'Request withdrawal'}</span>
        <h2>{kind === 'deposit' ? 'Create a secure provider session' : 'Move funds through review and settlement'}</h2>
        <p>{kind === 'deposit' ? 'Skill Arena selects the provider. Your balance changes only after a signed settlement callback.' : 'Funds are reserved once. Policy, treasury, and provider stages remain visible until completion.'}</p>
        {disabledReason ? <div className="financial-blocker"><Ban /><span>{disabledReason}</span></div> : null}
        <form className="financial-form" onSubmit={submit}>
          <label htmlFor={`${kind}-amount`}>Amount</label>
          <div className="money-input"><span>{data.wallet.currency}</span><input id={`${kind}-amount`} inputMode="decimal" value={amount} onChange={(event) => setAmount(event.target.value)} placeholder="0.00" disabled={Boolean(disabledReason)} required /></div>
          <label htmlFor={`${kind}-method`}>Payment method</label>
          <select id={`${kind}-method`} value={method} onChange={(event) => setMethod(event.target.value)} disabled={Boolean(disabledReason)}>
            {methods.length === 0 ? <option value="">Unavailable</option> : methods.map((item) => <option key={item.id} value={item.id}>{item.displayName}</option>)}
          </select>
          <button className="button primary" disabled={Boolean(disabledReason) || submitting}>{submitting ? <><RefreshCw className="spin" /> Submitting</> : kind === 'deposit' ? <><ArrowDownLeft /> Continue to provider</> : <><ArrowUpRight /> Submit withdrawal</>}</button>
        </form>
      </div>
      <div className="financial-panel">
        <div className="hub-section-heading"><div><span className="eyebrow">Lifecycle</span><h2>{kind === 'deposit' ? 'Deposit requests' : 'Withdrawal requests'}</h2></div></div>
        {records.length === 0 ? <div className="financial-empty"><Landmark /><p>No {kind} request has been created.</p></div> : (
          <div className="financial-request-list">
            {kind === 'deposit'
              ? data.deposits.map((item) => <RequestRow key={item.id} item={{ ...item, kind: 'deposit' }} />)
              : data.withdrawals.map((item) => <RequestRow key={item.id} item={{ ...item, kind: 'withdrawal' }} />)}
          </div>
        )}
      </div>
    </section>
  )
}

function RequestRow({ item }: { item: (Deposit & { kind: 'deposit' }) | (Withdrawal & { kind: 'withdrawal' }) }) {
  const stages = item.kind === 'deposit'
    ? ['pending_provider', 'pending_verification', 'completed']
    : ['pending_review', 'approved', 'processing', 'completed']
  const currentIndex = stages.indexOf(item.status)
  return (
    <article className="financial-request">
      <div className={`request-icon ${item.kind}`} >{item.kind === 'deposit' ? <ArrowDownLeft /> : <ArrowUpRight />}</div>
      <div><strong>{item.kind === 'deposit' ? 'Deposit' : 'Withdrawal'} · {money(item.amountMinor, item.currency)}</strong><small>{new Date(item.requestedAt).toLocaleString()} · {label(item.method)}</small></div>
      <div className="request-progress" aria-label={`${label(item.status)} status`}>
        <span className={`status-badge ${statusTone(item.status)}`}>{label(item.status)}</span>
        <span className="stage-dots" aria-hidden="true">{stages.map((stage, index) => <i key={stage} className={index <= currentIndex ? 'complete' : ''} />)}</span>
      </div>
    </article>
  )
}

function ActivityView({ entries, statement, onStatement, setMessage }: {
  entries: Entry[]; statement: Statement | null; onStatement: (value: Statement) => void; setMessage: (value: string) => void
}) {
  async function generate() {
    try {
      const result = await apiFetch<Statement>('/api/v1/financial/statements')
      onStatement(result)
      setMessage('Statement generated from the immutable financial journal.')
    } catch (cause) {
      setMessage(cause instanceof Error ? cause.message : 'Statement generation failed.')
    }
  }

  async function exportStatement() {
    try {
      const artifact = await apiFetch<Artifact>('/api/v1/financial/statements/export', { method: 'POST' })
      const response = await fetch(`${apiBase}/api/v1/financial/artifacts/${artifact.id}`, { credentials: 'include' })
      if (!response.ok) throw new Error('The stored statement could not be downloaded.')
      const blob = await response.blob()
      const href = URL.createObjectURL(blob)
      const anchor = document.createElement('a')
      anchor.href = href
      anchor.download = `skill-arena-statement-${artifact.id}.csv`
      anchor.click()
      URL.revokeObjectURL(href)
      setMessage('Statement stored securely and downloaded from object storage.')
    } catch (cause) {
      setMessage(cause instanceof Error ? cause.message : 'Statement export failed.')
    }
  }

  return (
    <section className="financial-panel">
      <div className="hub-section-heading"><div><span className="eyebrow">Immutable journal</span><h2>Transaction activity</h2></div><div className="financial-heading-actions"><button className="button secondary compact" onClick={() => void generate()}><FileCheck2 /> Preview statement</button><button className="button primary compact" onClick={() => void exportStatement()}><Download /> Export CSV</button></div></div>
      {statement ? <div className="statement-summary"><span>Opening <strong>{money(statement.openingMinor, statement.currency)}</strong></span><span>Credits <strong>{money(statement.totalCreditMinor, statement.currency)}</strong></span><span>Debits <strong>{money(statement.totalDebitMinor, statement.currency)}</strong></span><span>Closing <strong>{money(statement.closingMinor, statement.currency)}</strong></span></div> : null}
      {entries.length === 0 ? <div className="financial-empty"><Clock3 /><p>No settled transaction exists yet.</p></div> : (
        <div className="transaction-list financial-transactions">{entries.map((entry) => (
          <article key={entry.id}>
            <span className={`transaction-icon ${entry.direction}`}>{entry.direction === 'credit' ? <ArrowDownLeft /> : <ArrowUpRight />}</span>
            <span><strong>{entry.description}</strong><small>{label(entry.referenceType)} · {new Date(entry.createdAt).toLocaleString()}</small></span>
            <span><strong>{entry.direction === 'credit' ? '+' : '-'}{money(entry.amountMinor, entry.currency)}</strong><small>Balance {money(entry.balanceAfterMinor, entry.currency)}</small></span>
          </article>
        ))}</div>
      )}
    </section>
  )
}

function LimitsView({ limits, onComplete, setMessage }: { limits: Limits; onComplete: () => Promise<void>; setMessage: (value: string) => void }) {
  const [dailyDeposit, setDailyDeposit] = useState(String(limits.dailyDepositMinor / 100))
  const [monthlyDeposit, setMonthlyDeposit] = useState(String(limits.monthlyDepositMinor / 100))
  const [dailyWithdrawal, setDailyWithdrawal] = useState(String(limits.dailyWithdrawalMinor / 100))
  const [monthlyWithdrawal, setMonthlyWithdrawal] = useState(String(limits.monthlyWithdrawalMinor / 100))
  const [coolingOffDays, setCoolingOffDays] = useState('0')
  const [selfExcludeDays, setSelfExcludeDays] = useState('0')

  async function submit(event: FormEvent) {
    event.preventDefault()
    try {
      await apiFetch('/api/v1/financial/limits', {
        method: 'PUT',
        body: JSON.stringify({
          dailyDepositMinor: Math.round(Number(dailyDeposit) * 100),
          monthlyDepositMinor: Math.round(Number(monthlyDeposit) * 100),
          dailyWithdrawalMinor: Math.round(Number(dailyWithdrawal) * 100),
          monthlyWithdrawalMinor: Math.round(Number(monthlyWithdrawal) * 100),
          coolingOffDays: Number(coolingOffDays), selfExcludeDays: Number(selfExcludeDays),
        }),
      })
      setMessage('Your lower limits and responsible gaming controls are active.')
      await onComplete()
    } catch (cause) {
      setMessage(cause instanceof Error ? cause.message : 'Limits could not be updated.')
    }
  }

  return (
    <section className="financial-workspace">
      <form className="financial-form-panel financial-limit-form" onSubmit={submit}>
        <span className="eyebrow">Responsible gaming</span><h2>Set boundaries before competition</h2><p>Limits can be lowered immediately. Increases require compliance review and are never applied silently.</p>
        <div className="limit-grid">
          <label>Daily deposit<input inputMode="decimal" value={dailyDeposit} onChange={(event) => setDailyDeposit(event.target.value)} /></label>
          <label>Monthly deposit<input inputMode="decimal" value={monthlyDeposit} onChange={(event) => setMonthlyDeposit(event.target.value)} /></label>
          <label>Daily withdrawal<input inputMode="decimal" value={dailyWithdrawal} onChange={(event) => setDailyWithdrawal(event.target.value)} /></label>
          <label>Monthly withdrawal<input inputMode="decimal" value={monthlyWithdrawal} onChange={(event) => setMonthlyWithdrawal(event.target.value)} /></label>
          <label>Cooling-off period<select value={coolingOffDays} onChange={(event) => setCoolingOffDays(event.target.value)}><option value="0">No change</option><option value="1">24 hours</option><option value="7">7 days</option><option value="30">30 days</option></select></label>
          <label>Self-exclusion<select value={selfExcludeDays} onChange={(event) => setSelfExcludeDays(event.target.value)}><option value="0">No change</option><option value="30">30 days</option><option value="180">6 months</option><option value="365">1 year</option></select></label>
        </div>
        <button className="button primary"><SlidersHorizontal /> Apply controls</button>
      </form>
      <div className="financial-panel">
        <div className="hub-section-heading"><div><span className="eyebrow">Current usage</span><h2>Limit position</h2></div></div>
        <LimitBar label="Deposits today" used={limits.depositUsedTodayMinor} limit={limits.dailyDepositMinor} currency={limits.currency} />
        <LimitBar label="Deposits this month" used={limits.depositUsedMonthMinor} limit={limits.monthlyDepositMinor} currency={limits.currency} />
        <LimitBar label="Withdrawals today" used={limits.withdrawalUsedTodayMinor} limit={limits.dailyWithdrawalMinor} currency={limits.currency} />
        <LimitBar label="Withdrawals this month" used={limits.withdrawalUsedMonthMinor} limit={limits.monthlyWithdrawalMinor} currency={limits.currency} />
        {limits.coolingOffUntil ? <div className="financial-blocker"><Clock3 />Cooling-off active until {new Date(limits.coolingOffUntil).toLocaleString()}</div> : null}
        {limits.selfExcludedUntil ? <div className="financial-blocker"><Ban />Self-exclusion active until {new Date(limits.selfExcludedUntil).toLocaleDateString()}</div> : null}
      </div>
    </section>
  )
}

function LimitBar({ label: text, used, limit, currency }: { label: string; used: Minor; limit: Minor; currency: string }) {
  const percent = limit > 0 ? Math.min(100, Math.round((used / limit) * 100)) : 100
  return <div className="limit-bar"><div><strong>{text}</strong><span>{money(used, currency)} of {money(limit, currency)}</span></div><progress max="100" value={percent}>{percent}%</progress></div>
}

function AssessmentView({ assessment, evidence, onComplete, setMessage }: {
  assessment: Assessment; evidence: Evidence[]; onComplete: () => Promise<void>; setMessage: (value: string) => void
}) {
  const [country, setCountry] = useState(assessment.country || 'ZA')
  const [occupation, setOccupation] = useState(assessment.occupation || '')
  const [source, setSource] = useState(assessment.sourceOfFunds || '')
  const [evidenceType, setEvidenceType] = useState('identity')
  const [evidenceFile, setEvidenceFile] = useState<File | null>(null)
  const [uploading, setUploading] = useState(false)

  async function submit(event: FormEvent) {
    event.preventDefault()
    try {
      await apiFetch('/api/v1/financial/assessment', {
        method: 'PUT', body: JSON.stringify({ country, occupation, sourceOfFunds: source }),
      })
      setMessage('Assessment submitted. Financial review is now pending.')
      await onComplete()
    } catch (cause) {
      setMessage(cause instanceof Error ? cause.message : 'Assessment could not be submitted.')
    }
  }

  async function uploadEvidence(event: FormEvent) {
    event.preventDefault()
    if (!evidenceFile) {
      setMessage('Choose a PDF, JPEG, or PNG before uploading.')
      return
    }
    setUploading(true)
    try {
      const form = new FormData()
      form.set('type', evidenceType)
      form.set('file', evidenceFile)
      await apiFetch<Evidence>('/api/v1/financial/evidence', { method: 'POST', body: form })
      setEvidenceFile(null)
      setMessage('Evidence received and stored securely for review.')
      await onComplete()
    } catch (cause) {
      setMessage(cause instanceof Error ? cause.message : 'Evidence upload failed.')
    } finally {
      setUploading(false)
    }
  }

  return (
    <section className="financial-workspace">
      <form className="financial-form-panel" onSubmit={submit}>
        <span className="eyebrow">Financial eligibility</span><h2>Help us apply the right safeguards</h2><p>These answers support jurisdiction, affordability, and responsible gaming review. They never affect Practice access.</p>
        <div className="financial-form">
          <label htmlFor="assessment-country">Country</label><select id="assessment-country" value={country} onChange={(event) => setCountry(event.target.value)}><option value="ZA">South Africa</option></select>
          <label htmlFor="assessment-occupation">Occupation</label><select id="assessment-occupation" value={occupation} onChange={(event) => setOccupation(event.target.value)} required><option value="">Select occupation</option><option value="employed">Employed</option><option value="self_employed">Self-employed</option><option value="student">Student</option><option value="retired">Retired</option><option value="unemployed">Unemployed</option><option value="other">Other</option></select>
          <label htmlFor="assessment-source">Primary source of funds</label><select id="assessment-source" value={source} onChange={(event) => setSource(event.target.value)} required><option value="">Select source</option><option value="salary">Salary</option><option value="business">Business income</option><option value="savings">Savings</option><option value="investments">Investments</option><option value="pension">Pension</option><option value="other">Other</option></select>
          <button className="button primary"><ShieldCheck /> Submit for review</button>
        </div>
      </form>
      <div className="financial-panel assessment-status">
        <BadgeCheck />
        <span className="eyebrow">Current decision</span><h2>{label(assessment.status)}</h2>
        <p>{assessment.status === 'complete' ? 'Your financial assessment is complete. Available payment methods are controlled by country policy.' : 'A compliance decision must complete before live funds can move.'}</p>
        <dl className="financial-status-list">
          <div><dt>Verification</dt><dd className={statusTone(assessment.verificationStatus)}>{label(assessment.verificationStatus)}</dd></div>
          <div><dt>Risk</dt><dd>{label(assessment.riskClassification)}</dd></div>
          <div><dt>Responsible gaming</dt><dd className={statusTone(assessment.responsibleGamingStatus)}>{label(assessment.responsibleGamingStatus)}</dd></div>
        </dl>
        <form className="evidence-upload" onSubmit={uploadEvidence}>
          <span className="eyebrow">Supporting evidence</span>
          <label htmlFor="evidence-type">Document type</label>
          <select id="evidence-type" value={evidenceType} onChange={(event) => setEvidenceType(event.target.value)}>
            <option value="identity">Identity</option>
            <option value="address">Address</option>
            <option value="source_of_funds">Source of funds</option>
            <option value="payout_destination">Payout destination</option>
          </select>
          <label htmlFor="evidence-file">PDF, JPEG, or PNG</label>
          <input id="evidence-file" type="file" accept="application/pdf,image/jpeg,image/png" onChange={(event) => setEvidenceFile(event.target.files?.[0] ?? null)} />
          <button className="button secondary" disabled={uploading}>{uploading ? <><RefreshCw className="spin" /> Uploading</> : <><Upload /> Upload securely</>}</button>
        </form>
        <div className="evidence-list" aria-label="Submitted evidence">
          {evidence.length === 0 ? <p>No evidence has been submitted.</p> : evidence.map((item) => (
            <div key={item.id}><FileCheck2 /><span><strong>{label(item.type)}</strong><small>{label(item.status)} / {Math.ceil(item.sizeBytes / 1024)} KB</small></span></div>
          ))}
        </div>
      </div>
    </section>
  )
}
