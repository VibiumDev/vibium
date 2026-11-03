/**
 * JS Library Tests: Persistent Profile (#715)
 * userDataDir keeps profile state (cookies, localStorage) across sessions,
 * and the directory is never deleted on close.
 *
 * Uses a local HTTP server — no external network dependencies.
 */

const { test, describe, before, after } = require("../../../helpers/capabilities").suite("core");
const assert = require('node:assert');
const fs = require('node:fs');
const http = require('node:http');
const os = require('node:os');
const path = require('node:path');

const { browser } = require('../../../../clients/javascript/dist');

let server;
let baseURL;

before(async () => {
  server = http.createServer((req, res) => {
    res.writeHead(200, { 'Content-Type': 'text/html' });
    res.end('<html><head><title>Profile Test</title></head><body>Hello</body></html>');
  });
  await new Promise((resolve) => {
    server.listen(0, '127.0.0.1', () => {
      baseURL = `http://127.0.0.1:${server.address().port}`;
      resolve();
    });
  });
});

after(() => {
  if (server) server.close();
});

describe('Persistent profile: userDataDir', () => {
  test('state persists across sessions and the dir survives close', async () => {
    const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'js-profile-'));
    try {
      const bro = await browser.start({ headless: true, userDataDir: dir });
      try {
        const vibe = await bro.page();
        await vibe.go(baseURL);
        await vibe.evaluate('localStorage.setItem("persisted_key", "yes")');
        // Let the write reach disk before stopping: storage commits are
        // asynchronous, and a host where the browser cannot complete a clean
        // shutdown (broken-GPU macOS VMs) never gets the exit-time flush.
        await vibe.wait(2000);
      } finally {
        await bro.stop();
      }

      assert.ok(fs.existsSync(dir), 'profile dir must survive browser close');
      assert.ok(fs.readdirSync(dir).length > 0, 'profile dir must keep its contents');

      const bro2 = await browser.start({ headless: true, userDataDir: dir });
      try {
        const vibe = await bro2.page();
        await vibe.go(baseURL);
        const stored = await vibe.evaluate('localStorage.getItem("persisted_key")');
        assert.strictEqual(stored, 'yes', 'localStorage must survive a restart');
      } finally {
        await bro2.stop();
      }

      assert.ok(fs.existsSync(dir), 'profile dir must survive the second close too');
    } finally {
      fs.rmSync(dir, { recursive: true, force: true });
    }
  });
});
