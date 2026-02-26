/**
 * Daemon CLI Commands Tests
 * Tests all new CLI commands that require daemon mode.
 */

const { test, describe, before, after } = require('node:test');
const assert = require('node:assert');
const { execSync } = require('node:child_process');
const fs = require('node:fs');
const path = require('node:path');
const { VIBIUM } = require('../helpers');

// Helper to run clicker and return trimmed output
function clicker(args, opts = {}) {
  const result = execSync(`${VIBIUM} ${args}`, {
    encoding: 'utf-8',
    timeout: opts.timeout || 60000,
    env: { ...process.env, ...opts.env },
  });
  return result.trim();
}

function clickerJSON(args, opts = {}) {
  const result = clicker(`${args} --json`, opts);
  return JSON.parse(result);
}

// Helper to stop daemon (ignore errors if not running)
function stopDaemon() {
  try {
    execSync(`${VIBIUM} daemon stop`, { encoding: 'utf-8', timeout: 10000 });
  } catch (e) {
    // Daemon may not be running
  }
}

describe('Daemon CLI: Navigation commands', () => {
  before(() => {
    stopDaemon();
    clicker('daemon start -d --headless');
    clicker('go https://example.com');
  });

  after(() => {
    stopDaemon();
  });

  test('back navigates back in history', () => {
    // Navigate to a second page
    clickerJSON('go https://example.com/');
    const result = clickerJSON('back');
    assert.strictEqual(result.ok, true, 'back should succeed');
  });

  test('forward navigates forward in history', () => {
    const result = clickerJSON('forward');
    assert.strictEqual(result.ok, true, 'forward should succeed');
  });

  test('reload reloads the page', () => {
    const result = clickerJSON('reload');
    assert.strictEqual(result.ok, true, 'reload should succeed');
    assert.ok(result.result.includes('reload'), 'Should confirm page reloaded');
  });
});

describe('Daemon CLI: Element state commands', () => {
  before(() => {
    stopDaemon();
    clicker('daemon start -d --headless');
    clicker('go https://example.com');
  });

  after(() => {
    stopDaemon();
  });

  test('is-visible returns true for visible element', () => {
    const result = clickerJSON('is-visible "h1"');
    assert.strictEqual(result.ok, true);
    assert.strictEqual(result.result, 'true');
  });

  test('is-visible returns false for non-existent element', () => {
    const result = clickerJSON('is-visible "#does-not-exist"');
    assert.strictEqual(result.ok, true);
    assert.strictEqual(result.result, 'false');
  });

  test('attr gets element attribute', () => {
    const result = clickerJSON('attr "a" "href"');
    assert.strictEqual(result.ok, true);
    assert.ok(result.result.includes('iana.org'), 'Should return href value');
  });
});

describe('Daemon CLI: Accessibility and search commands', () => {
  before(() => {
    stopDaemon();
    clicker('daemon start -d --headless');
    clicker('go https://example.com');
  });

  after(() => {
    stopDaemon();
  });

  test('a11y-tree returns accessibility tree', () => {
    const result = clickerJSON('a11y-tree');
    assert.strictEqual(result.ok, true);
    assert.ok(result.result.includes('WebArea'), 'Should contain WebArea root');
  });

  test('a11y-tree --everything includes more nodes', () => {
    const result = clickerJSON('a11y-tree --everything');
    assert.strictEqual(result.ok, true);
    assert.ok(result.result.includes('WebArea'), 'Should contain WebArea root');
  });

  test('find --role finds element by role and returns @ref', () => {
    const result = clickerJSON('find --role heading');
    assert.strictEqual(result.ok, true);
    assert.ok(result.result.includes('@e1'), 'Should return @e1 ref');
    assert.ok(result.result.includes('[h1]'), 'Should find heading element');
  });

  test('find --role finds element by role and text', () => {
    const result = clickerJSON('find --role link --text "Learn more"');
    assert.strictEqual(result.ok, true);
    assert.ok(result.result.includes('@e1'), 'Should return @e1 ref');
    assert.ok(result.result.includes('[a]'), 'Should find link element');
  });
});

describe('Daemon CLI: Waiting commands', () => {
  before(() => {
    stopDaemon();
    clicker('daemon start -d --headless');
    clicker('go https://example.com');
  });

  after(() => {
    stopDaemon();
  });

  test('wait-for-load succeeds on loaded page', () => {
    const result = clickerJSON('wait-for-load');
    assert.strictEqual(result.ok, true);
    assert.ok(result.result.includes('complete'), 'Should report page loaded');
  });

  test('wait-for-url matches current URL', () => {
    const result = clickerJSON('wait-for-url "example.com"');
    assert.strictEqual(result.ok, true);
    assert.ok(result.result.includes('example.com'), 'Should match URL pattern');
  });

  test('sleep pauses execution', () => {
    const start = Date.now();
    const result = clickerJSON('sleep 200');
    const elapsed = Date.now() - start;
    assert.strictEqual(result.ok, true);
    assert.ok(elapsed >= 150, `Should have waited ~200ms (actual: ${elapsed}ms)`);
  });
});

describe('Daemon CLI: Interaction commands', () => {
  before(() => {
    stopDaemon();
    clicker('daemon start -d --headless');
  });

  after(() => {
    stopDaemon();
  });

  test('scroll-into-view scrolls element into view', () => {
    clicker('go https://example.com');
    const result = clickerJSON('scroll-into-view "a"');
    assert.strictEqual(result.ok, true);
    assert.ok(result.result.includes('Scrolled'), 'Should confirm scroll');
  });

  test('press sends key to focused element', () => {
    clicker('go https://example.com');
    const result = clickerJSON('press Tab');
    assert.strictEqual(result.ok, true);
    assert.ok(result.result.includes('Pressed'), 'Should confirm key press');
  });

  test('fill and value work together on input', () => {
    // Navigate to a page with an input via eval
    clicker('go https://example.com');
    clicker('eval "document.body.innerHTML = \'<input id=test type=text>\';"');

    // Fill the input
    const fillResult = clickerJSON('fill "#test" "hello world"');
    assert.strictEqual(fillResult.ok, true);

    // Read the value back
    const valueResult = clickerJSON('value "#test"');
    assert.strictEqual(valueResult.ok, true);
    assert.strictEqual(valueResult.result, 'hello world');
  });

  test('check and uncheck toggle checkbox', () => {
    clicker('go https://example.com');
    clicker('eval "document.body.innerHTML = \'<input id=cb type=checkbox>\';"');

    // Check
    const checkResult = clickerJSON('check "#cb"');
    assert.strictEqual(checkResult.ok, true);

    // Uncheck
    const uncheckResult = clickerJSON('uncheck "#cb"');
    assert.strictEqual(uncheckResult.ok, true);
  });
});

describe('Daemon CLI: Screenshot --full-page', () => {
  let savedPath = null;

  before(() => {
    stopDaemon();
    clicker('daemon start -d --headless');
    clicker('go https://example.com');
  });

  after(() => {
    stopDaemon();
    // Clean up screenshot file
    if (savedPath) {
      try { fs.unlinkSync(savedPath); } catch (e) {}
    }
  });

  test('screenshot --full-page captures full page', () => {
    const result = clickerJSON('screenshot -o test-cli-fullpage.png --full-page');
    assert.strictEqual(result.ok, true);
    assert.ok(result.result.includes('test-cli-fullpage.png'), 'Should save to specified file');
    // Extract the full path from the result (daemon saves to its screenshotDir)
    const match = result.result.match(/Screenshot saved to (.+)/);
    assert.ok(match, 'Should report save path');
    savedPath = match[1];
    assert.ok(fs.existsSync(savedPath), `File should exist at ${savedPath}`);
    const stats = fs.statSync(savedPath);
    assert.ok(stats.size > 0, 'File should not be empty');
  });
});

describe('Daemon CLI: quit command', () => {
  before(() => {
    stopDaemon();
    clicker('daemon start -d --headless');
    clicker('go https://example.com');
  });

  after(() => {
    stopDaemon();
  });

  test('quit closes browser session', () => {
    const result = clickerJSON('quit');
    assert.strictEqual(result.ok, true);
    assert.ok(result.result.includes('closed'), 'Should confirm browser closed');
  });
});
