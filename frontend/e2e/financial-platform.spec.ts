import { expect, test } from '@playwright/test'
import { promises as fs } from 'node:fs'
import path from 'node:path'

const outbox = path.resolve(__dirname, '../../backend/.e2e-data/email_outbox')
const proof = path.resolve(__dirname, '../../docs/proof/sprint-3-financial-platform')

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

async function capture(page: import('@playwright/test').Page, step: string, project: string) {
  await fs.mkdir(proof, { recursive: true })
  await page.evaluate(() => window.scrollTo(0, 0))
  await page.screenshot({ path: path.join(proof, `${step}-${project}.png`), fullPage: true })
}

test('player financial journey is backend-driven and policy gated', async ({ page }, testInfo) => {
  const email = `financial-${testInfo.project.name}@example.com`
  const password = 'FinancialPlatform!42'

  await page.goto('/auth/register')
  await page.getByLabel(/email address/i).fill(email)
  await page.getByLabel(/date of birth/i).fill('1990-01-01')
  await page.getByLabel(/^password$/i).fill(password)
  await page.getByText(/I am at least 18/i).click()
  await page.getByText(/I accept the Fair Play/i).click()
  await page.getByRole('button', { name: /create identity/i }).click()
  await page.goto(await verificationLink(email))
  await expect(page.getByRole('heading', { name: /identity confirmed/i })).toBeVisible()

  await page.goto('/auth/login?next=%2Fwallet')
  await page.getByLabel(/email address/i).fill(email)
  await page.getByLabel(/^password$/i).fill(password)
  await page.getByRole('button', { name: /enter securely/i }).click()
  await expect(page).toHaveURL(/\/wallet$/)
  await expect(page.getByRole('heading', { name: /your money, with every stage visible/i })).toBeVisible()
  await expect(page.getByText(/no money movement has been requested/i)).toBeVisible()
  await capture(page, 'overview', testInfo.project.name)

  await page.getByRole('button', { name: 'Assessment', exact: true }).click()
  await page.getByLabel('Occupation').selectOption('employed')
  await page.getByLabel('Primary source of funds').selectOption('salary')
  await page.getByRole('button', { name: /submit for review/i }).click()
  await expect(page.getByText(/assessment submitted/i)).toBeVisible()
  await expect(page.getByRole('heading', { name: 'Submitted' })).toBeVisible()
  await capture(page, 'assessment-submitted', testInfo.project.name)

  await page.getByRole('button', { name: 'Deposit', exact: true }).click()
  await expect(page.getByText(/complete the financial assessment before moving live funds/i)).toBeVisible()
  await expect(page.getByRole('button', { name: /continue to provider/i })).toBeDisabled()
  await capture(page, 'deposit-policy-gate', testInfo.project.name)

  await page.getByRole('button', { name: 'Limits', exact: true }).click()
  await page.getByLabel('Daily deposit').fill('250')
  await page.getByLabel('Monthly deposit').fill('2500')
  await page.getByLabel('Daily withdrawal').fill('100')
  await page.getByLabel('Monthly withdrawal').fill('1000')
  await page.getByRole('button', { name: /apply controls/i }).click()
  await expect(page.getByText(/lower limits and responsible gaming controls are active/i)).toBeVisible()
  await capture(page, 'responsible-limits', testInfo.project.name)
})
