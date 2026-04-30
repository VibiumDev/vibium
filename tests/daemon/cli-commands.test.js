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
  const execOpts = {
    encoding: 'utf-8',
    timeout: opts.timeout || 60000,
    env: { ...process.env, ...opts.env },
  };
  if (opts.cwd) execOpts.cwd = opts.cwd;
  const result = execSync(`${VIBIUM} ${args}`, execOpts);
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
    clicker('daemon start --headless');
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
    clicker('daemon start --headless');
    clicker('go https://example.com');
  });

  after(() => {
    stopDaemon();
  });

  test('is visible returns true for visible element', () => {
    const result = clickerJSON('is visible "h1"');
    assert.strictEqual(result.ok, true);
    assert.strictEqual(result.result, 'true');
  });

  test('is visible returns false for non-existent element', () => {
    const result = clickerJSON('is visible "#does-not-exist"');
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
    clicker('daemon start --headless');
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

  test('find role finds element by role and returns @ref', () => {
    const result = clickerJSON('find role heading');
    assert.strictEqual(result.ok, true);
    assert.ok(result.result.includes('@e1'), 'Should return @e1 ref');
    assert.ok(result.result.includes('[h1]'), 'Should find heading element');
  });

  test('find role finds element by role and name', () => {
    const result = clickerJSON('find role link --name "Learn more"');
    assert.strictEqual(result.ok, true);
    assert.ok(result.result.includes('@e1'), 'Should return @e1 ref');
    assert.ok(result.result.includes('[a]'), 'Should find link element');
  });
});

describe('Daemon CLI: Waiting commands', () => {
  before(() => {
    stopDaemon();
    clicker('daemon start --headless');
    clicker('go https://example.com');
  });

  after(() => {
    stopDaemon();
  });

  test('wait load succeeds on loaded page', () => {
    const result = clickerJSON('wait load');
    assert.strictEqual(result.ok, true);
    assert.ok(result.result.includes('complete'), 'Should report page loaded');
  });

  test('wait url matches current URL', () => {
    const result = clickerJSON('wait url "example.com"');
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
    clicker('daemon start --headless');
  });

  after(() => {
    stopDaemon();
  });

  test('scroll into-view scrolls element into view', () => {
    clicker('go https://example.com');
    const result = clickerJSON('scroll into-view "a"');
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
    clicker('daemon start --headless');
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

describe('Daemon CLI: Screenshot honors --output path', () => {
  const tmpDir = fs.mkdtempSync(path.join(require('node:os').tmpdir(), 'vibium-shot-'));
  const savedPaths = [];

  before(() => {
    stopDaemon();
    clicker('daemon start --headless');
    clicker('go https://example.com');
  });

  after(() => {
    stopDaemon();
    for (const p of savedPaths) {
      try { fs.unlinkSync(p); } catch (e) {}
    }
    try { fs.rmSync(tmpDir, { recursive: true, force: true }); } catch (e) {}
  });

  test('absolute path is honored as written', () => {
    const target = path.join(tmpDir, 'abs.png');
    const result = clickerJSON(`screenshot -o ${target}`);
    assert.strictEqual(result.ok, true);
    assert.ok(result.result.includes(target), `Should report exact path, got: ${result.result}`);
    assert.ok(fs.existsSync(target), `File should exist at ${target}`);
    assert.ok(fs.statSync(target).size > 0, 'File should not be empty');
    savedPaths.push(target);
  });

  test('relative path with ./ prefix lands in CWD, not screenshot dir', () => {
    const cwd = fs.mkdtempSync(path.join(require('node:os').tmpdir(), 'vibium-cwd-'));
    const result = clickerJSON('screenshot -o ./rel.png', { cwd });
    assert.strictEqual(result.ok, true);
    const expected = path.join(cwd, 'rel.png');
    assert.ok(fs.existsSync(expected), `File should exist at ${expected}`);
    assert.ok(!result.result.includes(path.join('Pictures', 'Vibium')), `Relative path should not land in default screenshot dir, got ${result.result}`);
    savedPaths.push(expected);
    try { fs.rmSync(cwd, { recursive: true, force: true }); } catch (e) {}
  });

  test('subdir relative path creates parent and saves there', () => {
    const target = path.join(tmpDir, 'nested', 'sub.png');
    const result = clickerJSON(`screenshot -o ${target}`);
    assert.strictEqual(result.ok, true);
    assert.ok(fs.existsSync(target), `File should exist at ${target}`);
    savedPaths.push(target);
  });

  test('bare basename still goes to default screenshot dir', () => {
    const result = clickerJSON('screenshot -o bare-basename-test.png');
    assert.strictEqual(result.ok, true);
    const match = result.result.match(/Screenshot saved to (.+)/);
    assert.ok(match, 'Should report save path');
    const saved = match[1].trim();
    assert.ok(path.basename(saved) === 'bare-basename-test.png', 'Basename should be preserved');
    assert.ok(saved.includes(path.join('Pictures', 'Vibium')) || saved.includes('Vibium'), `Bare basename should land in default screenshot dir, got ${saved}`);
    savedPaths.push(saved);
  });
});

describe('Daemon CLI: quit command', () => {
  before(() => {
    stopDaemon();
    clicker('daemon start --headless');
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
