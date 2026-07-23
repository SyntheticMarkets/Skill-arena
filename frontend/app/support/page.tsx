'use client'

import { BookOpen, CircleHelp, Mail, Send } from 'lucide-react'
import { FormEvent, useEffect, useState } from 'react'
import { apiFetch, postJSON } from '../lib/api'

type SupportContent = {
  contactEmail: string
  articles: Array<{ id: string; category: string; title: string; body: string }>
}

type SupportTicket = {
  id: string
  category: string
  subject: string
  message: string
  status: string
  createdAt: string
  updatedAt: string
}

export default function SupportPage() {
  const [content, setContent] = useState<SupportContent | null>(null)
  const [tickets, setTickets] = useState<SupportTicket[]>([])
  const [category, setCategory] = useState('account')
  const [subject, setSubject] = useState('')
  const [message, setMessage] = useState('')
  const [status, setStatus] = useState<'loading' | 'ready' | 'error'>('loading')
  const [submitting, setSubmitting] = useState(false)
  const [notice, setNotice] = useState('')

  async function load() {
    setStatus('loading')
    try {
      const [supportContent, ticketResponse] = await Promise.all([
        apiFetch<SupportContent>('/api/v1/support/content'),
        apiFetch<{ tickets: SupportTicket[] }>('/api/v1/support/tickets'),
      ])
      setContent(supportContent)
      setTickets(ticketResponse.tickets)
      setStatus('ready')
    } catch {
      setStatus('error')
    }
  }

  useEffect(() => {
    const timer = window.setTimeout(() => void load(), 0)
    return () => window.clearTimeout(timer)
  }, [])

  async function submit(event: FormEvent) {
    event.preventDefault()
    setSubmitting(true)
    setNotice('')
    try {
      await postJSON<SupportTicket>('/api/v1/support/tickets', { category, subject, message })
      setSubject('')
      setMessage('')
      setNotice('Your ticket was received and is now visible in your support history.')
      await load()
    } catch (cause) {
      setNotice(cause instanceof Error ? cause.message : 'The ticket could not be submitted.')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <main className="hub-page">
      <section className="subpage-heading">
        <div><span className="eyebrow">Support centre</span><h1>Clear answers. Traceable help.</h1><p>Learn the rules, review responsible gaming guidance, or create a support ticket tied to your identity.</p></div>
        <CircleHelp aria-hidden="true" />
      </section>

      {status === 'loading' ? <div className="inline-loading" role="status">Loading support services...</div> : null}
      {status === 'error' ? <div className="form-message error" role="alert">Support services are unavailable.<button type="button" onClick={() => void load()}>Retry</button></div> : null}

      {content ? (
        <section className="support-layout">
          <div className="support-articles">
            <div className="hub-section-heading"><div><span className="eyebrow">Rules and guidance</span><h2>Understand every stage.</h2></div><BookOpen /></div>
            {content.articles.map((article) => <article key={article.id}><span>{article.category.replace('_', ' ')}</span><h3>{article.title}</h3><p>{article.body}</p></article>)}
            <a className="support-contact" href={`mailto:${content.contactEmail}`}><Mail /><span><strong>Contact support</strong><small>{content.contactEmail}</small></span></a>
          </div>

          <form className="support-form" onSubmit={submit}>
            <span className="eyebrow">Create ticket</span>
            <h2>Tell us what happened.</h2>
            <label htmlFor="support-category">Category</label>
            <select id="support-category" value={category} onChange={(event) => setCategory(event.target.value)}>
              <option value="account">Account</option>
              <option value="security">Security</option>
              <option value="gameplay">Gameplay</option>
              <option value="wallet">Wallet status</option>
              <option value="responsible_gaming">Responsible gaming</option>
            </select>
            <label htmlFor="support-subject">Subject</label>
            <input id="support-subject" value={subject} minLength={4} maxLength={120} onChange={(event) => setSubject(event.target.value)} required />
            <label htmlFor="support-message">What should the support team understand?</label>
            <textarea id="support-message" value={message} minLength={10} maxLength={4000} rows={7} onChange={(event) => setMessage(event.target.value)} required />
            {notice ? <p className="form-message" role="status">{notice}</p> : null}
            <button className="button" type="submit" disabled={submitting}>{submitting ? 'Submitting...' : 'Submit ticket'}<Send /></button>
          </form>
        </section>
      ) : null}

      <section className="hub-section ticket-history" aria-labelledby="ticket-history-title">
        <div className="hub-section-heading"><div><span className="eyebrow">Ticket history</span><h2 id="ticket-history-title">Your support trail.</h2></div></div>
        {tickets.length ? tickets.map((ticket) => (
          <article key={ticket.id}>
            <span><strong>{ticket.subject}</strong><small>{ticket.category.replace('_', ' ')} / {new Date(ticket.createdAt).toLocaleString()}</small></span>
            <span className={`ticket-status ${ticket.status}`}>{ticket.status}</span>
          </article>
        )) : <p className="hub-empty">You have not created a support ticket.</p>}
      </section>
    </main>
  )
}
