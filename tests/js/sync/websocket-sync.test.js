/**
 * JS Sync Tests: WebSocket Monitoring — onWebSocket
 * Tests page.onWebSocket(), WebSocketInfoSync methods, and removeAllListeners('websocket').
 *
 * The WS echo server runs inside the forked sync-test-server.js child process.
 */

const { test, describe, before, after } = require('node:test');
const assert = require('node:assert');
const { fork } = require('child_process');
const path = require('path');

const { browser } = require('../../../clients/javascript/dist/sync');

// --- Server child process ---

let serverProcess;
let baseURL;
let wsURL;

before(async () => {
  serverProcess = fork(path.join(__dirname, 'sync-test-server.js'), [], { silent: true });

  // Read both HTTP base URL (line 1) and WS URL (line 2)
  const urls = await new Promise((resolve, reject) => {
    let data = '';
    serverProcess.stdout.on('data', (chunk) => {
      data += chunk.toString();
      const lines = data.trim().split('\n');
      if (lines.length >= 2 && lines[0].startsWith('http://') && lines[1].startsWith('ws://')) {
        resolve(lines);
      }
    });
    serverProcess.on('error', reject);
    setTimeout(() => reject(new Error('Server startup timeout')), 5000);
  });

  baseURL = urls[0];
  wsURL = urls[1];
});

after(() => {
  if (serverProcess) serverProcess.kill();
});

// --- Tests ---

describe('Sync API: onWebSocket', () => {
  let bro;
  before(() => { bro = browser.launch({ headless: true }); });
  after(() => { bro.close(); });

  test('onWebSocket fires when page creates a WebSocket', () => {
    const vibe = bro.newPage();
    vibe.go(`${baseURL}/ws-page`);

    let wsCreated = false;
    vibe.onWebSocket(() => {
      wsCreated = true;
    });

    vibe.wait(200);
    vibe.eval(`window.createWS('${wsURL}')`);
    vibe.wait(1000);

    assert.ok(wsCreated, 'onWebSocket should have fired');
  });

  test('ws.url() returns the correct URL', () => {
    const vibe = bro.newPage();
    vibe.go(`${baseURL}/ws-page`);

    let capturedUrl = '';
    vibe.onWebSocket((ws) => {
      capturedUrl = ws.url();
    });

    vibe.wait(200);
    vibe.eval(`window.createWS('${wsURL}')`);
    vibe.wait(1000);

    assert.strictEqual(capturedUrl, wsURL);
  });

  test('ws.onMessage() captures sent messages', () => {
    const vibe = bro.newPage();
    vibe.go(`${baseURL}/ws-page`);

    const messages = [];
    vibe.onWebSocket((ws) => {
      ws.onMessage((data, info) => {
        messages.push({ data, direction: info.direction });
      });
    });

    vibe.wait(200);
    vibe.eval(`
      const ws = window.createWS('${wsURL}');
      ws.onopen = () => ws.send('hello');
    `);
    vibe.wait(1500);

    const sent = messages.filter(m => m.direction === 'sent');
    assert.ok(sent.length > 0, `Should have captured sent messages, got: ${JSON.stringify(messages)}`);
    assert.strictEqual(sent[0].data, 'hello');
  });

  test('ws.onMessage() captures received messages', () => {
    const vibe = bro.newPage();
    vibe.go(`${baseURL}/ws-page`);

    const messages = [];
    vibe.onWebSocket((ws) => {
      ws.onMessage((data, info) => {
        messages.push({ data, direction: info.direction });
      });
    });

    vibe.wait(200);
    vibe.eval(`
      const ws = window.createWS('${wsURL}');
      ws.onopen = () => ws.send('echo-me');
    `);
    vibe.wait(1500);

    const received = messages.filter(m => m.direction === 'received');
    assert.ok(received.length > 0, `Should have captured received messages, got: ${JSON.stringify(messages)}`);
    assert.strictEqual(received[0].data, 'echo-me');
  });

  test('ws.onClose() fires when connection closes', () => {
    const vibe = bro.newPage();
    vibe.go(`${baseURL}/ws-page`);

    let closeFired = false;
    let closeCode;
    vibe.onWebSocket((ws) => {
      ws.onClose((code) => {
        closeFired = true;
        closeCode = code;
      });
    });

    vibe.wait(200);
    vibe.eval(`
      const ws = window.createWS('${wsURL}');
      ws.onopen = () => ws.close(1000, 'done');
    `);
    vibe.wait(1000);

    assert.ok(closeFired, 'onClose should have fired');
    assert.strictEqual(closeCode, 1000);
  });

  test('ws.isClosed() returns true after close', () => {
    const vibe = bro.newPage();
    vibe.go(`${baseURL}/ws-page`);

    let wsInfo;
    vibe.onWebSocket((ws) => {
      wsInfo = ws;
      ws.onClose(() => {});
    });

    vibe.wait(200);
    vibe.eval(`
      const ws = window.createWS('${wsURL}');
      ws.onopen = () => ws.close();
    `);
    vibe.wait(1000);

    assert.ok(wsInfo, 'Should have captured a WebSocket');
    assert.strictEqual(wsInfo.isClosed(), true);
  });

  test('removeAllListeners("websocket") stops onWebSocket callbacks', () => {
    const vibe = bro.newPage();
    vibe.go(`${baseURL}/ws-page`);

    let wsCount = 0;
    vibe.onWebSocket(() => {
      wsCount++;
    });

    vibe.wait(200);
    vibe.eval(`window.createWS('${wsURL}')`);
    vibe.wait(1000);
    assert.strictEqual(wsCount, 1);

    vibe.removeAllListeners('websocket');

    vibe.eval(`window.createWS('${wsURL}')`);
    vibe.wait(1000);
    assert.strictEqual(wsCount, 1, 'Should still be 1 after removing listeners');
  });
});
