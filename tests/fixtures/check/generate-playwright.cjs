// Test-fixture generation only; Playwright is not a Vibium runtime dependency.
// PLAYWRIGHT_PACKAGE=/path/to/node_modules/playwright CHROME_EXECUTABLE=/path/to/chrome node generate-playwright.cjs
const { chromium } = require(process.env.PLAYWRIGHT_PACKAGE || 'playwright');
const path = require('node:path');
(async () => {
  const browser = await chromium.launch({ headless: true, executablePath: process.env.CHROME_EXECUTABLE });
  try {
    const context = await browser.newContext({ viewport: { width: 800, height: 600 } });
    await context.tracing.start({ screenshots: true, snapshots: true });
    const page = await context.newPage();
    await page.route('http://verify-fixture.test/**', route => route.fulfill({
      contentType: 'text/html', body: '<!doctype html><title>Checkout fixture</title><h1>Checkout</h1><button onclick="document.querySelector(\'h1\').textContent=\'Order confirmed\'; console.log(\'Order 123 confirmed\')">Place order</button>',
    }));
    await page.goto('http://verify-fixture.test/checkout');
    await page.getByRole('button', { name: 'Place order' }).click();
    await page.getByRole('heading', { name: 'Order confirmed' }).waitFor();
    await page.screenshot();
    await context.tracing.stop({ path: path.join(__dirname, 'playwright-checkout.zip') });
    await context.close();
  } finally { await browser.close(); }
})().catch(error => { console.error(error); process.exitCode = 1; });
