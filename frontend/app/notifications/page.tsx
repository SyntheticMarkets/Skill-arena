'use client'

import Link from 'next/link'
import { Archive, Bell, Check, ChevronRight, Inbox } from 'lucide-react'
import { useEffect, useState } from 'react'
import { useHub } from '../hub-context'
import { apiFetch, postJSON } from '../lib/api'

type Notification = {
  id: string
  category: string
  title: string
  message: string
  status: 'unread' | 'read' | 'archived'
  actionUrl?: string
  createdAt: string
}

type NotificationResponse = { notifications: Notification[] }
type Filter = 'all' | Notification['status']

export default function NotificationsPage() {
  const { reload: reloadHub } = useHub()
  const [filter, setFilter] = useState<Filter>('all')
  const [notifications, setNotifications] = useState<Notification[]>([])
  const [status, setStatus] = useState<'loading' | 'ready' | 'error'>('loading')
  const [error, setError] = useState('')

  async function load(selected: Filter) {
    setStatus('loading')
    setError('')
    try {
      const query = selected === 'all' ? '' : `?status=${selected}`
      const response = await apiFetch<NotificationResponse>(`/api/v1/notifications${query}`)
      setNotifications(response.notifications)
      setStatus('ready')
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : 'Notifications could not be loaded.')
      setStatus('error')
    }
  }

  useEffect(() => {
    const timer = window.setTimeout(() => void load(filter), 0)
    return () => window.clearTimeout(timer)
  }, [filter])

  async function update(notificationId: string, action: 'read' | 'archive') {
    await postJSON(`/api/v1/notifications/${action}`, { notificationId })
    await Promise.all([load(filter), reloadHub()])
  }

  return (
    <main className="hub-page">
      <section className="subpage-heading">
        <div><span className="eyebrow">Notification centre</span><h1>Know what changed.</h1><p>Security, progression, competition, and wallet updates share one verified timeline.</p></div>
        <Bell aria-hidden="true" />
      </section>

      <div className="segmented-control notification-filters" aria-label="Filter notifications">
        {(['all', 'unread', 'read', 'archived'] as Filter[]).map((item) => (
          <button key={item} type="button" aria-pressed={filter === item} onClick={() => setFilter(item)}>{item}</button>
        ))}
      </div>

      {status === 'loading' ? <div className="inline-loading" role="status">Loading notification state...</div> : null}
      {status === 'error' ? <div className="form-message error" role="alert">{error}<button type="button" onClick={() => void load(filter)}>Retry</button></div> : null}
      {status === 'ready' && notifications.length === 0 ? (
        <section className="empty-state"><Inbox /><h2>No notifications in this view.</h2><p>New updates will appear only when the platform records a real event for your account.</p></section>
      ) : null}

      <section className="notification-list" aria-live="polite">
        {notifications.map((notification) => (
          <article key={notification.id} className={notification.status === 'unread' ? 'unread' : ''}>
            <span className="notification-dot" aria-label={notification.status} />
            <div>
              <span>{notification.category}</span>
              <h2>{notification.title}</h2>
              <p>{notification.message}</p>
              <time dateTime={notification.createdAt}>{new Date(notification.createdAt).toLocaleString()}</time>
            </div>
            <div className="notification-actions">
              {notification.status === 'unread' ? <button className="icon-button" type="button" title="Mark as read" aria-label={`Mark ${notification.title} as read`} onClick={() => void update(notification.id, 'read')}><Check /></button> : null}
              {notification.status !== 'archived' ? <button className="icon-button" type="button" title="Archive" aria-label={`Archive ${notification.title}`} onClick={() => void update(notification.id, 'archive')}><Archive /></button> : null}
              {notification.actionUrl ? <Link className="icon-button" href={notification.actionUrl} aria-label={`Open ${notification.title}`}><ChevronRight /></Link> : null}
            </div>
          </article>
        ))}
      </section>
    </main>
  )
}
