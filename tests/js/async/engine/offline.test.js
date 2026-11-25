/**
 * JS Library Tests: page.setOffline() (#713)
 */

const { test, describe, before, after } = require("../../../helpers/capabilities").suite("core");
const assert = require('node:assert');
const http = require('http');

const { browser } = require('../../../../clients/javascript/dist');

let server, baseURL, bro;

before(async () => {
  server = http.createServer((req, res) => {
    res.writeHead(200, { 'Content-Type': 'text/html', 'Cache-Control': 'no-store' });
    res.end('<html><head><title>Offline</title></head><body></body></html>');
  });
  await new Promise((resolve) => server.listen(0, '127.0.0.1', resolve));
  baseURL = `http://127.0.0.1:${server.address().port}`;
  bro = await browser.start({ headless: true });
});

after(async () => {
  if (bro) await bro.stop();
  if (server) server.close();
});

const fetched = (page) => page.evaluate("fetch('/data').then(() => 'served').catch(() => 'failed')");

describe('page.setOffline', () => {
  test('takes one page offline and back online', async () => {
    const vibe = await bro.newPage();
    const other = await bro.newPage();
    try {
      await vibe.go(baseURL);
      await other.go(baseURL);

      await vibe.setOffline(true);
      assert.strictEqual(await fetched(vibe), 'failed');
      assert.strictEqual(await vibe.evaluate('navigator.onLine'), false);
      assert.strictEqual(await fetched(other), 'served', 'another page stays online');

      await vibe.setOffline(false);
      assert.strictEqual(await fetched(vibe), 'served');
      assert.strictEqual(await vibe.evaluate('navigator.onLine'), true);
    } finally {
      await vibe.close();
      await other.close();
    }
  });
});
