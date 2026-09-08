/**
 * JS Library Tests: command dispatch concurrency
 *
 * The engine serialized every dispatched vibium: command on a per-session mutex
 * whose only purpose is keeping a recording's before/after snapshots consistent.
 * It was taken unconditionally, so all 104 dispatched methods ran one at a time
 * even with recording off.
 *
 * Uses a local HTTP server — no external network dependencies.
 */

const { test, describe, before, after } = require("../../../helpers/capabilities").suite("core");
const assert = require('node:assert');
const http = require('http');

const { browser } = require('../../../../clients/javascript/dist');

let server;
let baseURL;
let bro;

before(async () => {
  server = http.createServer((req, res) => {
    res.writeHead(200, { 'Content-Type': 'text/html' });
    res.end('<html><head><title>Dispatch</title></head><body><h1>Dispatch</h1></body></html>');
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

describe('Dispatch concurrency', () => {
  test('commands overlap when not recording', async () => {
    const vibe = await bro.newPage();
    await vibe.go(baseURL);

    // page.wait sleeps server-side for the requested duration, so three
    // concurrent 800ms waits take ~800ms if they overlap and ~2400ms if the
    // dispatcher serializes them.
    const started = Date.now();
    await Promise.all([vibe.wait(800), vibe.wait(800), vibe.wait(800)]);
    const elapsed = Date.now() - started;

    assert.ok(
      elapsed < 2000,
      `three concurrent 800ms waits took ${elapsed}ms — dispatch is still serialized`
    );
    await vibe.close();
  });

  test('a second page is not blocked by a slow command on the first', async () => {
    const one = await bro.newPage();
    const two = await bro.newPage();
    await one.go(baseURL);
    await two.go(baseURL);

    const started = Date.now();
    const slow = one.wait(1500);
    const title = await two.title();
    const titleElapsed = Date.now() - started;

    assert.strictEqual(title, 'Dispatch');
    assert.ok(
      titleElapsed < 1200,
      `title() on a second page waited ${titleElapsed}ms behind a slow command on the first`
    );

    await slow;
    await one.close();
    await two.close();
  });
});
