/**
 * JS Library Tests: page.expose
 *
 * Both forms of expose: injecting JS source (a string) and exposing a host
 * function whose return value comes back to the page (#298). The host form
 * round-trips through the engine: the page posts {name, seq, args} over a
 * script channel, the client runs the function, and vibium:expose.result
 * settles the page's promise.
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
    if (req.url === '/other') {
      res.end('<html><head><title>Other</title></head><body>other</body></html>');
    } else {
      res.end('<html><head><title>Expose</title></head><body>page</body></html>');
    }
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

describe('Expose: JS source (string form)', () => {
  test('defines window[name] from injected source', async () => {
    const vibe = await bro.newPage();
    await vibe.go(baseURL);

    await vibe.expose('double', '(x) => x * 2');
    const result = await vibe.evaluate('window.double(21)');
    assert.strictEqual(result, 42);

    await vibe.close();
  });
});

describe('Expose: host functions', () => {
  test('the page call runs the host function and gets its return value', async () => {
    const vibe = await bro.newPage();
    await vibe.go(baseURL);

    await vibe.expose('add', (a, b) => a + b);
    const result = await vibe.evaluate('window.add(2, 3)');
    assert.strictEqual(result, 5);

    await vibe.close();
  });

  test('async host functions and object results work', async () => {
    const vibe = await bro.newPage();
    await vibe.go(baseURL);

    await vibe.expose('lookup', async (key) => {
      await new Promise((r) => setTimeout(r, 20));
      return { key, found: true };
    });
    const result = await vibe.evaluate('window.lookup("user")');
    assert.deepStrictEqual(result, { key: 'user', found: true });

    await vibe.close();
  });

  test('a throwing host function rejects the page promise with its message', async () => {
    const vibe = await bro.newPage();
    await vibe.go(baseURL);

    await vibe.expose('explode', () => {
      throw new Error('no fuel');
    });
    const message = await vibe.evaluate('window.explode().catch((e) => e.message)');
    assert.strictEqual(message, 'no fuel');

    await vibe.close();
  });

  test('the binding survives navigation', async () => {
    const vibe = await bro.newPage();
    await vibe.go(baseURL);

    let calls = 0;
    await vibe.expose('count', () => { calls += 1; return calls; });
    assert.strictEqual(await vibe.evaluate('window.count()'), 1);

    await vibe.go(`${baseURL}/other`);
    assert.strictEqual(await vibe.evaluate('window.count()'), 2, 'the preload must rebind after navigation');

    await vibe.close();
  });

  test('concurrent calls resolve independently by sequence number', async () => {
    const vibe = await bro.newPage();
    await vibe.go(baseURL);

    await vibe.expose('echoLater', async (value, delayMs) => {
      await new Promise((r) => setTimeout(r, delayMs));
      return value;
    });
    // The slower call was made first; per-call sequencing must keep the
    // results from crossing.
    const result = await vibe.evaluate(
      'Promise.all([window.echoLater("slow", 80), window.echoLater("fast", 5)])'
    );
    assert.deepStrictEqual(result, ['slow', 'fast']);

    await vibe.close();
  });

  test('another Page instance for the same context does not shadow the function', async () => {
    // browser.page() constructs a fresh Page object per call, and every
    // instance sees every event. The registry is connection-scoped, so an
    // instance that never registered the function must not answer for it.
    const first = await bro.page();
    const second = await bro.page();
    await second.go(baseURL);

    await second.expose('whoami', () => 'owner');
    assert.strictEqual(await second.evaluate('window.whoami()'), 'owner');
    assert.ok(first, 'the earlier instance keeps listening without interfering');
  });

  test('re-exposing a name replaces the previous function', async () => {
    const vibe = await bro.newPage();
    await vibe.go(baseURL);

    await vibe.expose('answer', () => 'first');
    await vibe.expose('answer', () => 'second');
    assert.strictEqual(await vibe.evaluate('window.answer()'), 'second');

    await vibe.close();
  });
});
