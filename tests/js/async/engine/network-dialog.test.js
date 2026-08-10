/**
 * JS Library Tests: Network Interception & Dialogs
 * Tests page.route, route.fulfill/continue/abort, page.onRequest/onResponse,
 * page.capture.request/response, page.onDialog, dialog.accept/dismiss.
 *
 * Uses a local HTTP server — no external network dependencies.
 * Each test uses bro.newPage() for route/dialog handler isolation.
 */

const { test, describe, before, after } = require("../../../helpers/capabilities").suite("core");
const assert = require('node:assert');
const http = require('http');

const { browser } = require('../../../../clients/javascript/dist');

// --- Local test server ---

let server;
let baseURL;
let bro;

before(async () => {
  server = http.createServer((req, res) => {
    if (req.url === '/json') {
      res.writeHead(200, { 'Content-Type': 'application/json' });
      res.end(JSON.stringify({ name: 'vibium', version: 1 }));
    } else if (req.url === '/text') {
      res.writeHead(200, { 'Content-Type': 'text/plain' });
      res.end('hello world');
    } else if (req.url === '/page2') {
      res.writeHead(200, { 'Content-Type': 'text/html' });
      res.end('<html><head><title>Page 2</title></head><body><h1>Page 2</h1></body></html>');
    } else if (req.url === '/nav-test') {
      res.writeHead(200, { 'Content-Type': 'text/html' });
      res.end(`<html><head><title>Nav Test</title></head><body><a id="link" href="/page2">Go to page 2</a></body></html>`);
    } else if (req.url === '/download-file') {
      res.writeHead(200, {
        'Content-Type': 'application/octet-stream',
        'Content-Disposition': 'attachment; filename="test.txt"',
      });
      res.end('download content');
    } else if (req.url === '/download') {
      res.writeHead(200, { 'Content-Type': 'text/html' });
      res.end('<html><head><title>Download</title></head><body><a href="/download-file" id="download-link" download="test.txt">Download</a></body></html>');
    } else if (req.url === '/api/echo') {
      // Echoes the received body so tests can assert what actually arrived.
      let body = '';
      req.on('data', (chunk) => { body += chunk; });
      req.on('end', () => {
        res.writeHead(200, { 'Content-Type': 'application/json' });
        res.end(JSON.stringify({ method: req.method, body }));
      });
    } else {
      res.writeHead(200, { 'Content-Type': 'text/html' });
      res.end('<html><head><title>Test Page</title></head><body>Test Content</body></html>');
    }
  });

  await new Promise((resolve) => {
    server.listen(0, '127.0.0.1', () => {
      const { port } = server.address();
      baseURL = `http://127.0.0.1:${port}`;
      resolve();
    });
  });

  bro = await browser.start({ headless: true });
});

after(async () => {
  await bro.stop();
  if (server) server.close();
});

// --- Network Interception ---

describe.requires('network')('Network Interception: page.route', () => {
  test('route.abort() blocks a request', async () => {
    const vibe = await bro.newPage();

    // Block all .png requests
    await vibe.route('**/*.png', (route) => {
      route.abort();
    });

    await vibe.go(baseURL);

    // Verify the page loaded (route didn't break navigation)
    const title = await vibe.title();
    assert.strictEqual(title, 'Test Page');
    await vibe.close();
  });

  test('a route that matches nothing does not block navigation (#128)', async () => {
    const vibe = await bro.newPage();

    // #128 reported that registering any route deadlocked the next go()
    // permanently, and specifically that it happened even when the pattern
    // could never match. The existing route tests all use patterns that do
    // match, so this is the case they miss.
    await vibe.route('**/*.jpg', (route) => route.continue());

    await vibe.go(baseURL);
    assert.strictEqual(await vibe.title(), 'Test Page', 'navigation should complete');

    // Still responsive afterwards, not wedged behind the registration.
    assert.strictEqual(await vibe.evaluate('1 + 1'), 2);
    await vibe.close();
  });

  test('setHeaders() does not block navigation (#128)', async () => {
    const vibe = await bro.newPage();

    await vibe.setHeaders({ 'X-Vibium-Test': '128' });

    await vibe.go(baseURL);
    assert.strictEqual(await vibe.title(), 'Test Page', 'navigation should complete');
    await vibe.close();
  });

  test('route.fulfill() returns a mock response', async () => {
    const vibe = await bro.newPage();
    await vibe.go(baseURL);

    await vibe.route('**/json', (route) => {
      route.fulfill({
        status: 200,
        body: JSON.stringify({ mocked: true }),
        contentType: 'application/json',
      });
    });

    const result = await vibe.evaluate(`
      fetch('${baseURL}/json')
        .then(r => r.json())
    `);

    assert.deepStrictEqual(result, { mocked: true });
    await vibe.close();
  });

  test('route.fulfill() with custom headers', async () => {
    const vibe = await bro.newPage();
    await vibe.go(baseURL);

    await vibe.route('**/text', (route) => {
      route.fulfill({
        status: 201,
        headers: { 'X-Custom': 'test-value', 'Content-Type': 'text/plain' },
        body: 'custom body',
      });
    });

    const result = await vibe.evaluate(`
      fetch('${baseURL}/text')
        .then(r => r.text().then(body => ({ status: r.status, body, custom: r.headers.get('X-Custom') })))
    `);

    assert.strictEqual(result.status, 201);
    assert.strictEqual(result.body, 'custom body');
    assert.strictEqual(result.custom, 'test-value');
    await vibe.close();
  });

  test('route.continue() lets request through', async () => {
    const vibe = await bro.newPage();
    await vibe.go(baseURL);

    let intercepted = false;
    await vibe.route('**', (route) => {
      intercepted = true;
      route.continue();
    });

    // Fetch triggers the intercept
    await vibe.evaluate(`fetch('${baseURL}/text')`);
    await vibe.wait(200);

    assert.ok(intercepted, 'Route handler should have been called');
    await vibe.close();
  });

  test('route.continue() sends an overridden postData to the server', async () => {
    const vibe = await bro.newPage();
    await vibe.go(baseURL);

    await vibe.route('**/api/echo', (route) => {
      route.continue({ postData: 'replaced-body' });
    });

    const result = await vibe.evaluate(`
      fetch('${baseURL}/api/echo', { method: 'POST', body: 'original-body' })
        .then(r => r.json())
    `);

    assert.strictEqual(result.method, 'POST');
    assert.strictEqual(result.body, 'replaced-body');
    await vibe.close();
  });

  test('unroute() removes a route', async () => {
    const vibe = await bro.newPage();
    await vibe.go(baseURL);

    let callCount = 0;
    await vibe.route('**/text', (route) => {
      callCount++;
      route.continue();
    });

    // First fetch — should be intercepted
    await vibe.evaluate(`fetch('${baseURL}/text')`);
    await vibe.wait(200);
    assert.ok(callCount > 0, 'Route handler should have been called');

    const countBefore = callCount;
    await vibe.unroute('**/text');

    // Second fetch — should NOT be intercepted
    await vibe.evaluate(`fetch('${baseURL}/text')`);
    await vibe.wait(200);
    assert.strictEqual(callCount, countBefore, 'Route should not fire after unroute');
    await vibe.close();
  });
});

// --- Network Events & Waiters ---

describe.requires('network')('Network Events: onRequest/onResponse', () => {
  test('onRequest() fires for page navigation', async () => {
    const vibe = await bro.newPage();

    const urls = [];
    vibe.onRequest((req) => {
      urls.push(req.url());
    });

    await vibe.go(baseURL);
    await vibe.wait(200);

    assert.ok(urls.length > 0, 'Should have captured at least one request');
    assert.ok(
      urls.some(u => u.includes('127.0.0.1')),
      `Should have a request to local server, got: ${urls.join(', ')}`
    );
    await vibe.close();
  });

  test('onResponse() fires for page navigation', async () => {
    const vibe = await bro.newPage();

    const statuses = [];
    vibe.onResponse((resp) => {
      statuses.push(resp.status());
    });

    await vibe.go(baseURL);
    await vibe.wait(200);

    assert.ok(statuses.length > 0, 'Should have captured at least one response');
    assert.ok(statuses.includes(200), `Should have a 200 response, got: ${statuses.join(', ')}`);
    await vibe.close();
  });

  test('request.method() and request.headers() work', async () => {
    const vibe = await bro.newPage();

    let capturedMethod = '';
    let capturedHeaders = {};
    vibe.onRequest((req) => {
      if (req.url().includes('127.0.0.1') && !capturedMethod) {
        capturedMethod = req.method();
        capturedHeaders = req.headers();
      }
    });

    await vibe.go(baseURL);
    await vibe.wait(200);

    assert.strictEqual(capturedMethod, 'GET');
    assert.ok(typeof capturedHeaders === 'object');
    await vibe.close();
  });

  test('response.url() and response.status() work', async () => {
    const vibe = await bro.newPage();
    await vibe.go(baseURL);

    const responsePromise = vibe.capture.response('**/json');
    await vibe.evaluate(`fetch('${baseURL}/json')`);
    const resp = await responsePromise;

    assert.ok(resp.url().includes('/json'));
    assert.strictEqual(resp.status(), 200);
    assert.ok(typeof resp.headers() === 'object');
    await vibe.close();
  });
});

describe.requires('network')('Network Waiters: capture.request/capture.response', () => {
  test('capture.response() resolves on matching response', async () => {
    const vibe = await bro.newPage();
    await vibe.go(baseURL);

    const responsePromise = vibe.capture.response('**/json');
    await vibe.evaluate(`fetch('${baseURL}/json')`);

    const resp = await responsePromise;
    assert.ok(resp.url().includes('/json'), `Response URL should include /json, got: ${resp.url()}`);
    assert.strictEqual(resp.status(), 200);
    await vibe.close();
  });

  test('capture.request() resolves on matching request', async () => {
    const vibe = await bro.newPage();
    await vibe.go(baseURL);

    const requestPromise = vibe.capture.request('**/text');
    await vibe.evaluate(`fetch('${baseURL}/text')`);

    const req = await requestPromise;
    assert.ok(req.url().includes('/text'), `Request URL should include /text, got: ${req.url()}`);
    assert.strictEqual(req.method(), 'GET');
    await vibe.close();
  });
});

// --- Response Body ---

describe.requires('network')('Response Body: response.body() and response.json()', () => {
  test('response.body() returns text content via onResponse', async () => {
    const vibe = await bro.newPage();
    await vibe.go(baseURL);

    let captured = null;
    vibe.onResponse((resp) => {
      if (resp.url().includes('/text')) {
        captured = resp;
      }
    });

    await vibe.evaluate(`fetch('${baseURL}/text')`);
    await vibe.wait(500);

    assert.ok(captured, 'Should have captured the /text response');
    const body = await captured.body();
    assert.strictEqual(body, 'hello world');
    await vibe.close();
  });

  test('response.json() parses JSON content via onResponse', async () => {
    const vibe = await bro.newPage();
    await vibe.go(baseURL);

    let captured = null;
    vibe.onResponse((resp) => {
      if (resp.url().includes('/json')) {
        captured = resp;
      }
    });

    await vibe.evaluate(`fetch('${baseURL}/json')`);
    await vibe.wait(500);

    assert.ok(captured, 'Should have captured the /json response');
    const data = await captured.json();
    assert.deepStrictEqual(data, { name: 'vibium', version: 1 });
    await vibe.close();
  });

  test('response.body() works with capture.response', async () => {
    const vibe = await bro.newPage();
    await vibe.go(baseURL);

    const responsePromise = vibe.capture.response('**/text');
    await vibe.evaluate(`fetch('${baseURL}/text')`);
    const resp = await responsePromise;

    const body = await resp.body();
    assert.strictEqual(body, 'hello world');
    await vibe.close();
  });

  test('response.json() works with capture.response', async () => {
    const vibe = await bro.newPage();
    await vibe.go(baseURL);

    const responsePromise = vibe.capture.response('**/json');
    await vibe.evaluate(`fetch('${baseURL}/json')`);
    const resp = await responsePromise;

    const data = await resp.json();
    assert.deepStrictEqual(data, { name: 'vibium', version: 1 });
    await vibe.close();
  });
});

// --- Dialogs ---

describe.requires('dialogs')('Dialogs: page.onDialog', () => {
  test('onDialog() handles alert', async () => {
    const vibe = await bro.newPage();
    await vibe.go(baseURL);

    let dialogMessage = '';
    let dialogType = '';
    vibe.onDialog((dialog) => {
      dialogMessage = dialog.message();
      dialogType = dialog.type();
      dialog.accept();
    });

    await vibe.evaluate('alert("Hello from test")');

    assert.strictEqual(dialogMessage, 'Hello from test');
    assert.strictEqual(dialogType, 'alert');
    await vibe.close();
  });

  test('onDialog() handles confirm with accept', async () => {
    const vibe = await bro.newPage();
    await vibe.go(baseURL);

    vibe.onDialog((dialog) => {
      dialog.accept();
    });

    const result = await vibe.evaluate('confirm("Are you sure?")');
    assert.strictEqual(result, true);
    await vibe.close();
  });

  test('onDialog() handles confirm with dismiss', async () => {
    const vibe = await bro.newPage();
    await vibe.go(baseURL);

    vibe.onDialog((dialog) => {
      dialog.dismiss();
    });

    const result = await vibe.evaluate('confirm("Are you sure?")');
    assert.strictEqual(result, false);
    await vibe.close();
  });

  test('onDialog() handles prompt with text', async () => {
    const vibe = await bro.newPage();
    await vibe.go(baseURL);

    vibe.onDialog((dialog) => {
      assert.strictEqual(dialog.type(), 'prompt');
      dialog.accept('my answer');
    });

    const result = await vibe.evaluate('prompt("Enter name:")');
    assert.strictEqual(result, 'my answer');
    await vibe.close();
  });

  test('dialogs are auto-dismissed when no handler registered', async () => {
    const vibe = await bro.newPage();
    await vibe.go(baseURL);

    // No onDialog handler — should auto-dismiss
    const result = await vibe.evaluate('confirm("Auto dismiss?")');
    assert.strictEqual(result, false);
    await vibe.close();
  });
});

// --- Capture Navigation ---

describe('Capture: navigation', () => {
  test('capture.navigation() resolves with URL on link click', async () => {
    const vibe = await bro.newPage();
    await vibe.go(`${baseURL}/nav-test`);

    const link = await vibe.find('#link');
    const url = await vibe.capture.navigation(async () => {
      await link.click();
    });

    assert.ok(url.includes('/page2'), `Navigation URL should include /page2, got: ${url}`);
    await vibe.close();
  });
});

// --- Capture Download ---

describe('Capture: download', () => {
  test('capture.download() resolves with Download object', async () => {
    const vibe = await bro.newPage();
    await vibe.go(`${baseURL}/download`);

    const link = await vibe.find('#download-link');
    const download = await vibe.capture.download(async () => {
      await link.click();
    });

    assert.ok(download, 'Should resolve with a Download object');
    assert.ok(download.url().includes('/download-file'), `Download URL should include /download-file, got: ${download.url()}`);
    assert.strictEqual(download.suggestedFilename(), 'test.txt');
    await vibe.close();
  });
});

// --- Capture Dialog ---

describe.requires('dialogs')('Capture: dialog', () => {
  test('capture.dialog() resolves with Dialog object', async () => {
    const vibe = await bro.newPage();
    await vibe.go(baseURL);

    // Use setTimeout because alert() blocks eval — the dialog must fire
    // asynchronously so capture.dialog can capture it.
    await vibe.evaluate('setTimeout(() => alert("Hello from expect"), 50)');
    const dialog = await vibe.capture.dialog();

    assert.ok(dialog, 'Should resolve with a Dialog object');
    assert.strictEqual(dialog.type(), 'alert');
    assert.strictEqual(dialog.message(), 'Hello from expect');
    await dialog.accept();
    await vibe.close();
  });
});

// --- Capture Event ---

describe.requires('network')('Capture: event', () => {
  test('capture.event("response") resolves on fetch', async () => {
    const vibe = await bro.newPage();
    await vibe.go(baseURL);

    const result = await vibe.capture.event('response', async () => {
      await vibe.evaluate(`fetch('${baseURL}/json')`);
    });

    assert.ok(result, 'Should resolve with event data');
    assert.ok(typeof result.url === 'function', 'Should be a Response object with url()');
    await vibe.close();
  });
});

// --- WebSocket Stubs ---

describe('Stubs: WebSocket methods', () => {
  test('routeWebSocket() throws not implemented', async () => {
    const vibe = await bro.newPage();

    assert.throws(
      () => vibe.routeWebSocket('**', () => {}),
      /Not implemented/
    );
    await vibe.close();
  });
});

// --- Checkpoint ---

describe.requires('network', 'dialogs')('Network & Dialog Checkpoint', () => {
  test('route.continue, onResponse, and onDialog work together', async () => {
    const vibe = await bro.newPage();

    // Set up route that intercepts and continues
    let intercepted = false;
    await vibe.route('**', (route) => {
      intercepted = true;
      route.continue();
    });

    // Track responses
    const responseUrls = [];
    vibe.onResponse((resp) => {
      responseUrls.push(resp.url());
    });

    await vibe.go(baseURL);
    await vibe.wait(200);

    assert.ok(intercepted, 'Route should have intercepted');
    assert.ok(responseUrls.length > 0, 'Should have captured responses');

    // Set up dialog handler and trigger a dialog
    let dialogHandled = false;
    vibe.onDialog((dialog) => {
      dialogHandled = true;
      dialog.accept();
    });

    await vibe.evaluate('alert("checkpoint")');
    assert.ok(dialogHandled);
    await vibe.close();
  });
});
