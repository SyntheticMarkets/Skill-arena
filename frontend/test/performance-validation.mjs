import { chromium } from '@playwright/test'

async function main() {
  const browser = await chromium.launch({ headless: true })
  const samples = []
  for (let index = 0; index < 15; index += 1) {
    const context = await browser.newContext({ viewport: { width: 1440, height: 900 } })
    const page = await context.newPage()
    const started = performance.now()
    await page.goto('http://127.0.0.1:3000', { waitUntil: 'load' })
    await page.getByRole('heading', { name: /where skill becomes value/i }).waitFor()
    const wall = performance.now() - started
    const navigation = await page.evaluate(() => {
      const entry = performance.getEntriesByType('navigation')[0]
      return {
        response: entry.responseEnd,
        dom: entry.domContentLoadedEventEnd,
        load: entry.loadEventEnd,
      }
    })
    samples.push({ wall, ...navigation })
    await context.close()
  }
  await browser.close()

  const metric = (key) => {
    const values = samples.map((sample) => sample[key]).sort((left, right) => left - right)
    return {
      min: Number(values[0].toFixed(1)),
      p50: Number(values[Math.floor(values.length * 0.5)].toFixed(1)),
      p95: Number(values[Math.min(values.length - 1, Math.floor(values.length * 0.95))].toFixed(1)),
      max: Number(values[values.length - 1].toFixed(1)),
    }
  }

  console.log(JSON.stringify({
    runs: samples.length,
    responseEndMs: metric('response'),
    domContentLoadedMs: metric('dom'),
    loadEventMs: metric('load'),
    headingReadyWallMs: metric('wall'),
  }, null, 2))
}

main().catch((error) => {
  console.error(error)
  process.exit(1)
})
