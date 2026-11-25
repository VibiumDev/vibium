/**
 * JS Library Tests: page.setPermission() (#711)
 */

const { test, describe, before, after } = require("../../../helpers/capabilities").suite("core");
const assert = require('node:assert');
const http = require('http');

const { browser } = require('../../../../clients/javascript/dist');

let server, baseURL, bro;

before(async () => {
  server = http.createServer((req, res) => {
    res.writeHead(200, { 'Content-Type': 'text/html' });
    res.end('<html><head><title>Permissions</title></head><body></body></html>');
  });
  await new Promise((resolve) => server.listen(0, '127.0.0.1', resolve));
  baseURL = `http://127.0.0.1:${server.address().port}`;
  bro = await browser.start({ headless: true });
});

after(async () => {
  if (bro) await bro.stop();
  if (server) server.close();
});

const state = (page, name = 'geolocation') =>
  page.evaluate(`navigator.permissions.query({ name: '${name}' }).then((s) => s.state)`);

describe('page.setPermission', () => {
  test('grants, denies and resets a permission for the page origin', async () => {
    const vibe = await bro.newPage();
    try {
      await vibe.go(baseURL);
      await vibe.setPermission('geolocation', 'granted');
      assert.strictEqual(await state(vibe), 'granted');
      await vibe.setPermission('geolocation', 'denied');
      assert.strictEqual(await state(vibe), 'denied');
      await vibe.setPermission('geolocation', 'prompt');
      assert.strictEqual(await state(vibe), 'prompt');
    } finally {
      await vibe.close();
    }
  });

  test('an explicit origin leaves other origins alone', async () => {
    const vibe = await bro.newPage();
    try {
      await vibe.go(baseURL);
      await vibe.setPermission('geolocation', 'prompt');
      await vibe.setPermission('geolocation', 'granted', { origin: 'https://other.example' });
      assert.strictEqual(await state(vibe), 'prompt');
    } finally {
      await vibe.close();
    }
  });

  test('applies to the browser context the page belongs to', async () => {
    const context = await bro.newContext();
    const inContext = await context.newPage();
    const inDefault = await bro.newPage();
    try {
      await inContext.go(baseURL);
      await inDefault.go(baseURL);
      await inDefault.setPermission('geolocation', 'prompt');
      await inContext.setPermission('geolocation', 'granted');
      assert.strictEqual(await state(inContext), 'granted');
      assert.strictEqual(await state(inDefault), 'prompt');
    } finally {
      await inDefault.close();
      await context.close();
    }
  });

  test('rejects an unknown state, an unknown permission and a page with no origin', async () => {
    const vibe = await bro.newPage();
    try {
      await vibe.go(baseURL);
      await assert.rejects(vibe.setPermission('geolocation', 'allowed'), /state must be one of granted, denied, prompt/);
      await assert.rejects(vibe.setPermission('not-a-permission', 'granted'), /failed to set the not-a-permission permission/);
      await vibe.go('about:blank');
      await assert.rejects(vibe.setPermission('geolocation', 'granted'), /no origin to set a permission for/);
    } finally {
      await vibe.close();
    }
  });
});
