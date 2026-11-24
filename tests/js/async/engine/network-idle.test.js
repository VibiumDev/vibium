/**
 * JS Library Tests: waitForLoad('networkidle') (#846)
 * A page that fetches after load has nothing else to wait on.
 */

const { test, describe, before, after } = require("../../../helpers/capabilities").suite("core");
const assert = require('node:assert');
const http = require('http');

const { browser } = require('../../../../clients/javascript/dist');

const LATE_DATA_HTML = `<html><head><title>Late data</title></head><body>
  <div id="out">loading</div>
  <script>
    window.addEventListener('load', async () => {
      const first = await fetch('/slow?ms=600').then((r) => r.text());
      const second = await fetch('/slow?ms=600').then((r) => r.text());
      document.getElementById('out').textContent = first + ' ' + second;
    });
  </script>
</body></html>`;

let server, baseURL, bro;

before(async () => {
  server = http.createServer((req, res) => {
    const url = new URL(req.url, 'http://localhost');
    if (url.pathname === '/slow') {
      setTimeout(() => {
        res.writeHead(200, { 'Content-Type': 'text/plain', 'Cache-Control': 'no-store' });
        res.end('ok');
      }, Number(url.searchParams.get('ms')));
      return;
    }
    res.writeHead(200, { 'Content-Type': 'text/html' });
    res.end(LATE_DATA_HTML);
  });
  await new Promise((resolve) => server.listen(0, '127.0.0.1', resolve));
  baseURL = `http://127.0.0.1:${server.address().port}`;
  bro = await browser.start({ headless: true });
});

after(async () => {
  if (bro) await bro.stop();
  if (server) {
    server.closeAllConnections();
    server.close();
  }
});

describe('waitForLoad networkidle', () => {
  test('waits out requests a page starts after load', async () => {
    const vibe = await bro.newPage();
    try {
      await vibe.go(baseURL);
      await vibe.waitForLoad('networkidle', { timeout: 15000 });
      assert.strictEqual(await vibe.evaluate("document.getElementById('out').textContent"), 'ok ok');
    } finally {
      await vibe.close();
    }
  });

  test('a failed request does not count as in flight', async () => {
    const vibe = await bro.newPage();
    try {
      await vibe.go(baseURL);
      await vibe.waitForLoad('networkidle', { timeout: 15000 });
      await vibe.evaluate("(() => { fetch('http://127.0.0.1:1/refused').catch(() => {}); })()");
      await vibe.waitForLoad('networkidle', { timeout: 15000 });
    } finally {
      await vibe.close();
    }
  });

  test('times out while a request is in flight, and says how many', async () => {
    const vibe = await bro.newPage();
    try {
      await vibe.go(baseURL);
      await vibe.waitForLoad('networkidle', { timeout: 15000 });
      await vibe.evaluate("(() => { fetch('/slow?ms=4000'); })()");
      await assert.rejects(
        vibe.waitForLoad('networkidle', { timeout: 1000 }),
        /waiting for the network to be idle: 1 requests in flight/,
      );
    } finally {
      await vibe.close();
    }
  });
});
