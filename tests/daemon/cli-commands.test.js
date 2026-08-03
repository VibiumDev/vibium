/**
 * Daemon CLI Commands Tests
 * Tests all new CLI commands that require daemon mode.
 */

const { test, describe, before, after } = require('node:test');
const assert = require('node:assert');
const { execSync, spawn } = require('node:child_process');
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
  const result = clicker(`--json ${args}`, opts);
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

let serverProcess, baseURL;

before(async () => {
  serverProcess = spawn('node', [path.join(__dirname, '../helpers/test-server.js')], {
    stdio: ['pipe', 'pipe', 'pipe'],
  });
  baseURL = await new Promise((resolve) => {
    serverProcess.stdout.once('data', (data) => {
      resolve(data.toString().trim());
    });
  });
  // One daemon for the whole file; each describe re-navigates for a clean page.
  stopDaemon();
  clicker('daemon start --headless');
});

after(() => {
  stopDaemon();
  if (serverProcess) serverProcess.kill();
});

describe('Daemon CLI: Navigation commands', () => {
  before(() => {
    clicker(`go ${baseURL}/example`);
  });

  test('back navigates back in history', () => {
    // Navigate to a second page
    clickerJSON(`go ${baseURL}/more-information`);
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
    clicker(`go ${baseURL}/example`);
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
    assert.ok(result.result.includes('/more-information'), 'Should return href value');
  });

  test('text returns textContent, not rendered text (#215)', () => {
    clicker(`content '<div id="d">vis<span style="display:none">HIDDEN</span></div>'`);

    const el = clickerJSON('text "#d"');
    assert.strictEqual(el.result, 'visHIDDEN', 'element text is textContent, so hidden text is included');

    // Page-level text is a different question and stays rendered text.
    const page = clickerJSON('text');
    assert.ok(!page.result.includes('HIDDEN'), 'page text stays innerText');
  });

  test('is actionable accepts a bare selector like its siblings (#199)', () => {
    clicker(`go ${baseURL}/example`);
    const out = clicker('is actionable "h1"');
    assert.match(out, /Visible/, `1-arg form should work, got: ${out.slice(0, 120)}`);

    const withUrl = clicker(`is actionable ${baseURL}/example "h1"`);
    assert.match(withUrl, /Visible/, 'the 2-arg form must keep working');
  });

  test('attr distinguishes a present-but-empty attribute from an absent one (#198)', () => {
    clicker(`content '<button id="b" disabled>Go</button>'`);

    // Present but valueless: empty string, and a success exit.
    const present = clickerJSON('attr "#b" "disabled"');
    assert.strictEqual(present.ok, true);
    assert.strictEqual(present.result, '', 'a valueless attribute is present with an empty value');

    // Absent: a distinct value, but still a normal result — erroring here is
    // what #153 was about.
    const absent = clickerJSON('attr "#b" "nonexistent"');
    assert.strictEqual(absent.ok, true, 'an absent attribute must not error (#153)');
    assert.strictEqual(absent.result, 'null', 'absent must be distinguishable from present-but-empty');
  });
});

describe('Daemon CLI: Accessibility and search commands', () => {
  before(() => {
    clicker(`go ${baseURL}/example`);
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
    const result = clickerJSON('find role link --name "More information"');
    assert.strictEqual(result.ok, true);
    assert.ok(result.result.includes('@e1'), 'Should return @e1 ref');
    assert.ok(result.result.includes('[a]'), 'Should find link element');
  });

  test('find role button --name matches a submit input by its value (#204)', () => {
    // <input type="submit" value="Login"> has no textContent, so --name used to
    // land on a textContent filter and poll to the 30s timeout.
    clicker(`go ${baseURL}/selectors`);
    const result = clickerJSON('find role button --name "Login"');
    assert.strictEqual(result.ok, true);
    assert.ok(result.result.includes('[input'), `Should find the submit input, got: ${result.result}`);
  });
});

describe('Daemon CLI: Waiting commands', () => {
  before(() => {
    clicker(`go ${baseURL}/example`);
  });

  test('wait load succeeds on loaded page', () => {
    const result = clickerJSON('wait load');
    assert.strictEqual(result.ok, true);
    assert.ok(result.result.includes('complete'), 'Should report page loaded');
  });

  test('wait url matches current URL', () => {
    const result = clickerJSON('wait url "/example"');
    assert.strictEqual(result.ok, true);
    assert.ok(result.result.includes('/example'), 'Should match URL pattern');
  });

  test('sleep pauses execution', () => {
    const start = Date.now();
    // sleep sets DisableFlagParsing (so negative values work), which also
    // rejects --json — call it without JSON output
    const result = clicker('sleep 200');
    const elapsed = Date.now() - start;
    assert.ok(result.includes('Slept') || result.includes('200'), 'Should confirm sleep');
    assert.ok(elapsed >= 150, `Should have waited ~200ms (actual: ${elapsed}ms)`);
  });
});

describe('Daemon CLI: Interaction commands', () => {
  test('scroll into-view scrolls element into view', () => {
    clicker(`go ${baseURL}/example`);
    const result = clickerJSON('scroll into-view "a"');
    assert.strictEqual(result.ok, true);
    assert.ok(result.result.includes('Scrolled'), 'Should confirm scroll');
  });

  test('press sends key to focused element', () => {
    clicker(`go ${baseURL}/example`);
    const result = clickerJSON('press Tab');
    assert.strictEqual(result.ok, true);
    assert.ok(result.result.includes('Pressed'), 'Should confirm key press');
  });

  test('fill and value work together on input', () => {
    // Navigate to a page with an input via eval
    clicker(`go ${baseURL}/example`);
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
    clicker(`go ${baseURL}/example`);
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
    clicker(`go ${baseURL}/example`);
  });

  after(() => {
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
    clicker(`go ${baseURL}/example`);
  });

  test('stop closes browser session', () => {
    const result = clickerJSON('stop');
    assert.strictEqual(result.ok, true);
    assert.ok(result.result.includes('closed'), 'Should confirm browser closed');
  });
});
