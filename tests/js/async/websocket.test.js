/**
 * JS Library Tests: WebSocket Monitoring
 * Tests page.onWebSocket(), WebSocketInfo.url/onMessage/onClose/isClosed,
 * and removeAllListeners('websocket').
 *
 * Uses a WS echo server (ws library) + HTTP server.
 */

const { test, describe, before, after } = require('node:test');
const assert = require('node:assert');
const http = require('http');
const { WebSocketServer } = require('ws');

const { browser } = require('../../../clients/javascript/dist');

// --- Local test servers ---

let httpServer;
let wsServer;
let baseURL;
let wsURL;

before(async () => {
  httpServer = http.createServer((req, res) => {
    res.writeHead(200, { 'Content-Type': 'text/html' });
    res.end(`<html><head><title>WS Test</title></head><body>
      <script>
        window.createWS = function(url) {
          return new WebSocket(url);
        };
      </script>
    </body></html>`);
  });

  await new Promise((resolve) => {
    httpServer.listen(0, '127.0.0.1', () => {
      const { port } = httpServer.address();
      baseURL = `http://127.0.0.1:${port}`;
      resolve();
    });
  });

  // WebSocket echo server
  wsServer = new WebSocketServer({ port: 0, host: '127.0.0.1' });
  wsServer.on('connection', (ws) => {
    ws.on('message', (data) => {
      ws.send(data.toString());
    });
  });

  await new Promise((resolve) => {
    wsServer.on('listening', () => {
      const addr = wsServer.address();
      wsURL = `ws://127.0.0.1:${addr.port}`;
      resolve();
    });
  });
});

after(() => {
  if (wsServer) wsServer.close();
  if (httpServer) httpServer.close();
});

// --- WebSocket Monitoring ---

describe('WebSocket Monitoring: page.onWebSocket', () => {
  test('onWebSocket fires when page creates a WebSocket', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(baseURL);

      let wsCreated = false;
      vibe.onWebSocket(() => {
        wsCreated = true;
      });

      await vibe.wait(200);
      await vibe.eval(`window.createWS('${wsURL}')`);
      await vibe.wait(500);

      assert.ok(wsCreated, 'onWebSocket should have fired');
    } finally {
      await bro.close();
    }
  });

  test('ws.url() returns the correct URL', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(baseURL);

      let capturedUrl = '';
      vibe.onWebSocket((ws) => {
        capturedUrl = ws.url();
      });

      await vibe.wait(200);
      await vibe.eval(`window.createWS('${wsURL}')`);
      await vibe.wait(500);

      assert.strictEqual(capturedUrl, wsURL);
    } finally {
      await bro.close();
    }
  });

  test('ws.onMessage() captures sent messages (direction: sent)', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(baseURL);

      const messages = [];
      vibe.onWebSocket((ws) => {
        ws.onMessage((data, info) => {
          messages.push({ data, direction: info.direction });
        });
      });

      await vibe.wait(200);

      // Create WS and send a message (fire-and-forget, no Promise)
      await vibe.eval(`
        const ws = window.createWS('${wsURL}');
        ws.onopen = () => ws.send('hello');
      `);
      await vibe.wait(1000);

      const sent = messages.filter(m => m.direction === 'sent');
      assert.ok(sent.length > 0, `Should have captured sent messages, got: ${JSON.stringify(messages)}`);
      assert.strictEqual(sent[0].data, 'hello');
    } finally {
      await bro.close();
    }
  });

  test('ws.onMessage() captures received messages (direction: received)', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(baseURL);

      const messages = [];
      vibe.onWebSocket((ws) => {
        ws.onMessage((data, info) => {
          messages.push({ data, direction: info.direction });
        });
      });

      await vibe.wait(200);

      // Create WS and send — echo server echoes back
      await vibe.eval(`
        const ws = window.createWS('${wsURL}');
        ws.onopen = () => ws.send('echo-me');
      `);
      await vibe.wait(1000);

      const received = messages.filter(m => m.direction === 'received');
      assert.ok(received.length > 0, `Should have captured received messages, got: ${JSON.stringify(messages)}`);
      assert.strictEqual(received[0].data, 'echo-me');
    } finally {
      await bro.close();
    }
  });

  test('ws.onClose() fires when connection closes', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(baseURL);

      let closeFired = false;
      let closeCode;
      vibe.onWebSocket((ws) => {
        ws.onClose((code) => {
          closeFired = true;
          closeCode = code;
        });
      });

      await vibe.wait(200);

      await vibe.eval(`
        const ws = window.createWS('${wsURL}');
        ws.onopen = () => ws.close(1000, 'done');
      `);
      await vibe.wait(500);

      assert.ok(closeFired, 'onClose should have fired');
      assert.strictEqual(closeCode, 1000);
    } finally {
      await bro.close();
    }
  });

  test('ws.isClosed() returns true after close', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(baseURL);

      let wsInfo;
      vibe.onWebSocket((ws) => {
        wsInfo = ws;
        ws.onClose(() => {});
      });

      await vibe.wait(200);
      await vibe.eval(`
        const ws = window.createWS('${wsURL}');
        ws.onopen = () => ws.close();
      `);
      await vibe.wait(500);

      assert.ok(wsInfo, 'Should have captured a WebSocket');
      assert.strictEqual(wsInfo.isClosed(), true);
    } finally {
      await bro.close();
    }
  });

  test('monitoring survives page navigation (preload script persists)', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(baseURL);

      let wsCount = 0;
      vibe.onWebSocket(() => {
        wsCount++;
      });

      await vibe.wait(200);

      // Create WS on first page
      await vibe.eval(`window.createWS('${wsURL}')`);
      await vibe.wait(500);
      assert.strictEqual(wsCount, 1, 'Should have captured 1 WS on first page');

      // Navigate to a new page
      await vibe.go(baseURL);
      await vibe.wait(200);

      // Create WS on second page — preload script should still be active
      await vibe.eval(`window.createWS('${wsURL}')`);
      await vibe.wait(500);
      assert.strictEqual(wsCount, 2, 'Should have captured 2 WS total after navigation');
    } finally {
      await bro.close();
    }
  });

  test("removeAllListeners('websocket') clears callbacks", async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(baseURL);

      let wsCount = 0;
      vibe.onWebSocket(() => {
        wsCount++;
      });

      await vibe.wait(200);
      await vibe.eval(`window.createWS('${wsURL}')`);
      await vibe.wait(500);
      assert.strictEqual(wsCount, 1);

      // Remove listeners
      vibe.removeAllListeners('websocket');

      // Create another WS — should not fire callback
      await vibe.eval(`window.createWS('${wsURL}')`);
      await vibe.wait(500);
      assert.strictEqual(wsCount, 1, 'Should still be 1 after removing listeners');
    } finally {
      await bro.close();
    }
  });
});
