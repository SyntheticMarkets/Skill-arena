import { expect, test } from '@playwright/test'
import { promises as fs } from 'node:fs'
import path from 'node:path'

const outbox = path.resolve(__dirname, '../../backend/.e2e-data/email_outbox')
const proof = path.resolve(__dirname, '../../docs/proof/sprint-5-realtime-arena')

async function verificationLink(recipient: string) {
  const deadline = Date.now() + 20_000
  while (Date.now() < deadline) {
    const entries = await fs.readdir(outbox).catch(() => [] as string[])
    for (const name of entries.reverse()) {
      const content = await fs.readFile(path.join(outbox, name), 'utf8')
      if (!content.includes(`To: ${recipient}`)) continue
      const match = content.match(/https?:\/\/[^\s<"]+/)
      if (match) return match[0].replace(/&amp;/g, '&')
    }
    await new Promise((resolve) => setTimeout(resolve, 250))
  }
  throw new Error(`No verification email arrived for ${recipient}`)
}

test('authenticated realtime lifecycle recovers on a new connection', async ({ page }, testInfo) => {
  const email = `realtime-${testInfo.project.name}@example.com`
  const password = 'RealtimeArena!42'
  await page.goto('/auth/register')
  await page.getByLabel(/email address/i).fill(email)
  await page.getByLabel(/date of birth/i).fill('1990-01-01')
  await page.getByLabel(/^password$/i).fill(password)
  await page.getByText(/I am at least 18/i).click()
  await page.getByText(/I accept the Fair Play/i).click()
  await page.getByRole('button', { name: /create identity/i }).click()
  await page.goto(await verificationLink(email))
  await expect(page.getByRole('heading', { name: /identity confirmed/i })).toBeVisible()
  await page.goto('/auth/login?next=%2Farena')
  await page.getByLabel(/email address/i).fill(email)
  await page.getByLabel(/^password$/i).fill(password)
  await page.getByRole('button', { name: /enter securely/i }).click()
  await expect(page).toHaveURL(/\/arena$/)

  const evidence = await page.evaluate(async () => {
    const api = 'http://127.0.0.1:18080'
    const queueResponse = await fetch(`${api}/api/v1/realtime/queue`, {
      method: 'POST',
      credentials: 'include',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ gameId: 'maze_arena', mode: 'practice', walletCategory: 'practice', region: 'af-south', jurisdiction: 'ZA', latencyMs: 15 }),
    })
    if (!queueResponse.ok) throw new Error(`queue failed: ${queueResponse.status} ${await queueResponse.text()}`)
    const queued = await queueResponse.json()
    const matchId = queued.match.id as string

    const session = (reconnect: boolean) => new Promise<{ types: string[]; sequence: number }>((resolve, reject) => {
      const socket = new WebSocket('ws://127.0.0.1:18080/api/v1/realtime/gateway')
      const types: string[] = []
      let sequence = 0
      const timeout = setTimeout(() => reject(new Error(`gateway timeout: ${types.join(',')}`)), 10_000)
      socket.onmessage = (event) => {
        const message = JSON.parse(String(event.data))
        types.push(message.type)
        if (message.event?.sequence) sequence = message.event.sequence
        if (message.type === 'session.negotiated') {
          socket.send(JSON.stringify({ type: reconnect ? 'reconnect' : 'ready', matchId, afterSequence: 0 }))
        }
        if ((!reconnect && message.type === 'match.status' && message.match?.status === 'live') ||
            (reconnect && message.type === 'state.sync' && message.match?.status === 'live')) {
          clearTimeout(timeout)
          socket.close()
          resolve({ types, sequence })
        }
      }
      socket.onerror = () => reject(new Error('gateway connection failed'))
    })
    const started = await session(false)
    await new Promise((resolve) => setTimeout(resolve, 250))
    const recovered = await session(true)
    const leaveResponse = await fetch(`${api}/api/v1/realtime/matches/${matchId}/leave`, {
      method: 'POST', credentials: 'include', headers: { 'Content-Type': 'application/json' }, body: '{}',
    })
    const terminal = await leaveResponse.json()
    return { matchId, queueStatus: queued.queue.status, started, recovered, terminalStatus: terminal.status }
  })

  expect(evidence.queueStatus).toBe('matched')
  expect(evidence.started.types).toContain('match.status')
  expect(evidence.recovered.types).toContain('state.sync')
  expect(evidence.terminalStatus).toBe('abandoned')
  await fs.mkdir(proof, { recursive: true })
  await page.screenshot({ path: path.join(proof, `reconnect-${testInfo.project.name}.png`), fullPage: true })
})
