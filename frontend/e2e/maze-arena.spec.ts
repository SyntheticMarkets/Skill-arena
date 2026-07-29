import { expect, test } from '@playwright/test'
import { promises as fs } from 'node:fs'
import path from 'node:path'

const outbox = path.resolve(__dirname, '../../backend/.e2e-data/email_outbox')
const proof = path.resolve(__dirname, '../../docs/proof/sprint-6-phase-8-maze')

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

async function authenticatedPlayer(page: import('@playwright/test').Page, projectName: string, suffix: string) {
  const email = `maze-${suffix}-${projectName}@example.com`
  const password = 'MazeArena!42'
  await page.goto('/auth/register')
  await page.getByLabel(/email address/i).fill(email)
  await page.getByLabel(/date of birth/i).fill('1990-01-01')
  await page.getByLabel(/^password$/i).fill(password)
  await page.getByText(/I am at least 18/i).click()
  await page.getByText(/I accept the Fair Play/i).click()
  await page.getByRole('button', { name: /create identity/i }).click()
  await page.goto(await verificationLink(email))
  await expect(page.getByRole('heading', { name: /identity confirmed/i })).toBeVisible()
  await page.goto('/auth/login?next=%2Fgames%2Fmaze')
  await page.getByLabel(/email address/i).fill(email)
  await page.getByLabel(/^password$/i).fill(password)
  await page.getByRole('button', { name: /enter securely/i }).click()
  await expect(page).toHaveURL(/\/games\/maze$/)
}

test('Maze Arena renders authoritative state and sends accessible intent', async ({ page }, testInfo) => {
  await authenticatedPlayer(page, testInfo.project.name, 'render')
  await expect(page.getByRole('heading', { name: /read the board/i })).toBeVisible()
  await page.getByRole('switch', { name: /reduced motion/i }).click()
  const matchStart = performance.now()
  await page.getByRole('button', { name: /enter practice/i }).click()

  const board = page.getByRole('img', { name: /maze board/i })
  await expect(board).toBeVisible({ timeout: 30_000 })
  const boardReadyMs = performance.now() - matchStart
  await expect(page.getByText(/your move/i)).toBeVisible()
  await expect(page.getByText(/connection/i).locator('..')).toContainText('connected')
  await page.getByText('Keyboard arrow controls').click()

  const firstArrow = page.getByRole('button', { name: /arrow 1, points/i })
  await firstArrow.focus()
  await expect(firstArrow).toBeFocused()
  const actionStart = performance.now()
  await firstArrow.press('Enter')
  await expect(page.locator('.maze-stage-footer')).toContainText('Client 1')
  const actionReceiptMs = performance.now() - actionStart
  await expect(page.locator('.sr-status')).toContainText(/arrow (blocked|escaped)/i)

  await page.getByRole('button', { name: 'Zoom in' }).click()
  await expect(page.getByLabel('Board zoom')).not.toHaveText('100%')
  await page.getByRole('button', { name: 'Fit board' }).click()
  await expect(page.getByLabel('Board zoom')).toHaveText('100%')

  const overflow = await page.evaluate(() => document.documentElement.scrollWidth - window.innerWidth)
  expect(overflow).toBeLessThanOrEqual(1)
  await fs.mkdir(proof, { recursive: true })
  await fs.writeFile(
    path.join(proof, `metrics-${testInfo.project.name}.json`),
    JSON.stringify({ boardReadyMs, actionReceiptMs, horizontalOverflowPx: overflow }, null, 2),
  )
  await page.screenshot({
    path: path.join(proof, `authoritative-${testInfo.project.name}.png`),
    fullPage: true,
  })
})

test('completed Practice exposes a verified server-event replay', async ({ page }, testInfo) => {
  test.skip(testInfo.project.name !== 'desktop-chromium')
  test.setTimeout(180_000)
  await authenticatedPlayer(page, testInfo.project.name, 'replay')
  await page.getByRole('switch', { name: /reduced motion/i }).click()
  await page.getByRole('button', { name: /enter practice/i }).click()
  await expect(page.getByRole('img', { name: /maze board/i })).toBeVisible({ timeout: 30_000 })
  await page.getByText('Keyboard arrow controls').click()

  let cursor = 0
  let submitted = 0
  let previousCount = await page.locator('.maze-keyboard-controls button').count()
  for (let attempt = 0; attempt < 400; attempt += 1) {
    if (await page.getByRole('heading', { name: /dependency chain/i }).isVisible().catch(() => false)) break
    const controls = page.locator('.maze-keyboard-controls button:not([disabled])')
    const count = await controls.count()
    if (count === 0) {
      await page.waitForTimeout(100)
      continue
    }
    if (count < previousCount) cursor = 0
    previousCount = count
    await controls.nth(cursor % count).click()
    submitted += 1
    await expect(page.locator('.maze-stage-footer')).toContainText(`Client ${submitted}`)
    cursor += 1
  }

  await expect(page.getByRole('heading', { name: /dependency chain/i })).toBeVisible()
  const watch = page.getByRole('button', { name: /watch verified replay/i })
  if (!await watch.isVisible().catch(() => false)) {
    await page.getByRole('button', { name: /check replay/i }).click()
  }
  await expect(watch).toBeVisible({ timeout: 20_000 })
  await watch.click()
  await expect(page.getByRole('heading', { name: /authoritative match reconstruction/i })).toBeVisible()
  await expect(page.getByText(/playback follows committed server events/i)).toBeVisible()
  await page.getByRole('button', { name: /play replay/i }).click()
  await expect(page.getByText(/1 \//)).toBeVisible({ timeout: 10_000 })
  await fs.mkdir(proof, { recursive: true })
  await page.screenshot({
    path: path.join(proof, 'verified-replay-desktop-chromium.png'),
    fullPage: true,
  })
})
