/**
 * JS Library Tests: page.setLocale() (#717)
 */

const { test, describe, before, after } = require("../../../helpers/capabilities").suite("core");
const assert = require('node:assert');
const http = require('http');

const { browser } = require('../../../../clients/javascript/dist');

let server, baseURL, bro;

before(async () => {
  server = http.createServer((req, res) => {
    res.writeHead(200, { 'Content-Type': 'text/html', 'Cache-Control': 'no-store' });
    res.end(`<html><head><title>Locale</title></head><body><pre id="sent">${req.headers['accept-language'] || ''}</pre></body></html>`);
  });
  await new Promise((resolve) => server.listen(0, '127.0.0.1', resolve));
  baseURL = `http://127.0.0.1:${server.address().port}`;
  bro = await browser.start({ headless: true });
});

after(async () => {
  if (bro) await bro.stop();
  if (server) server.close();
});

const formatted = (page) => page.evaluate('(1234.5).toLocaleString()');

describe('page.setLocale', () => {
  test('sets the language a page reports, formats with and sends', async () => {
    const vibe = await bro.newPage();
    const other = await bro.newPage();
    try {
      await vibe.go(baseURL);
      await other.go(baseURL);
      const before = await formatted(vibe);

      await vibe.setLocale('de-DE');
      assert.strictEqual(await vibe.evaluate('navigator.language'), 'de-DE');
      assert.strictEqual(await formatted(vibe), '1.234,5');
      await vibe.reload();
      assert.match(await vibe.evaluate("document.getElementById('sent').textContent"), /^de-DE/, 'Accept-Language header');
      assert.strictEqual(await formatted(other), before, 'another page keeps its locale');

      await vibe.setLocale(null);
      assert.strictEqual(await formatted(vibe), before, 'null removes the override');
    } finally {
      await vibe.close();
      await other.close();
    }
  });

  test('rejects a tag that is not a locale', async () => {
    const vibe = await bro.newPage();
    try {
      await vibe.go(baseURL);
      await assert.rejects(vibe.setLocale('not a locale'), /failed to set the locale/);
    } finally {
      await vibe.close();
    }
  });
});
