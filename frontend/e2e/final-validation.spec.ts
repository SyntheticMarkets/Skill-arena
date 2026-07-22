import { expect, test } from '@playwright/test'
import { createHmac } from 'node:crypto'
import { promises as fs } from 'node:fs'
import path from 'node:path'

const outbox = path.resolve(__dirname, '../../backend/.e2e-data/email_outbox')
const proof = path.resolve(__dirname, '../../docs/proof/sprint-1-final-validation')

async function capture(page: import('@playwright/test').Page, step: string, project: string) {
  await fs.mkdir(proof, { recursive: true })
  await page.screenshot({ path: path.join(proof, `${step}-${project}.png`), fullPage: true })
}

async function emailLink(recipient: string, route: string) {
  const deadline = Date.now() + 20_000
  while (Date.now() < deadline) {
    const entries = await fs.readdir(outbox).catch(() => [] as string[])
    const messages = await Promise.all(entries.map(async (name) => ({
      content: await fs.readFile(path.join(outbox, name), 'utf8'),
      stat: await fs.stat(path.join(outbox, name)),
    })))
    messages.sort((a, b) => b.stat.mtimeMs - a.stat.mtimeMs)
    const message = messages.find((item) => item.content.includes(`To: ${recipient}`) && item.content.includes(route))
    const match = message?.content.match(/https?:\/\/[^\s<"]+/)
    if (match) return match[0].replace(/&amp;/g, '&')
    await new Promise((resolve) => setTimeout(resolve, 250))
  }
  throw new Error(`No ${route} email arrived for ${recipient}`)
}

function decodeBase32(value: string) {
  const alphabet = 'ABCDEFGHIJKLMNOPQRSTUVWXYZ234567'
  let bits = ''
  for (const char of value.replace(/=+$/, '').toUpperCase()) bits += alphabet.indexOf(char).toString(2).padStart(5, '0')
  const bytes: number[] = []
  for (let index = 0; index + 8 <= bits.length; index += 8) bytes.push(Number.parseInt(bits.slice(index, index + 8), 2))
  return Buffer.from(bytes)
}

function totp(secret: string, now = Date.now()) {
  const counter = Buffer.alloc(8)
  counter.writeBigUInt64BE(BigInt(Math.floor(now / 30_000)))
  const digest = createHmac('sha1', decodeBase32(secret)).update(counter).digest()
  const offset = digest[digest.length - 1] & 0x0f
  const value = (digest.readUInt32BE(offset) & 0x7fffffff) % 1_000_000
  return value.toString().padStart(6, '0')
}

async function registerAndVerify(page: import('@playwright/test').Page, email: string, password: string, project: string) {
  await page.goto('/auth/register')
  await capture(page, 'register', project)
  await page.getByLabel(/email address/i).fill(email)
  await page.getByLabel(/date of birth/i).fill('1990-01-01')
  await page.getByLabel(/^password$/i).fill(password)
  await page.getByText(/I am at least 18/i).click()
  await page.getByText(/I accept the Fair Play/i).click()
  await page.getByRole('button', { name: /create identity/i }).click()
  await expect(page).toHaveURL(/verification-pending/)
  await capture(page, 'verification-pending', project)
  await page.goto(await emailLink(email, '/auth/verify-email'))
  await expect(page.getByRole('heading', { name: /identity confirmed/i })).toBeVisible()
  await capture(page, 'verify-email', project)
}

test('every public and recovery page has responsive proof', async ({ page }, testInfo) => {
  const project = testInfo.project.name
  await page.route('**/api/v1/auth/session', async (route) => {
    await new Promise((resolve) => setTimeout(resolve, 1_200))
    await route.fulfill({ status: 401, contentType: 'application/json', body: JSON.stringify({ code: 'AUTH_UNAUTHORIZED', message: 'authentication required' }) })
  }, { times: 1 })
  await page.goto('/')
  await expect(page.getByText(/preparing your arena/i)).toBeVisible()
  await capture(page, 'boot', project)
  await expect(page.getByRole('heading', { name: /where skill becomes value/i })).toBeVisible()
  await capture(page, 'landing', project)
  await page.goto('/arena')
  await expect(page.getByRole('heading', { name: /choose the skill/i })).toBeVisible()
  await capture(page, 'guest-arena', project)
  await page.goto('/auth/login')
  await capture(page, 'login', project)
  await page.goto('/auth/forgot-password')
  await capture(page, 'forgot-password', project)
  await page.goto('/auth/reset-password?token=invalid-proof-token')
  await capture(page, 'password-reset', project)
})

test('MFA enrollment, MFA login, recovery code, session recovery, and logout', async ({ page, context }, testInfo) => {
  const project = testInfo.project.name
  const email = `mfa-${project}@example.com`
  const password = 'PrivilegedPassword!42'
  await registerAndVerify(page, email, password, project)
  await page.getByRole('link', { name: /continue to login/i }).click()
  await page.getByLabel(/email address/i).fill(email)
  await page.getByLabel(/^password$/i).fill(password)
  await page.getByRole('button', { name: /enter securely/i }).click()
  await expect(page).toHaveURL(/\/auth\/mfa\/setup/)
  await expect(page.locator('.mfa-setup code')).toBeVisible()
  const secret = (await page.locator('.mfa-setup code').textContent())?.trim() || ''
  expect(secret).not.toBe('')
  await capture(page, 'mfa-enrollment', project)
  await page.getByLabel(/authenticator code/i).fill(totp(secret))
  await page.getByRole('button', { name: /enable mfa/i }).click()
  await expect(page.getByRole('heading', { name: /store your recovery codes/i })).toBeVisible()
  const recoveryCode = (await page.locator('.code-grid code').first().textContent())?.trim() || ''
  expect(recoveryCode).not.toBe('')
  await capture(page, 'mfa-recovery-codes', project)
  await page.getByRole('button', { name: /stored them securely/i }).click()
  await expect(page).toHaveURL(/\/arena$/)
  await capture(page, 'session-recovery', project)

  await context.clearCookies()
  await page.goto('/auth/login')
  await page.getByLabel(/email address/i).fill(email)
  await page.getByLabel(/^password$/i).fill(password)
  await page.getByRole('button', { name: /enter securely/i }).click()
  await expect(page).toHaveURL(/\/auth\/mfa/)
  await capture(page, 'mfa-login', project)
  await page.getByLabel(/six-digit code/i).fill('000000')
  await page.getByRole('button', { name: /verify and enter/i }).click()
  await expect(page.getByText(/invalid mfa/i)).toBeVisible()
  await capture(page, 'mfa-login-invalid', project)
  await page.getByLabel(/six-digit code/i).fill(totp(secret))
  await page.getByRole('button', { name: /verify and enter/i }).click()
  await expect(page).toHaveURL(/\/arena$/)
  await page.reload()
  await expect(page.getByRole('heading', { name: /competitor identity is ready/i })).toBeVisible()

  const logout = page.getByRole('button', { name: /log out/i })
  await expect(logout).toBeVisible()
  await logout.click()
  await expect(page.getByRole('link', { name: /create identity/i }).first()).toBeVisible()
  await capture(page, 'logout', project)

  await page.goto('/auth/login')
  await page.getByLabel(/email address/i).fill(email)
  await page.getByLabel(/^password$/i).fill(password)
  await page.getByRole('button', { name: /enter securely/i }).click()
  await page.getByRole('button', { name: /recovery code/i }).click()
  await page.getByLabel(/recovery code/i).fill(recoveryCode)
  await page.getByRole('button', { name: /verify and enter/i }).click()
  await expect(page).toHaveURL(/\/arena$/)
  await capture(page, 'mfa-recovery-login', project)
})
