'use client'

import { useEffect, useState } from 'react'
import { useRouter } from 'next/navigation'

const apiBase = process.env.NEXT_PUBLIC_API_BASE_URL ?? 'http://localhost:8080'

type AdminUser = {
  id: string
  email: string
  role: string
  emailVerified: boolean
  kycStatus: string
  createdAt: string
}

type AuditLog = {
  id: string
  actorId?: string
  action: string
  targetId?: string
  metadata?: Record<string, string>
  ipAddress?: string
  createdAt: string
}

type TreasuryHealth = {
  playerLiabilities: number
  totalReserves: number
  coverageRatio: number
  isSolvent: boolean
  houseExposure: number
  state: {
    playerReserve: number
    revenueReserve: number
    seasonReserve: number
    championshipReserve: number
    jackpotReserve: number
    emergencyReserve: number
  }
}

type HouseRisk = {
  tierId: string
  attempts: number
  wins: number
  losses: number
  playerWinRate: number
  targetHouseEdge: number
  recommendedAction: string
}

type BehavioralBaseline = {
  userId: string
  calibrationRuns: number
  averageEfficiency: number
  averageMoveSeconds: number
  bestMoveCount?: number
  lastSessionId?: string
  lastRunAt: string
  riskSignal: string
}

type TournamentDetail = {
  tournament: {
    id: string
    name: string
    status: string
    entryFee: number
    walletType: string
    prizePool: number
    startsAt: string
  }
  participants: Array<{ id: string; userId: string; email?: string; seed: number; status: string }>
  matches: Array<{ id: string; round: number; matchNumber: number; playerAId?: string; playerBId?: string; winnerId?: string; status: string }>
}

type BackgroundJob = {
  id: string
  type: string
  status: string
  attempts: number
  maxAttempts: number
  worker?: string
  lastError?: string
  resultArtifact?: string
  runAfter: string
  updatedAt: string
}

type JobQueueStats = {
  pendingJobs: number
  runningJobs: number
  completedJobs: number
  failedJobs: number
  cancelledJobs: number
  retryCount: number
  averageProcessingSeconds: number
  workerStatus?: Record<string, string>
}

type SystemHealth = {
  apiStatus: string
  databaseStatus: string
  queueStatus: string
  backupStatus: string
  maintenanceEnabled: boolean
  maintenanceMessage?: string
  workerHealth?: Record<string, string>
  activeMatches: number
  playersOnline: number
}

type BackupRecord = {
  id: string
  type: string
  status: string
  path: string
  verified: boolean
  sizeBytes: number
  startedAt: string
  finishedAt: string
}

function authHeaders() {
  const token = window.localStorage.getItem('skill-arena-token')
  return token ? { Authorization: `Bearer ${token}` } : null
}

const normalizeTournaments = (items: TournamentDetail[] | null): TournamentDetail[] =>
  (items ?? []).map((detail) => ({
    ...detail,
    participants: detail.participants ?? [],
    matches: detail.matches ?? [],
  }))

export default function AdminPage() {
  const router = useRouter()
  const [users, setUsers] = useState<AdminUser[]>([])
  const [auditLogs, setAuditLogs] = useState<AuditLog[]>([])
  const [treasuryHealth, setTreasuryHealth] = useState<TreasuryHealth | null>(null)
  const [houseRisk, setHouseRisk] = useState<HouseRisk | null>(null)
  const [baselines, setBaselines] = useState<BehavioralBaseline[]>([])
  const [tournaments, setTournaments] = useState<TournamentDetail[]>([])
  const [jobs, setJobs] = useState<BackgroundJob[]>([])
  const [jobStats, setJobStats] = useState<JobQueueStats | null>(null)
  const [systemHealth, setSystemHealth] = useState<SystemHealth | null>(null)
  const [backups, setBackups] = useState<BackupRecord[]>([])
  const [selectedTournamentId, setSelectedTournamentId] = useState('daily-maze-open')
  const [selectedMatchId, setSelectedMatchId] = useState('')
  const [winnerId, setWinnerId] = useState('')
  const [message, setMessage] = useState('')
  const [error, setError] = useState('')

  useEffect(() => {
    const headers = authHeaders()
    if (!headers) {
      router.replace('/auth/login')
      return
    }
    loadAdmin().catch(() => setError('Unable to load admin data. Confirm this account has admin access.'))
  }, [router])

  async function loadAdmin() {
    const headers = authHeaders()
    if (!headers) return

    const [usersRes, auditRes, treasuryRes, riskRes, baselinesRes, tournamentsRes, jobsRes, statsRes, healthRes, backupsRes] = await Promise.all([
      fetch(`${apiBase}/api/v1/admin/users`, { headers }),
      fetch(`${apiBase}/api/v1/admin/audit-logs`, { headers }),
      fetch(`${apiBase}/api/v1/admin/treasury/health`, { headers }),
      fetch(`${apiBase}/api/v1/admin/house-risk/bronze`, { headers }),
      fetch(`${apiBase}/api/v1/admin/baselines`, { headers }),
      fetch(`${apiBase}/api/v1/tournaments`, { headers }),
      fetch(`${apiBase}/api/v1/admin/jobs`, { headers }),
      fetch(`${apiBase}/api/v1/admin/jobs/stats`, { headers }),
      fetch(`${apiBase}/api/v1/admin/system-health`, { headers }),
      fetch(`${apiBase}/api/v1/admin/backups`, { headers }),
    ])

    if ([usersRes, auditRes, treasuryRes, riskRes, baselinesRes, tournamentsRes, jobsRes, statsRes, healthRes, backupsRes].some((response) => !response.ok)) {
      setError('Admin access denied or backend unavailable.')
      return
    }

    setUsers(await usersRes.json())
    setAuditLogs(await auditRes.json())
    setTreasuryHealth(await treasuryRes.json())
    setHouseRisk(await riskRes.json())
    setBaselines(await baselinesRes.json())
    setJobs(await jobsRes.json())
    setJobStats(await statsRes.json())
    setSystemHealth(await healthRes.json())
    setBackups(await backupsRes.json())
    const tournamentBody = normalizeTournaments(await tournamentsRes.json())
    setTournaments(tournamentBody)
    if (tournamentBody.length > 0 && !selectedTournamentId) {
      setSelectedTournamentId(tournamentBody[0].tournament.id)
    }
  }

  async function runJobAction(action: 'retry' | 'cancel' | 'requeue', jobId: string) {
    const headers = authHeaders()
    if (!headers) return
    setMessage('')
    setError('')

    const response = await fetch(`${apiBase}/api/v1/admin/jobs/${action}`, {
      method: 'POST',
      headers: { ...headers, 'Content-Type': 'application/json' },
      body: JSON.stringify({ jobId }),
    })
    if (!response.ok) {
      setError(await response.text())
      return
    }
    setMessage(`Job ${action} requested.`)
    loadAdmin()
  }

  async function requestBackup() {
    const headers = authHeaders()
    if (!headers) return
    setMessage('')
    setError('')

    const response = await fetch(`${apiBase}/api/v1/admin/backups`, {
      method: 'POST',
      headers: { ...headers, 'Content-Type': 'application/json' },
      body: JSON.stringify({}),
    })
    if (!response.ok) {
      setError(await response.text())
      return
    }
    setMessage('Manual backup queued.')
    loadAdmin()
  }

  async function approveKYC(userId: string) {
    const headers = authHeaders()
    if (!headers) return
    setMessage('')
    setError('')

    const response = await fetch(`${apiBase}/api/v1/admin/kyc/approve`, {
      method: 'POST',
      headers: { ...headers, 'Content-Type': 'application/json' },
      body: JSON.stringify({ userId }),
    })
    if (!response.ok) {
      setError(await response.text())
      return
    }
    setMessage('KYC approved.')
    loadAdmin()
  }

  async function generateBracket(tournamentId: string) {
    const headers = authHeaders()
    if (!headers) return
    setMessage('')
    setError('')

    const response = await fetch(`${apiBase}/api/v1/admin/tournaments/bracket`, {
      method: 'POST',
      headers: { ...headers, 'Content-Type': 'application/json' },
      body: JSON.stringify({ tournamentId }),
    })
    if (!response.ok) {
      setError(await response.text())
      return
    }
    setMessage('Bracket generated.')
    loadAdmin()
  }

  async function reportResult(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const headers = authHeaders()
    if (!headers) return
    setMessage('')
    setError('')

    const response = await fetch(`${apiBase}/api/v1/admin/tournaments/result`, {
      method: 'POST',
      headers: { ...headers, 'Content-Type': 'application/json' },
      body: JSON.stringify({ tournamentId: selectedTournamentId, matchId: selectedMatchId, winnerId }),
    })
    if (!response.ok) {
      setError(await response.text())
      return
    }
    setMessage('Tournament result recorded.')
    setSelectedMatchId('')
    setWinnerId('')
    loadAdmin()
  }

  return (
    <main className="container dashboard-shell">
      <div className="dashboard-header">
        <div>
          <h1>Admin Operations</h1>
          <p>Manage users, treasury health, tournaments, risk, and audit activity.</p>
        </div>
        <button type="button" className="button secondary" onClick={() => loadAdmin()}>
          Refresh
        </button>
      </div>

      {error ? <p className="form-error">{error}</p> : null}
      {message ? <p className="form-success">{message}</p> : null}

      <div className="dashboard-grid">
        <section className="card">
          <h2>Treasury health</h2>
          {treasuryHealth ? (
            <div className="stat-grid">
              <div><span>Coverage</span><strong>{treasuryHealth.coverageRatio.toFixed(2)}x</strong></div>
              <div><span>Solvent</span><strong>{treasuryHealth.isSolvent ? 'Yes' : 'No'}</strong></div>
              <div><span>Liabilities</span><strong>{treasuryHealth.playerLiabilities.toFixed(2)}</strong></div>
              <div><span>Exposure</span><strong>{treasuryHealth.houseExposure.toFixed(2)}</strong></div>
            </div>
          ) : <p>Loading treasury...</p>}
        </section>

        <section className="card">
          <h2>House risk</h2>
          {houseRisk ? (
            <ul className="profile-list">
              <li><strong>Tier:</strong> {houseRisk.tierId}</li>
              <li><strong>Attempts:</strong> {houseRisk.attempts}</li>
              <li><strong>Player win rate:</strong> {Math.round(houseRisk.playerWinRate * 100)}%</li>
              <li><strong>Action:</strong> {houseRisk.recommendedAction}</li>
            </ul>
          ) : <p>Loading risk...</p>}
        </section>
      </div>

      <div className="dashboard-grid">
        <section className="card">
          <h2>System health</h2>
          {systemHealth ? (
            <ul className="profile-list">
              <li><strong>API:</strong> {systemHealth.apiStatus}</li>
              <li><strong>Database:</strong> {systemHealth.databaseStatus}</li>
              <li><strong>Queue:</strong> {systemHealth.queueStatus}</li>
              <li><strong>Backup:</strong> {systemHealth.backupStatus}</li>
              <li><strong>Maintenance:</strong> {systemHealth.maintenanceEnabled ? systemHealth.maintenanceMessage || 'Enabled' : 'Disabled'}</li>
              <li><strong>Active matches:</strong> {systemHealth.activeMatches}</li>
              <li><strong>Players online:</strong> {systemHealth.playersOnline}</li>
            </ul>
          ) : <p>Loading system health...</p>}
        </section>

        <section className="card">
          <h2>Worker status</h2>
          {systemHealth?.workerHealth && Object.keys(systemHealth.workerHealth).length > 0 ? (
            <ul className="profile-list">
              {Object.entries(systemHealth.workerHealth).map(([name, status]) => (
                <li key={name}><strong>{name.replace(/_/g, ' ')}:</strong> {status}</li>
              ))}
            </ul>
          ) : <p>No worker heartbeat yet.</p>}
        </section>
      </div>

      <section className="card leaderboard-card">
        <div className="card-header-row">
          <h2>Background jobs</h2>
          <button type="button" className="button secondary" onClick={requestBackup}>Manual backup</button>
        </div>
        {jobStats ? (
          <div className="stat-grid">
            <div><span>Pending</span><strong>{jobStats.pendingJobs}</strong></div>
            <div><span>Running</span><strong>{jobStats.runningJobs}</strong></div>
            <div><span>Completed</span><strong>{jobStats.completedJobs}</strong></div>
            <div><span>Failed</span><strong>{jobStats.failedJobs}</strong></div>
            <div><span>Retries</span><strong>{jobStats.retryCount}</strong></div>
            <div><span>Avg time</span><strong>{jobStats.averageProcessingSeconds.toFixed(2)}s</strong></div>
          </div>
        ) : <p>Loading queue statistics...</p>}
        <table className="leaderboard-table">
          <thead>
            <tr><th>Type</th><th>Status</th><th>Worker</th><th>Retries</th><th>Updated</th><th>Action</th></tr>
          </thead>
          <tbody>
            {jobs.slice(0, 12).map((job) => (
              <tr key={job.id}>
                <td>{job.type}</td>
                <td>{job.status}</td>
                <td>{job.worker || '-'}</td>
                <td>{Math.max(job.attempts - 1, 0)} / {job.maxAttempts}</td>
                <td>{new Date(job.updatedAt).toLocaleString()}</td>
                <td>
                  <button type="button" className="button table-button" onClick={() => runJobAction('retry', job.id)}>Retry</button>
                  <button type="button" className="button table-button secondary" onClick={() => runJobAction('cancel', job.id)}>Cancel</button>
                  <button type="button" className="button table-button secondary" onClick={() => runJobAction('requeue', job.id)}>Requeue</button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>

      <section className="card leaderboard-card">
        <h2>Backup history</h2>
        <table className="leaderboard-table">
          <thead>
            <tr><th>Started</th><th>Status</th><th>Verified</th><th>Size</th><th>Path</th></tr>
          </thead>
          <tbody>
            {backups.slice(0, 8).map((backup) => (
              <tr key={backup.id}>
                <td>{new Date(backup.startedAt).toLocaleString()}</td>
                <td>{backup.status}</td>
                <td>{backup.verified ? 'Yes' : 'No'}</td>
                <td>{Math.round(backup.sizeBytes / 1024)} KB</td>
                <td>{backup.path}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>

      <section className="card leaderboard-card">
        <h2>User management</h2>
        <table className="leaderboard-table">
          <thead>
            <tr><th>Email</th><th>Role</th><th>KYC</th><th>Email</th><th>Action</th></tr>
          </thead>
          <tbody>
            {users.map((user) => (
              <tr key={user.id}>
                <td>{user.email}</td>
                <td>{user.role}</td>
                <td>{user.kycStatus}</td>
                <td>{user.emailVerified ? 'Verified' : 'Pending'}</td>
                <td>
                  {user.kycStatus !== 'approved' ? (
                    <button type="button" className="button table-button" onClick={() => approveKYC(user.id)}>
                      Approve KYC
                    </button>
                  ) : 'Approved'}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>

      <section className="card leaderboard-card">
        <h2>Tournament operations</h2>
        <div className="tournament-grid">
          {tournaments.map((detail) => (
            <article key={detail.tournament.id} className="tournament-item">
              <div>
                <strong>{detail.tournament.name}</strong>
                <span>{detail.tournament.status} | {detail.participants.length} players | {detail.matches.length} matches</span>
              </div>
              <button type="button" className="button" onClick={() => generateBracket(detail.tournament.id)}>
                Generate bracket
              </button>
            </article>
          ))}
        </div>

        <form className="form-grid admin-result-form" onSubmit={reportResult}>
          <label>
            Tournament
            <select value={selectedTournamentId} onChange={(event) => setSelectedTournamentId(event.target.value)}>
              {tournaments.map((detail) => (
                <option key={detail.tournament.id} value={detail.tournament.id}>{detail.tournament.name}</option>
              ))}
            </select>
          </label>
          <label>
            Match ID
            <input value={selectedMatchId} onChange={(event) => setSelectedMatchId(event.target.value)} placeholder="Match UUID" />
          </label>
          <label>
            Winner user ID
            <input value={winnerId} onChange={(event) => setWinnerId(event.target.value)} placeholder="Winner UUID" />
          </label>
          <button type="submit" className="button">Report result</button>
        </form>
      </section>

      <section className="card leaderboard-card">
        <h2>Behavior baselines</h2>
        {baselines.length === 0 ? (
          <p>No calibration baselines yet.</p>
        ) : (
          <table className="leaderboard-table">
            <thead>
              <tr><th>User</th><th>Runs</th><th>Efficiency</th><th>Move time</th><th>Signal</th></tr>
            </thead>
            <tbody>
              {baselines.map((baseline) => (
                <tr key={baseline.userId}>
                  <td>{baseline.userId}</td>
                  <td>{baseline.calibrationRuns}</td>
                  <td>{Math.round(baseline.averageEfficiency * 100)}%</td>
                  <td>{baseline.averageMoveSeconds.toFixed(2)}s</td>
                  <td>{baseline.riskSignal}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </section>

      <section className="card leaderboard-card">
        <h2>Audit log</h2>
        <table className="leaderboard-table">
          <thead>
            <tr><th>Time</th><th>Action</th><th>Actor</th><th>Target</th></tr>
          </thead>
          <tbody>
            {auditLogs.map((log) => (
              <tr key={log.id}>
                <td>{new Date(log.createdAt).toLocaleString()}</td>
                <td>{log.action}</td>
                <td>{log.actorId || '-'}</td>
                <td>{log.targetId || '-'}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>
    </main>
  )
}
