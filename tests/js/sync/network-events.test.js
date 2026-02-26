/**
 * JS Sync Tests: Network Events — onRequest/onResponse
 * Tests page.onRequest(), page.onResponse(), and removeAllListeners for sync API.
 */

const { test, describe, before, after } = require('node:test');
const assert = require('node:assert');
const { fork } = require('child_process');
const path = require('path');

const { browser } = require('../../../clients/javascript/dist/sync');

// --- Server child process ---

let serverProcess;
let baseURL;

before(async () => {
  serverProcess = fork(path.join(__dirname, 'sync-test-server.js'), [], { silent: true });

  baseURL = await new Promise((resolve, reject) => {
    let data = '';
    serverProcess.stdout.on('data', (chunk) => {
      data += chunk.toString();
      const line = data.trim().split('\n')[0];
      if (line.startsWith('http://')) resolve(line);
    });
    serverProcess.on('error', reject);
    setTimeout(() => reject(new Error('Server startup timeout')), 5000);
  });
});

after(() => {
  if (serverProcess) serverProcess.kill();
});

// --- Tests ---

describe('Sync API: onRequest/onResponse', () => {
  let bro;
  before(() => { bro = browser.launch({ headless: true }); });
  after(() => { bro.close(); });

  test('onRequest fires for fetch requests', () => {
    const vibe = bro.newPage();
    vibe.go(`${baseURL}/fetch`);

    const urls = [];
    vibe.onRequest((req) => {
      urls.push(req.url);
    });

    vibe.evaluate('return doFetch()');
    vibe.wait(500);
    assert.ok(urls.some(u => u.includes('/api/data')), `Should capture request to /api/data, got: ${urls.join(', ')}`);
  });

  test('onRequest captures method and headers', () => {
    const vibe = bro.newPage();
    vibe.go(`${baseURL}/fetch`);

    let capturedMethod = '';
    let capturedHeaders = {};
    vibe.onRequest((req) => {
      if (req.url.includes('/api/data')) {
        capturedMethod = req.method;
        capturedHeaders = req.headers;
      }
    });

    vibe.evaluate('return doFetch()');
    vibe.wait(500);
    assert.strictEqual(capturedMethod, 'GET');
    assert.ok(typeof capturedHeaders === 'object');
  });

  test('onResponse fires for fetch requests', () => {
    const vibe = bro.newPage();
    vibe.go(`${baseURL}/fetch`);

    const statuses = [];
    vibe.onResponse((resp) => {
      statuses.push(resp.status);
    });

    vibe.evaluate('return doFetch()');
    vibe.wait(500);
    assert.ok(statuses.includes(200), `Should capture 200 response, got: ${statuses.join(', ')}`);
  });

  test('onResponse captures url, status, headers', () => {
    const vibe = bro.newPage();
    vibe.go(`${baseURL}/fetch`);

    let capturedUrl = '';
    let capturedStatus = 0;
    let capturedHeaders = {};
    vibe.onResponse((resp) => {
      if (resp.url.includes('/api/data')) {
        capturedUrl = resp.url;
        capturedStatus = resp.status;
        capturedHeaders = resp.headers;
      }
    });

    vibe.evaluate('return doFetch()');
    vibe.wait(500);
    assert.ok(capturedUrl.includes('/api/data'));
    assert.strictEqual(capturedStatus, 200);
    assert.ok(typeof capturedHeaders === 'object');
  });

  test('removeAllListeners("request") stops onRequest callbacks', () => {
    const vibe = bro.newPage();
    vibe.go(`${baseURL}/fetch`);

    const urls = [];
    vibe.onRequest((req) => {
      urls.push(req.url);
    });

    vibe.removeAllListeners('request');
    vibe.evaluate('return doFetch()');
    vibe.wait(500);
    assert.strictEqual(urls.length, 0, 'Should not capture requests after removeAllListeners');
  });

  test('removeAllListeners("response") stops onResponse callbacks', () => {
    const vibe = bro.newPage();
    vibe.go(`${baseURL}/fetch`);

    const statuses = [];
    vibe.onResponse((resp) => {
      statuses.push(resp.status);
    });

    vibe.removeAllListeners('response');
    vibe.evaluate('return doFetch()');
    vibe.wait(500);
    assert.strictEqual(statuses.length, 0, 'Should not capture responses after removeAllListeners');
  });

  test('onRequest provides postData for POST requests', () => {
    const vibe = bro.newPage();
    vibe.go(`${baseURL}/fetch`);

    let capturedPostData = null;
    vibe.onRequest((req) => {
      if (req.url.includes('/api/echo')) {
        capturedPostData = req.postData;
      }
    });

    vibe.evaluate('return doPostFetch()');
    vibe.wait(500);
    assert.ok(capturedPostData !== null, 'Should capture postData');
    const parsed = JSON.parse(capturedPostData);
    assert.strictEqual(parsed.hello, 'world');
  });

  test('onResponse provides body', () => {
    const vibe = bro.newPage();
    vibe.go(`${baseURL}/fetch`);

    let capturedBody = null;
    vibe.onResponse((resp) => {
      if (resp.url.includes('/api/data')) {
        capturedBody = resp.body;
      }
    });

    vibe.evaluate('return doFetch()');
    vibe.wait(500);
    assert.ok(capturedBody !== null, 'Should capture body');
    const parsed = JSON.parse(capturedBody);
    assert.strictEqual(parsed.message, 'real data');
    assert.strictEqual(parsed.count, 42);
  });

  test('capture.request includes postData', () => {
    const vibe = bro.newPage();
    vibe.go(`${baseURL}/fetch`);

    const result = vibe.capture.request('**/api/echo', () => {
      vibe.evaluate('return doPostFetch()');
    });

    assert.ok(result.postData !== null && result.postData !== undefined, 'Should have postData');
    const parsed = JSON.parse(result.postData);
    assert.strictEqual(parsed.hello, 'world');
  });

  test('capture.response includes body', () => {
    const vibe = bro.newPage();
    vibe.go(`${baseURL}/fetch`);

    const result = vibe.capture.response('**/api/data', () => {
      vibe.evaluate('return doFetch()');
    });

    assert.ok(result.body !== null && result.body !== undefined, 'Should have body');
    const parsed = JSON.parse(result.body);
    assert.strictEqual(parsed.message, 'real data');
    assert.strictEqual(parsed.count, 42);
  });
});
