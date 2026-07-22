import { defineConfig, devices } from '@playwright/test'

export default defineConfig({
  testDir: './e2e',
  timeout: 45_000,
  expect: { timeout: 10_000 },
  fullyParallel: false,
  workers: 1,
  retries: 0,
  reporter: [['list'], ['html', { open: 'never' }]],
  use: {
    baseURL: 'http://127.0.0.1:13000',
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
    video: 'on',
  },
  projects: [
    { name: 'desktop-chromium', use: { ...devices['Desktop Chrome'] } },
    { name: 'tablet-chromium', use: { ...devices['Desktop Chrome'], viewport: { width: 1024, height: 1366 }, hasTouch: true } },
    { name: 'mobile-chromium', use: { ...devices['Pixel 7'] } },
  ],
  webServer: [
    {
      command: String.raw`powershell -NoProfile -Command "$target=Join-Path (Get-Location) '.e2e-data'; if (-not $target.StartsWith((Get-Location).Path)) { throw 'Unsafe E2E data path' }; if (Test-Path $target) { Remove-Item -LiteralPath $target -Recurse -Force }; $env:SKILL_ARENA_JWT_SECRET='e2e-jwt-secret-at-least-32-characters'; $env:SKILL_ARENA_DATABASE_URL='.e2e-data'; $env:SKILL_ARENA_HTTP_ADDR='127.0.0.1:18080'; $env:SKILL_ARENA_PUBLIC_BASE_URL='http://127.0.0.1:13000'; $env:SKILL_ARENA_ALLOWED_ORIGINS='http://127.0.0.1:13000'; $env:SKILL_ARENA_SUPER_ADMINS='mfa-desktop-chromium@example.com,mfa-tablet-chromium@example.com,mfa-mobile-chromium@example.com'; & 'C:\Program Files\Go\bin\go.exe' run ./cmd/api"`,
      cwd: '../backend',
      url: 'http://127.0.0.1:18080/health/ready',
      timeout: 120_000,
      reuseExistingServer: false,
    },
    {
      command: `powershell -NoProfile -Command "$env:NEXT_PUBLIC_API_BASE_URL='http://127.0.0.1:18080'; npm.cmd run build; if ($LASTEXITCODE) { exit $LASTEXITCODE }; npm.cmd run start -- --hostname 127.0.0.1 --port 13000"`,
      cwd: '.',
      url: 'http://127.0.0.1:13000',
      timeout: 120_000,
      reuseExistingServer: false,
    },
  ],
})
