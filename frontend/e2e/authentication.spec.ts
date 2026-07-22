import { expect, test } from '@playwright/test'
import { promises as fs } from 'node:fs'
import path from 'node:path'

const outbox = path.resolve(__dirname, '../../backend/.e2e-data/email_outbox')
const proof = path.resolve(__dirname, '../../docs/proof/sprint-1')

async function capture(page: import('@playwright/test').Page, name: string) {
  await fs.mkdir(proof, { recursive: true })
  await page.screenshot({ path: path.join(proof, `${name}.png`), fullPage: true })
}

async function emailLink(recipient: string, pathName: string) {
  const deadline = Date.now() + 15_000
  while (Date.now() < deadline) {
    const entries = await fs.readdir(outbox).catch(() => [] as string[])
    const messages = await Promise.all(entries.map(async (name) => ({
      name,
      content: await fs.readFile(path.join(outbox, name), 'utf8'),
      stat: await fs.stat(path.join(outbox, name)),
    })))
    messages.sort((a, b) => b.stat.mtimeMs - a.stat.mtimeMs)
    const message = messages.find((item) => item.content.includes(`To: ${recipient}`) && item.content.includes(pathName))
    const match = message?.content.match(/https?:\/\/[^\s<"]+/)
    if (match) return match[0].replace(/&amp;/g, '&')
    await new Promise((resolve) => setTimeout(resolve, 250))
  }
  throw new Error(`No ${pathName} email arrived for ${recipient}`)
}

test('visitor can explore before registration', async ({ page }, testInfo) => {
  await page.goto('/')
  await expect(page.getByRole('heading', { name: /where skill becomes value/i })).toBeVisible()
  await capture(page, `landing-${testInfo.project.name}`)
  await page.getByRole('link', { name: /explore the arena/i }).first().click()
  await expect(page).toHaveURL(/\/arena$/)
  await expect(page.getByRole('heading', { name: /choose the skill/i })).toBeVisible()
  await expect(page.getByRole('heading', { name: 'Maze Arena' })).toBeVisible()
  await capture(page, `guest-arena-${testInfo.project.name}`)
})

test('registration, verification, recovery, and password reset work end to end', async ({ page, context }, testInfo) => {
  const email = `e2e-${testInfo.project.name}-${Date.now()}@example.com`
  const password = 'StrongPassword!42'
  const newPassword = 'ReplacementPassword!43'

  await page.goto('/auth/register')
  await capture(page, `journey-register-${testInfo.project.name}`)
  await page.getByLabel(/email address/i).fill(email)
  await page.getByLabel(/date of birth/i).fill('1990-01-01')
  await page.getByLabel(/^password$/i).fill(password)
  await page.getByText(/I am at least 18/i).click()
  await page.getByText(/I accept the Fair Play/i).click()
  await page.getByRole('button', { name: /create identity/i }).click()
  await expect(page).toHaveURL(/verification-pending/)
  await expect(page.getByText(/verification pending/i)).toBeVisible()
  await capture(page, `verification-pending-${testInfo.project.name}`)

  const verifyLink = await emailLink(email, '/auth/verify-email')
  await page.goto(verifyLink)
  await expect(page.getByRole('heading', { name: /identity confirmed/i })).toBeVisible()
  await capture(page, `journey-verify-email-${testInfo.project.name}`)
  await page.getByRole('link', { name: /continue to login/i }).click()
  await capture(page, `journey-login-${testInfo.project.name}`)
  await page.getByLabel(/email address/i).fill(email)
  await page.getByLabel(/^password$/i).fill(password)
  await page.getByRole('button', { name: /enter securely/i }).click()
  await expect(page).toHaveURL(/\/arena$/)

  await context.clearCookies()
  await page.goto('/auth/forgot-password')
  await capture(page, `journey-forgot-password-${testInfo.project.name}`)
  await page.getByLabel(/email address/i).fill(email)
  await page.getByRole('button', { name: /send recovery link/i }).click()
  await expect(page.getByText(/If the account exists/i)).toBeVisible()
  await capture(page, `journey-forgot-password-result-${testInfo.project.name}`)
  const resetLink = await emailLink(email, '/auth/reset-password')
  await page.goto(resetLink)
  await capture(page, `journey-password-reset-${testInfo.project.name}`)
  await page.getByLabel('New password', { exact: true }).fill(newPassword)
  await page.getByLabel('Confirm new password', { exact: true }).fill(newPassword)
  await page.getByRole('button', { name: /set new password/i }).click()
  await expect(page.getByText(/new password is active/i)).toBeVisible()
  await capture(page, `journey-password-reset-result-${testInfo.project.name}`)

  await page.getByRole('link', { name: /return to login/i }).click()
  await page.getByLabel(/email address/i).fill(email)
  await page.getByLabel(/^password$/i).fill(newPassword)
  await page.getByRole('button', { name: /enter securely/i }).click()
  await expect(page).toHaveURL(/\/arena$/)
  await expect(page.getByRole('heading', { name: /competitor identity is ready/i })).toBeVisible()
  await expect(page.getByRole('link', { name: /create competitor identity/i })).toHaveCount(0)
  await page.reload()
  await expect(page.getByRole('heading', { name: /competitor identity is ready/i })).toBeVisible()
  await capture(page, `authenticated-${testInfo.project.name}`)
})
