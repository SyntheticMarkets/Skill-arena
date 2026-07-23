import { expect, test } from '@playwright/test'
import { promises as fs } from 'node:fs'
import path from 'node:path'

const outbox = path.resolve(__dirname, '../../backend/.e2e-data/email_outbox')
const proof = path.resolve(__dirname, '../../docs/proof/sprint-2-arena-hub')

async function capture(page: import('@playwright/test').Page, step: string, project: string) {
  await fs.mkdir(proof, { recursive: true })
  await page.evaluate(() => window.scrollTo(0, 0))
  await page.screenshot({ path: path.join(proof, `${step}-${project}.png`), fullPage: true })
}

async function verificationLink(recipient: string) {
  const deadline = Date.now() + 20_000
  while (Date.now() < deadline) {
    const entries = await fs.readdir(outbox).catch(() => [] as string[])
    const messages = await Promise.all(entries.map(async (name) => ({
      content: await fs.readFile(path.join(outbox, name), 'utf8'),
      stat: await fs.stat(path.join(outbox, name)),
    })))
    messages.sort((a, b) => b.stat.mtimeMs - a.stat.mtimeMs)
    const message = messages.find((item) => item.content.includes(`To: ${recipient}`) && item.content.includes('/auth/verify-email'))
    const match = message?.content.match(/https?:\/\/[^\s<"]+/)
    if (match) return match[0].replace(/&amp;/g, '&')
    await new Promise((resolve) => setTimeout(resolve, 250))
  }
  throw new Error(`No verification email arrived for ${recipient}`)
}

test('Arena Hub renders real player state and durable player services', async ({ page }, testInfo) => {
  const project = testInfo.project.name
  const email = `hub-${project}@example.com`
  const password = 'ArenaHubPassword!42'

  await page.goto('/auth/register')
  await page.getByLabel(/email address/i).fill(email)
  await page.getByLabel(/date of birth/i).fill('1990-01-01')
  await page.getByLabel(/^password$/i).fill(password)
  await page.getByText(/I am at least 18/i).click()
  await page.getByText(/I accept the Fair Play/i).click()
  await page.getByRole('button', { name: /create identity/i }).click()
  await page.goto(await verificationLink(email))
  await expect(page.getByRole('heading', { name: /identity confirmed/i })).toBeVisible()

  await page.goto('/auth/login?next=%2Fdashboard')
  await page.getByLabel(/email address/i).fill(email)
  await page.getByLabel(/^password$/i).fill(password)
  await page.getByRole('button', { name: /enter securely/i }).click()
  await expect(page).toHaveURL(/\/dashboard$/)
  await expect(page.getByRole('heading', { name: /good (morning|afternoon|evening)/i })).toBeVisible()
  await expect(page.getByText('Maze Arena')).toBeVisible()
  await expect(page.getByLabel('Competition calendar').getByText(/no tournament is currently accepting players/i)).toBeVisible()
  await capture(page, 'dashboard', project)

  const mobileMenu = page.getByRole('button', { name: /open navigation/i })
  if (await mobileMenu.isVisible()) await mobileMenu.click()
  await page.getByRole('link', { name: /^profile$/i }).first().click()
  await page.getByRole('button', { name: /edit profile/i }).click()
  await page.getByLabel(/username/i).fill(`hub_${project.replace(/-/g, '_')}`)
  await page.getByLabel(/display name/i).fill('Hub Competitor')
  await page.getByLabel(/country code/i).fill('ZA')
  await page.getByRole('button', { name: /save profile/i }).click()
  await expect(page.getByText(/competitor profile was updated/i)).toBeVisible()
  await capture(page, 'profile', project)

  await page.goto('/notifications')
  await expect(page.getByRole('heading', { name: /know what changed/i })).toBeVisible()
  await expect(page.getByText(/no notifications in this view/i)).toBeVisible()
  await capture(page, 'notifications', project)

  await page.goto('/support')
  await page.getByLabel(/subject/i).fill('Arena Hub verification')
  await page.getByLabel(/support team understand/i).fill('Confirm that this support request is stored against my account.')
  await page.getByRole('button', { name: /submit ticket/i }).click()
  await expect(page.getByText(/ticket was received/i)).toBeVisible()
  await expect(page.getByText('Arena Hub verification')).toBeVisible()
  await capture(page, 'support', project)
})
