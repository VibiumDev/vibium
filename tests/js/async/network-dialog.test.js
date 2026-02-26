/**
 * JS Library Tests: Network Interception & Dialogs
 * Tests page.route, route.fulfill/continue/abort, page.onRequest/onResponse,
 * page.capture.request/response, page.onDialog, dialog.accept/dismiss.
 *
 * Uses a local HTTP server — no external network dependencies.
 */

const { test, describe, before, after } = require('node:test');
const assert = require('node:assert');
const http = require('http');

const { browser } = require('../../../clients/javascript/dist');

// --- Local test server ---

let server;
let baseURL;

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
});

after(() => {
  if (server) server.close();
});

// --- Network Interception ---

describe('Network Interception: page.route', () => {
  test('route.abort() blocks a request', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();

      // Block all .png requests
      await vibe.route('**/*.png', (route) => {
        route.abort();
      });

      await vibe.go(baseURL);

      // Verify the page loaded (route didn't break navigation)
      const title = await vibe.title();
      assert.strictEqual(title, 'Test Page');
    } finally {
      await bro.close();
    }
  });

  test('route.fulfill() returns a mock response', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(baseURL);

      await vibe.route('**/json', (route) => {
        route.fulfill({
          status: 200,
          body: JSON.stringify({ mocked: true }),
          contentType: 'application/json',
        });
      });

      const result = await vibe.eval(`
        fetch('${baseURL}/json')
          .then(r => r.json())
      `);

      assert.deepStrictEqual(result, { mocked: true });
    } finally {
      await bro.close();
    }
  });

  test('route.fulfill() with custom headers', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(baseURL);

      await vibe.route('**/text', (route) => {
        route.fulfill({
          status: 201,
          headers: { 'X-Custom': 'test-value', 'Content-Type': 'text/plain' },
          body: 'custom body',
        });
      });

      const result = await vibe.eval(`
        fetch('${baseURL}/text')
          .then(r => r.text().then(body => ({ status: r.status, body, custom: r.headers.get('X-Custom') })))
      `);

      assert.strictEqual(result.status, 201);
      assert.strictEqual(result.body, 'custom body');
      assert.strictEqual(result.custom, 'test-value');
    } finally {
      await bro.close();
    }
  });

  test('route.continue() lets request through', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(baseURL);

      let intercepted = false;
      await vibe.route('**', (route) => {
        intercepted = true;
        route.continue();
      });

      // Fetch triggers the intercept
      await vibe.eval(`fetch('${baseURL}/text')`);
      await vibe.wait(200);

      assert.ok(intercepted, 'Route handler should have been called');
    } finally {
      await bro.close();
    }
  });

  test('unroute() removes a route', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(baseURL);

      let callCount = 0;
      await vibe.route('**/text', (route) => {
        callCount++;
        route.continue();
      });

      // First fetch — should be intercepted
      await vibe.eval(`fetch('${baseURL}/text')`);
      await vibe.wait(200);
      assert.ok(callCount > 0, 'Route handler should have been called');

      const countBefore = callCount;
      await vibe.unroute('**/text');

      // Second fetch — should NOT be intercepted
      await vibe.eval(`fetch('${baseURL}/text')`);
      await vibe.wait(200);
      assert.strictEqual(callCount, countBefore, 'Route should not fire after unroute');
    } finally {
      await bro.close();
    }
  });
});

// --- Network Events & Waiters ---

describe('Network Events: onRequest/onResponse', () => {
  test('onRequest() fires for page navigation', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();

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
    } finally {
      await bro.close();
    }
  });

  test('onResponse() fires for page navigation', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();

      const statuses = [];
      vibe.onResponse((resp) => {
        statuses.push(resp.status());
      });

      await vibe.go(baseURL);
      await vibe.wait(200);

      assert.ok(statuses.length > 0, 'Should have captured at least one response');
      assert.ok(statuses.includes(200), `Should have a 200 response, got: ${statuses.join(', ')}`);
    } finally {
      await bro.close();
    }
  });

  test('request.method() and request.headers() work', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();

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
    } finally {
      await bro.close();
    }
  });

  test('response.url() and response.status() work', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(baseURL);

      const responsePromise = vibe.capture.response('**/json');
      await vibe.eval(`fetch('${baseURL}/json')`);
      const resp = await responsePromise;

      assert.ok(resp.url().includes('/json'));
      assert.strictEqual(resp.status(), 200);
      assert.ok(typeof resp.headers() === 'object');
    } finally {
      await bro.close();
    }
  });
});

describe('Network Waiters: capture.request/capture.response', () => {
  test('capture.response() resolves on matching response', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(baseURL);

      const responsePromise = vibe.capture.response('**/json');
      await vibe.eval(`fetch('${baseURL}/json')`);

      const resp = await responsePromise;
      assert.ok(resp.url().includes('/json'), `Response URL should include /json, got: ${resp.url()}`);
      assert.strictEqual(resp.status(), 200);
    } finally {
      await bro.close();
    }
  });

  test('capture.request() resolves on matching request', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(baseURL);

      const requestPromise = vibe.capture.request('**/text');
      await vibe.eval(`fetch('${baseURL}/text')`);

      const req = await requestPromise;
      assert.ok(req.url().includes('/text'), `Request URL should include /text, got: ${req.url()}`);
      assert.strictEqual(req.method(), 'GET');
    } finally {
      await bro.close();
    }
  });
});

// --- Response Body ---

describe('Response Body: response.body() and response.json()', () => {
  test('response.body() returns text content via onResponse', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(baseURL);

      let captured = null;
      vibe.onResponse((resp) => {
        if (resp.url().includes('/text')) {
          captured = resp;
        }
      });

      await vibe.eval(`fetch('${baseURL}/text')`);
      await vibe.wait(500);

      assert.ok(captured, 'Should have captured the /text response');
      const body = await captured.body();
      assert.strictEqual(body, 'hello world');
    } finally {
      await bro.close();
    }
  });

  test('response.json() parses JSON content via onResponse', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(baseURL);

      let captured = null;
      vibe.onResponse((resp) => {
        if (resp.url().includes('/json')) {
          captured = resp;
        }
      });

      await vibe.eval(`fetch('${baseURL}/json')`);
      await vibe.wait(500);

      assert.ok(captured, 'Should have captured the /json response');
      const data = await captured.json();
      assert.deepStrictEqual(data, { name: 'vibium', version: 1 });
    } finally {
      await bro.close();
    }
  });

  test('response.body() works with capture.response', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(baseURL);

      const responsePromise = vibe.capture.response('**/text');
      await vibe.eval(`fetch('${baseURL}/text')`);
      const resp = await responsePromise;

      const body = await resp.body();
      assert.strictEqual(body, 'hello world');
    } finally {
      await bro.close();
    }
  });

  test('response.json() works with capture.response', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(baseURL);

      const responsePromise = vibe.capture.response('**/json');
      await vibe.eval(`fetch('${baseURL}/json')`);
      const resp = await responsePromise;

      const data = await resp.json();
      assert.deepStrictEqual(data, { name: 'vibium', version: 1 });
    } finally {
      await bro.close();
    }
  });
});

// --- Dialogs ---

describe('Dialogs: page.onDialog', () => {
  test('onDialog() handles alert', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(baseURL);

      let dialogMessage = '';
      let dialogType = '';
      vibe.onDialog((dialog) => {
        dialogMessage = dialog.message();
        dialogType = dialog.type();
        dialog.accept();
      });

      await vibe.eval('alert("Hello from test")');

      assert.strictEqual(dialogMessage, 'Hello from test');
      assert.strictEqual(dialogType, 'alert');
    } finally {
      await bro.close();
    }
  });

  test('onDialog() handles confirm with accept', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(baseURL);

      vibe.onDialog((dialog) => {
        dialog.accept();
      });

      const result = await vibe.eval('confirm("Are you sure?")');
      assert.strictEqual(result, true);
    } finally {
      await bro.close();
    }
  });

  test('onDialog() handles confirm with dismiss', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(baseURL);

      vibe.onDialog((dialog) => {
        dialog.dismiss();
      });

      const result = await vibe.eval('confirm("Are you sure?")');
      assert.strictEqual(result, false);
    } finally {
      await bro.close();
    }
  });

  test('onDialog() handles prompt with text', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(baseURL);

      vibe.onDialog((dialog) => {
        assert.strictEqual(dialog.type(), 'prompt');
        dialog.accept('my answer');
      });

      const result = await vibe.eval('prompt("Enter name:")');
      assert.strictEqual(result, 'my answer');
    } finally {
      await bro.close();
    }
  });

  test('dialogs are auto-dismissed when no handler registered', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(baseURL);

      // No onDialog handler — should auto-dismiss
      const result = await vibe.eval('confirm("Auto dismiss?")');
      assert.strictEqual(result, false);
    } finally {
      await bro.close();
    }
  });
});

// --- Capture Navigation ---

describe('Capture: navigation', () => {
  test('capture.navigation() resolves with URL on link click', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(`${baseURL}/nav-test`);

      const link = await vibe.find('#link');
      const url = await vibe.capture.navigation(async () => {
        await link.click();
      });

      assert.ok(url.includes('/page2'), `Navigation URL should include /page2, got: ${url}`);
    } finally {
      await bro.close();
    }
  });
});

// --- Capture Download ---

describe('Capture: download', () => {
  test('capture.download() resolves with Download object', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(`${baseURL}/download`);

      const link = await vibe.find('#download-link');
      const download = await vibe.capture.download(async () => {
        await link.click();
      });

      assert.ok(download, 'Should resolve with a Download object');
      assert.ok(download.url().includes('/download-file'), `Download URL should include /download-file, got: ${download.url()}`);
      assert.strictEqual(download.suggestedFilename(), 'test.txt');
    } finally {
      await bro.close();
    }
  });
});

// --- Capture Dialog ---

describe('Capture: dialog', () => {
  test('capture.dialog() resolves with Dialog object', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(baseURL);

      // Use setTimeout because alert() blocks eval — the dialog must fire
      // asynchronously so capture.dialog can capture it.
      await vibe.eval('setTimeout(() => alert("Hello from expect"), 50)');
      const dialog = await vibe.capture.dialog();

      assert.ok(dialog, 'Should resolve with a Dialog object');
      assert.strictEqual(dialog.type(), 'alert');
      assert.strictEqual(dialog.message(), 'Hello from expect');
      await dialog.accept();
    } finally {
      await bro.close();
    }
  });
});

// --- Capture Event ---

describe('Capture: event', () => {
  test('capture.event("response") resolves on fetch', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(baseURL);

      const result = await vibe.capture.event('response', async () => {
        await vibe.eval(`fetch('${baseURL}/json')`);
      });

      assert.ok(result, 'Should resolve with event data');
      assert.ok(typeof result.url === 'function', 'Should be a Response object with url()');
    } finally {
      await bro.close();
    }
  });
});

// --- WebSocket Stubs ---

describe('Stubs: WebSocket methods', () => {
  test('routeWebSocket() throws not implemented', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();

      assert.throws(
        () => vibe.routeWebSocket('**', () => {}),
        /Not implemented/
      );
    } finally {
      await bro.close();
    }
  });
});

// --- Checkpoint ---

describe('Network & Dialog Checkpoint', () => {
  test('route.continue, onResponse, and onDialog work together', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();

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

      await vibe.eval('alert("checkpoint")');
      assert.ok(dialogHandled);
    } finally {
      await bro.close();
    }
  });
});
