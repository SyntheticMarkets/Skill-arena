import { defineConfig, devices } from "@playwright/test";

export default defineConfig({
  testDir: "./e2e",
  timeout: 120_000,
  expect: { timeout: 12_000 },
  fullyParallel: false,
  workers: 1,
  retries: 0,
  reporter: [["list"], ["html", { open: "never" }]],
  use: {
    baseURL: "http://127.0.0.1:13100",
    trace: "retain-on-failure",
    screenshot: "only-on-failure",
    video: "retain-on-failure"
  },
  projects: [
    { name: "desktop-chromium", use: { ...devices["Desktop Chrome"], viewport: { width: 1440, height: 1000 } } },
    { name: "tablet-chromium", use: { ...devices["Desktop Chrome"], viewport: { width: 1024, height: 1366 }, hasTouch: true } },
    { name: "mobile-chromium", use: { ...devices["Pixel 7"] } }
  ],
  webServer: [
    {
      command: String.raw`powershell -NoProfile -Command "$target=Join-Path (Get-Location) '.admin-e2e-data'; if (-not $target.StartsWith((Get-Location).Path)) { throw 'Unsafe E2E data path' }; if (Test-Path $target) { Remove-Item -LiteralPath $target -Recurse -Force }; $env:SKILL_ARENA_JWT_SECRET='admin-e2e-jwt-secret-at-least-32-characters'; $env:SKILL_ARENA_DATABASE_URL='.admin-e2e-data'; $env:SKILL_ARENA_HTTP_ADDR='127.0.0.1:18180'; $env:SKILL_ARENA_PUBLIC_BASE_URL='http://127.0.0.1:13100'; $env:SKILL_ARENA_ADMIN_BASE_URL='http://127.0.0.1:13100'; $env:SKILL_ARENA_ALLOWED_ORIGINS='http://127.0.0.1:13100'; $env:SKILL_ARENA_RATE_REGISTER_LIMIT='100'; $env:SKILL_ARENA_RATE_LOGIN_LIMIT='100'; $env:SKILL_ARENA_SUPER_ADMINS='admin-desktop-chromium@example.com,admin-tablet-chromium@example.com,admin-mobile-chromium@example.com'; & 'C:\Program Files\Go\bin\go.exe' run ./cmd/api"`,
      cwd: "../backend",
      url: "http://127.0.0.1:18180/health/ready",
      timeout: 120_000,
      reuseExistingServer: false
    },
    {
      command: `powershell -NoProfile -Command "$env:SKILL_ARENA_API_INTERNAL_URL='http://127.0.0.1:18180'; npm.cmd run build; if ($LASTEXITCODE) { exit $LASTEXITCODE }; Copy-Item -LiteralPath '.next/static' -Destination '.next/standalone/.next/static' -Recurse -Force; $env:PORT='13100'; $env:HOSTNAME='127.0.0.1'; Push-Location '.next/standalone'; try { node server.js } finally { Pop-Location }"`,
      cwd: ".",
      url: "http://127.0.0.1:13100/login",
      timeout: 120_000,
      reuseExistingServer: false
    }
  ]
});
