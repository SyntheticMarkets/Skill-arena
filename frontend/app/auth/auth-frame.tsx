import Link from 'next/link'
import { ArrowLeft, ShieldCheck } from 'lucide-react'

export function AuthFrame({ eyebrow, title, description, children }: {
  eyebrow: string
  title: string
  description: string
  children: React.ReactNode
}) {
  return (
    <main className="auth-experience">
      <header className="auth-header">
        <Link href="/" className="constructed-logo" aria-label="Skill Arena home">
          <span className="logo-mark" aria-hidden="true"><i /><i /><i /></span>
          <span>Skill Arena</span>
        </Link>
        <Link href="/arena" className="text-link"><ArrowLeft aria-hidden="true" /> Guest Arena</Link>
      </header>
      <section className="auth-stage">
        <aside className="auth-promise" aria-label="Skill Arena security promise">
          <ShieldCheck aria-hidden="true" />
          <p>Protected entry</p>
          <h2>Your identity is part of the competition.</h2>
          <span>Email verification, session controls, and MFA protect every future result attached to your name.</span>
          <div className="auth-signal"><i /><span>Encrypted session</span><strong>ACTIVE</strong></div>
        </aside>
        <div className="auth-panel">
          <div className="auth-copy">
            <span className="section-label">{eyebrow}</span>
            <h1>{title}</h1>
            <p>{description}</p>
          </div>
          {children}
        </div>
      </section>
    </main>
  )
}

export function FormMessage({ type, children }: { type: 'error' | 'success' | 'info'; children: React.ReactNode }) {
  return <div className={`form-message ${type}`} role={type === 'error' ? 'alert' : 'status'}>{children}</div>
}
