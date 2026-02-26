/**
 * JS Library Tests: Frames
 * Tests page.frames(), page.frame(nameOrUrl), page.mainFrame()
 */

const { test, describe, after } = require('node:test');
const assert = require('node:assert');
const http = require('http');

const { browser } = require('../../../clients/javascript/dist');

// Create a local HTTP server that serves pages with iframes
function startServer() {
  return new Promise((resolve) => {
    const server = http.createServer((req, res) => {
      res.setHeader('Content-Type', 'text/html');

      if (req.url === '/') {
        res.end(`<!DOCTYPE html>
<html><head><title>Main Page</title></head>
<body>
  <h1>Main Page</h1>
  <iframe src="/frame1" id="frame1"></iframe>
</body></html>`);
      } else if (req.url === '/frame1') {
        res.end(`<!DOCTYPE html>
<html><head><title>Frame 1</title></head>
<body>
  <h1>Frame 1 Content</h1>
  <iframe src="/frame2" name="nested"></iframe>
</body></html>`);
      } else if (req.url === '/frame2') {
        res.end(`<!DOCTYPE html>
<html><head><title>Frame 2</title></head>
<body>
  <h1>Nested Frame</h1>
  <p id="deep">Deep content</p>
</body></html>`);
      } else {
        res.statusCode = 404;
        res.end('Not found');
      }
    });

    server.listen(0, '127.0.0.1', () => {
      const port = server.address().port;
      resolve({ server, url: `http://127.0.0.1:${port}` });
    });
  });
}

describe('JS Frames', () => {
  let srv, baseUrl, bro;

  // Start server and browser once for all tests
  test('setup', async () => {
    const s = await startServer();
    srv = s.server;
    baseUrl = s.url;
    bro = await browser.launch({ headless: true });
  });

  after(async () => {
    if (bro) await bro.close().catch(() => {});
    if (srv) srv.close();
  });

  test('frames() returns child frames', async () => {
    const vibe = await bro.page();
    await vibe.go(baseUrl + '/');
    // Wait for iframes to load
    await vibe.wait(500);

    const frames = await vibe.frames();
    assert.ok(frames.length >= 1, `Should have at least 1 child frame, got ${frames.length}`);
  });

  test('frame(url) finds frame by URL substring', async () => {
    const vibe = await bro.page();
    await vibe.go(baseUrl + '/');
    await vibe.wait(500);

    const frame = await vibe.frame('/frame1');
    assert.ok(frame, 'Should find frame by URL substring');
    assert.ok(frame.id, 'Frame should have an id');
  });

  test('frame(name) finds frame by name attribute', async () => {
    const vibe = await bro.page();
    await vibe.go(baseUrl + '/');
    await vibe.wait(500);

    const frame = await vibe.frame('nested');
    assert.ok(frame, 'Should find frame by name attribute');

    // Verify it's the right frame by checking its URL
    const url = await frame.url();
    assert.ok(url.includes('/frame2'), `Nested frame URL should contain /frame2, got ${url}`);
  });

  test('frame() returns null for non-existent frame', async () => {
    const vibe = await bro.page();
    await vibe.go(baseUrl + '/');
    await vibe.wait(500);

    const frame = await vibe.frame('does-not-exist');
    assert.strictEqual(frame, null, 'Should return null for non-existent frame');
  });

  test('frame has full Page API (eval, find, title, url)', async () => {
    const vibe = await bro.page();
    await vibe.go(baseUrl + '/');
    await vibe.wait(500);

    const frame = await vibe.frame('/frame1');
    assert.ok(frame, 'Should find frame1');

    // eval() works on frame
    const title = await frame.title();
    assert.strictEqual(title, 'Frame 1', `Frame title should be "Frame 1", got "${title}"`);

    // url() works on frame
    const url = await frame.url();
    assert.ok(url.includes('/frame1'), `Frame URL should contain /frame1, got ${url}`);

    // find() works on frame
    const h1 = await frame.find('h1');
    const text = await h1.text();
    assert.strictEqual(text, 'Frame 1 Content', `h1 text should be "Frame 1 Content", got "${text}"`);
  });

  test('frames() includes nested frames (recursive)', async () => {
    const vibe = await bro.page();
    await vibe.go(baseUrl + '/');
    await vibe.wait(500);

    const frames = await vibe.frames();
    // Should have at least 2 frames: frame1 and frame2 (nested)
    assert.ok(frames.length >= 2, `Should have at least 2 frames (recursive), got ${frames.length}`);
  });

  test('mainFrame() returns the page itself', async () => {
    const vibe = await bro.page();
    const mainFrame = vibe.mainFrame();
    assert.strictEqual(mainFrame, vibe, 'mainFrame() should return the same page instance');
    assert.strictEqual(mainFrame.id, vibe.id, 'mainFrame should have the same context ID');
  });
});
