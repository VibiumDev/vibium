/**
 * JS Library Tests: commands against a prompt-blocked context
 *
 * The browser is launched with unhandledPromptBehavior "ignore", so an alert
 * opened by a click stays open and Chrome answers no script or input command
 * for that context. Without prompt tracking every such command sat out the full
 * 60s BiDi timeout (#151, #146).
 *
 * Uses a local HTTP server — no external network dependencies.
 */

const { test, describe, before, after } = require("../../../helpers/capabilities").suite("core", "prompt-blocking");
const assert = require('node:assert');
const http = require('http');

const { browser } = require('../../../../clients/javascript/dist');

let server;
let baseURL;
let bro;

before(async () => {
  server = http.createServer((req, res) => {
    res.writeHead(200, { 'Content-Type': 'text/html' });
    res.end(`<html><head><title>Prompt</title></head><body>
      <button id="alert-btn" onclick="alert('blocked')">Alert</button>
    </body></html>`);
  });
  await new Promise((resolve) => {
    server.listen(0, '127.0.0.1', () => {
      baseURL = `http://127.0.0.1:${server.address().port}`;
      resolve();
    });
  });
  bro = await browser.start({ headless: true });
});

after(async () => {
  await bro.stop();
  if (server) server.close();
});

describe('Prompt-blocked context', () => {
  test('a command fails fast instead of timing out', async () => {
    const vibe = await bro.newPage();
    await vibe.go(baseURL);

    // Observe the dialog without handling it, so the alert stays open and
    // blocks the context. Waiting on the event rather than a fixed sleep keeps
    // the test from racing the click.
    let dialogOpen = false;
    vibe.onDialog(() => { dialogOpen = true; });

    vibe.find('#alert-btn').click().catch(() => {});
    for (let i = 0; i < 50 && !dialogOpen; i++) {
      await new Promise((r) => setTimeout(r, 100));
    }
    assert.ok(dialogOpen, 'the alert should have opened');

    const started = Date.now();
    let message = '';
    try {
      await vibe.title();
      assert.fail('title() should fail while a dialog is open');
    } catch (e) {
      message = String(e.message);
    }
    const elapsed = Date.now() - started;

    assert.match(
      message,
      /blocked by an open .* dialog/,
      `expected an actionable prompt error, got: ${message}`
    );
    assert.ok(
      elapsed < 5000,
      `expected a fast failure, took ${elapsed}ms (the BiDi command timeout is 60s)`
    );

    await vibe.close();
  });

  test('dismissing the dialog unblocks the context', async () => {
    const vibe = await bro.newPage();
    await vibe.go(baseURL);

    vibe.onDialog((dialog) => {
      dialog.accept().catch(() => {});
    });

    await vibe.find('#alert-btn').click();
    await vibe.wait(500);

    // Once the prompt is handled the context must be usable again — the
    // tracker has to clear on userPromptClosed, not just set on Opened.
    const title = await vibe.title();
    assert.strictEqual(title, 'Prompt');

    await vibe.close();
  });

  test('a prompt on one page does not block another page', async () => {
    const blocked = await bro.newPage();
    const other = await bro.newPage();
    await blocked.go(baseURL);
    await other.go(baseURL);

    let dialogOpen = false;
    blocked.onDialog(() => { dialogOpen = true; });
    blocked.find('#alert-btn').click().catch(() => {});
    for (let i = 0; i < 50 && !dialogOpen; i++) {
      await new Promise((r) => setTimeout(r, 100));
    }
    assert.ok(dialogOpen, 'the alert should have opened on the first page');

    const title = await other.title();
    assert.strictEqual(title, 'Prompt', 'a second page must be unaffected');

    await blocked.close();
    await other.close();
  });
});
