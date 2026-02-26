/**
 * JS Library Tests: Console & Error Events
 * Tests page.onConsole, page.onError, ConsoleMessage class, and removeAllListeners.
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
    res.writeHead(200, { 'Content-Type': 'text/html' });
    res.end('<html><head><title>Console Test</title></head><body>Hello</body></html>');
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

// --- Console Events ---

describe('Console Events: page.onConsole', () => {
  test('onConsole() captures console.log', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(baseURL);

      const messages = [];
      vibe.onConsole((msg) => messages.push(msg));

      await vibe.eval('console.log("hello from test")');
      await vibe.wait(300);

      assert.ok(messages.length >= 1, `Expected at least 1 console message, got ${messages.length}`);
      const msg = messages.find(m => m.text().includes('hello from test'));
      assert.ok(msg, 'Should find console.log message');
      assert.strictEqual(msg.type(), 'log');
    } finally {
      await bro.close();
    }
  });

  test('onConsole() captures console.warn', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(baseURL);

      const messages = [];
      vibe.onConsole((msg) => messages.push(msg));

      await vibe.eval('console.warn("warning msg")');
      await vibe.wait(300);

      const msg = messages.find(m => m.text().includes('warning msg'));
      assert.ok(msg, 'Should find console.warn message');
      assert.strictEqual(msg.type(), 'warn');
    } finally {
      await bro.close();
    }
  });

  test('onConsole() captures console.error (not onError)', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(baseURL);

      const consoleMessages = [];
      const errors = [];
      vibe.onConsole((msg) => consoleMessages.push(msg));
      vibe.onError((err) => errors.push(err));

      await vibe.eval('console.error("console err")');
      await vibe.wait(300);

      // console.error should fire onConsole
      const msg = consoleMessages.find(m => m.text().includes('console err'));
      assert.ok(msg, 'console.error should fire onConsole');
      assert.strictEqual(msg.type(), 'error');

      // console.error should NOT fire onError
      const matchingError = errors.find(e => e.message.includes('console err'));
      assert.ok(!matchingError, 'console.error should NOT fire onError');
    } finally {
      await bro.close();
    }
  });

  test('ConsoleMessage.args() returns serialized arguments', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(baseURL);

      const messages = [];
      vibe.onConsole((msg) => messages.push(msg));

      await vibe.eval('console.log("arg1", 42)');
      await vibe.wait(300);

      const msg = messages.find(m => m.text().includes('arg1'));
      assert.ok(msg, 'Should find console.log message');
      const args = msg.args();
      assert.ok(Array.isArray(args), 'args() should return an array');
      assert.ok(args.length >= 2, `Expected at least 2 args, got ${args.length}`);
    } finally {
      await bro.close();
    }
  });
});

// --- Error Events ---

describe('Error Events: page.onError', () => {
  test('onError() captures uncaught exception', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(baseURL);

      const errors = [];
      vibe.onError((err) => errors.push(err));

      // Use setTimeout to create a truly uncaught exception (not caught by eval's promise)
      await vibe.eval('setTimeout(() => { throw new Error("boom uncaught") }, 0)');
      await vibe.wait(500);

      assert.ok(errors.length >= 1, `Expected at least 1 error, got ${errors.length}`);
      const err = errors.find(e => e.message.includes('boom uncaught'));
      assert.ok(err, 'Should capture uncaught exception');
      assert.ok(err instanceof Error, 'Should be an Error instance');
    } finally {
      await bro.close();
    }
  });

  test('onError() does NOT fire for console.error', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(baseURL);

      const errors = [];
      vibe.onError((err) => errors.push(err));

      await vibe.eval('console.error("just a console error")');
      await vibe.wait(300);

      const matchingError = errors.find(e => e.message.includes('just a console error'));
      assert.ok(!matchingError, 'onError should NOT fire for console.error');
    } finally {
      await bro.close();
    }
  });
});

// --- Collect Mode ---

describe('Collect Mode: onConsole("collect") + consoleMessages()', () => {
  test('onConsole("collect") + consoleMessages() captures console.log', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(baseURL);

      vibe.onConsole('collect');
      await vibe.eval('console.log("collect hello")');
      await vibe.wait(300);

      const messages = vibe.consoleMessages();
      const match = messages.find(m => m.text.includes('collect hello'));
      assert.ok(match, `Should find console.log message, got: ${JSON.stringify(messages)}`);
      assert.strictEqual(match.type, 'log');
    } finally {
      await bro.close();
    }
  });

  test('onConsole("collect") + consoleMessages() captures console.warn', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(baseURL);

      vibe.onConsole('collect');
      await vibe.eval('console.warn("collect warning")');
      await vibe.wait(300);

      const messages = vibe.consoleMessages();
      const match = messages.find(m => m.text.includes('collect warning'));
      assert.ok(match, `Should find console.warn message, got: ${JSON.stringify(messages)}`);
      assert.strictEqual(match.type, 'warn');
    } finally {
      await bro.close();
    }
  });

  test('consoleMessages() clears buffer after retrieval', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(baseURL);

      vibe.onConsole('collect');
      await vibe.eval('console.log("first")');
      await vibe.wait(300);

      const first = vibe.consoleMessages();
      assert.ok(first.length >= 1, 'Should have messages');

      const second = vibe.consoleMessages();
      assert.strictEqual(second.length, 0, 'Buffer should be empty after retrieval');
    } finally {
      await bro.close();
    }
  });

  test('consoleMessages() returns [] when not collecting', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(baseURL);

      const messages = vibe.consoleMessages();
      assert.deepStrictEqual(messages, []);
    } finally {
      await bro.close();
    }
  });
});

describe('Collect Mode: onError("collect") + errors()', () => {
  test('onError("collect") + errors() captures uncaught exception', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(baseURL);

      vibe.onError('collect');
      await vibe.eval('setTimeout(() => { throw new Error("collect boom") }, 0)');
      await vibe.wait(500);

      const errs = vibe.errors();
      const match = errs.find(e => e.message.includes('collect boom'));
      assert.ok(match, `Should capture uncaught exception, got: ${JSON.stringify(errs)}`);
    } finally {
      await bro.close();
    }
  });

  test('errors() clears buffer after retrieval', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(baseURL);

      vibe.onError('collect');
      await vibe.eval('setTimeout(() => { throw new Error("err1") }, 0)');
      await vibe.wait(500);

      const first = vibe.errors();
      assert.ok(first.length >= 1, 'Should have errors');

      const second = vibe.errors();
      assert.strictEqual(second.length, 0, 'Buffer should be empty after retrieval');
    } finally {
      await bro.close();
    }
  });

  test('errors() returns [] when not collecting', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(baseURL);

      const errs = vibe.errors();
      assert.deepStrictEqual(errs, []);
    } finally {
      await bro.close();
    }
  });
});

// --- removeAllListeners ---

describe('removeAllListeners for console/error', () => {
  test('removeAllListeners("console") clears console callbacks', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(baseURL);

      const messages = [];
      vibe.onConsole((msg) => messages.push(msg));

      await vibe.eval('console.log("before clear")');
      await vibe.wait(300);
      assert.ok(messages.length >= 1, 'Should have captured message before clear');

      vibe.removeAllListeners('console');

      const countBefore = messages.length;
      await vibe.eval('console.log("after clear")');
      await vibe.wait(300);
      assert.strictEqual(messages.length, countBefore, 'Should not capture messages after removeAllListeners');
    } finally {
      await bro.close();
    }
  });

  test('removeAllListeners("error") clears error callbacks', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(baseURL);

      const errors = [];
      vibe.onError((err) => errors.push(err));

      vibe.removeAllListeners('error');

      await vibe.eval('setTimeout(() => { throw new Error("should not capture") }, 0)');
      await vibe.wait(500);

      const matching = errors.find(e => e.message.includes('should not capture'));
      assert.ok(!matching, 'Should not capture errors after removeAllListeners');
    } finally {
      await bro.close();
    }
  });

  test('removeAllListeners("console") stops collect mode', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(baseURL);

      vibe.onConsole('collect');
      await vibe.eval('console.log("before remove")');
      await vibe.wait(300);

      const msgsBefore = vibe.consoleMessages();
      assert.ok(msgsBefore.length >= 1, 'Should have captured message before clear');

      vibe.removeAllListeners('console');
      await vibe.eval('console.log("after remove")');
      await vibe.wait(300);

      const msgsAfter = vibe.consoleMessages();
      assert.deepStrictEqual(msgsAfter, [], 'Should return [] after removeAllListeners clears collect mode');
    } finally {
      await bro.close();
    }
  });

  test('removeAllListeners("error") stops collect mode', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(baseURL);

      vibe.onError('collect');
      vibe.removeAllListeners('error');

      await vibe.eval('setTimeout(() => { throw new Error("should not collect") }, 0)');
      await vibe.wait(500);

      const errs = vibe.errors();
      assert.deepStrictEqual(errs, [], 'Should return [] after removeAllListeners clears collect mode');
    } finally {
      await bro.close();
    }
  });
});
