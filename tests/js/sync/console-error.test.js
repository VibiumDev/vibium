/**
 * JS Library Tests: Sync Console & Error Collection
 * Tests page.onConsole('collect'), consoleMessages(), onError('collect'),
 * errors(), and removeAllListeners() using the sync API.
 *
 * Uses sync-test-server.js in a child process (sync API blocks the main thread).
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
      const line = data.trim();
      if (line.startsWith('http://')) resolve(line);
    });
    serverProcess.on('error', reject);
    setTimeout(() => reject(new Error('Server startup timeout')), 5000);
  });
});

after(() => {
  if (serverProcess) serverProcess.kill();
});

// --- Console Collection ---

describe('Sync API: Console collection', () => {
  test('onConsole("collect") + consoleMessages() captures console.log', () => {
    const bro = browser.launch({ headless: true });
    try {
      const vibe = bro.page();
      vibe.go(baseURL);

      vibe.onConsole('collect');
      vibe.evaluate('console.log("sync hello")');
      vibe.wait(300);

      const messages = vibe.consoleMessages();
      const match = messages.find(m => m.text.includes('sync hello'));
      assert.ok(match, `Should find console.log message, got: ${JSON.stringify(messages)}`);
      assert.strictEqual(match.type, 'log');
    } finally {
      bro.close();
    }
  });

  test('onConsole("collect") + consoleMessages() captures console.warn', () => {
    const bro = browser.launch({ headless: true });
    try {
      const vibe = bro.page();
      vibe.go(baseURL);

      vibe.onConsole('collect');
      vibe.evaluate('console.warn("sync warning")');
      vibe.wait(300);

      const messages = vibe.consoleMessages();
      const match = messages.find(m => m.text.includes('sync warning'));
      assert.ok(match, `Should find console.warn message, got: ${JSON.stringify(messages)}`);
      assert.strictEqual(match.type, 'warn');
    } finally {
      bro.close();
    }
  });
});

// --- Error Collection ---

describe('Sync API: Error collection', () => {
  test('onError("collect") + errors() captures uncaught exception', () => {
    const bro = browser.launch({ headless: true });
    try {
      const vibe = bro.page();
      vibe.go(baseURL);

      vibe.onError('collect');
      vibe.evaluate('setTimeout(() => { throw new Error("sync boom") }, 0)');
      vibe.wait(500);

      const errs = vibe.errors();
      const match = errs.find(e => e.message.includes('sync boom'));
      assert.ok(match, `Should capture uncaught exception, got: ${JSON.stringify(errs)}`);
    } finally {
      bro.close();
    }
  });
});

// --- removeAllListeners ---

describe('Sync API: removeAllListeners for console/error', () => {
  test('removeAllListeners("console") stops collection', () => {
    const bro = browser.launch({ headless: true });
    try {
      const vibe = bro.page();
      vibe.go(baseURL);

      vibe.onConsole('collect');
      vibe.evaluate('console.log("before clear")');
      vibe.wait(300);

      const msgsBefore = vibe.consoleMessages();
      assert.ok(msgsBefore.length >= 1, 'Should have captured message before clear');

      vibe.removeAllListeners('console');
      vibe.evaluate('console.log("after clear")');
      vibe.wait(300);

      const msgsAfter = vibe.consoleMessages();
      const afterMsg = msgsAfter.find(m => m.text.includes('after clear'));
      assert.ok(!afterMsg, 'Should not capture messages after removeAllListeners');
    } finally {
      bro.close();
    }
  });

  test('removeAllListeners("error") stops collection', () => {
    const bro = browser.launch({ headless: true });
    try {
      const vibe = bro.page();
      vibe.go(baseURL);

      vibe.onError('collect');
      vibe.removeAllListeners('error');

      vibe.evaluate('setTimeout(() => { throw new Error("should not capture") }, 0)');
      vibe.wait(500);

      const errs = vibe.errors();
      const match = errs.find(e => e.message.includes('should not capture'));
      assert.ok(!match, 'Should not capture errors after removeAllListeners');
    } finally {
      bro.close();
    }
  });
});
