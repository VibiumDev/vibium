/**
 * CLI Tests: Page Context Tracking
 * Verifies that page switch and page new correctly track the active page
 * so subsequent commands target the right context.
 */

const { test, describe, before, after } = require("../../helpers/capabilities").suite("core");
const assert = require('node:assert');
const { execSync, spawn } = require('node:child_process');
const path = require('path');
const { VIBIUM } = require("../../helpers");

let serverProcess, baseURL;

before(async () => {
  serverProcess = spawn('node', [path.join(__dirname, '../../helpers/test-server.js')], {
    stdio: ['pipe', 'pipe', 'pipe'],
  });
  baseURL = await new Promise((resolve) => {
    serverProcess.stdout.once('data', (data) => {
      resolve(data.toString().trim());
    });
  });

  // Navigate page 0 to the home page
  execSync(`${VIBIUM} go ${baseURL}/`, { encoding: 'utf-8', timeout: 30000 });
});

after(() => {
  // Close any extra pages created during tests (switch to 1 and close, if it exists)
  try {
    execSync(`${VIBIUM} page switch 1`, { encoding: 'utf-8', timeout: 10000 });
    execSync(`${VIBIUM} page close`, { encoding: 'utf-8', timeout: 10000 });
  } catch {
    // ignore — page may not exist
  }
  if (serverProcess) serverProcess.kill();
});

describe('CLI: Page Context Tracking', () => {
  test('page switch targets correct page for subsequent commands', () => {
    // Page 0 is already on the home page (title: "The Internet")
    // Create page 1 and navigate to login page
    execSync(`${VIBIUM} page new ${baseURL}/login`, {
      encoding: 'utf-8',
      timeout: 30000,
    });

    // page new should have switched to the new page — verify title
    const loginTitle = execSync(`${VIBIUM} title`, {
      encoding: 'utf-8',
      timeout: 30000,
    }).trim();
    assert.match(loginTitle, /Login/, 'New page should show login page title');

    // Switch back to page 0 and verify title
    execSync(`${VIBIUM} page switch 0`, { encoding: 'utf-8', timeout: 30000 });
    const homeTitle = execSync(`${VIBIUM} title`, {
      encoding: 'utf-8',
      timeout: 30000,
    }).trim();
    assert.match(homeTitle, /The Internet/, 'Page 0 should show home page title');

    // Switch to page 1 and verify title
    execSync(`${VIBIUM} page switch 1`, { encoding: 'utf-8', timeout: 30000 });
    const loginTitle2 = execSync(`${VIBIUM} title`, {
      encoding: 'utf-8',
      timeout: 30000,
    }).trim();
    assert.match(loginTitle2, /Login/, 'Page 1 should show login page title');

    // Cleanup: close page 1
    execSync(`${VIBIUM} page close`, { encoding: 'utf-8', timeout: 30000 });
  });

  test('page new switches to the new page', () => {
    // Page 0 is on the home page
    const homeTitleBefore = execSync(`${VIBIUM} title`, {
      encoding: 'utf-8',
      timeout: 30000,
    }).trim();
    assert.match(homeTitleBefore, /The Internet/, 'Should start on home page');

    // Open a new page with the login page
    execSync(`${VIBIUM} page new ${baseURL}/login`, {
      encoding: 'utf-8',
      timeout: 30000,
    });

    // Title should now be the login page (we're on the new page)
    const titleAfter = execSync(`${VIBIUM} title`, {
      encoding: 'utf-8',
      timeout: 30000,
    }).trim();
    assert.match(titleAfter, /Login/, 'Should be on the new page after page new');

    // Cleanup: close the new page
    execSync(`${VIBIUM} page close`, { encoding: 'utf-8', timeout: 30000 });
  });

  test('page close without index closes the active page', () => {
    // Page 0 is on the home page
    execSync(`${VIBIUM} page new ${baseURL}/login`, {
      encoding: 'utf-8',
      timeout: 30000,
    });

    // We're now on page 1 (login page). Close without index — should close page 1.
    const result = execSync(`${VIBIUM} page close`, {
      encoding: 'utf-8',
      timeout: 30000,
    });
    assert.match(result, /closed/i, 'Should confirm page closed');

    // Only page 0 should remain, and it should be the home page
    const title = execSync(`${VIBIUM} title`, {
      encoding: 'utf-8',
      timeout: 30000,
    }).trim();
    assert.match(title, /The Internet/, 'Remaining page should be home page');

    const pages = execSync(`${VIBIUM} pages`, {
      encoding: 'utf-8',
      timeout: 30000,
    });
    // Should only have one page
    assert.ok(!pages.includes('[1]'), 'Should only have one page remaining');
  });
});

describe('CLI: active context reaches script-backed commands', () => {
  // The commands above only exercised `title`, which routes through
  // newSession(). eval/find/map called the BiDi client with an empty context,
  // which resolves to "the first context" — so they silently ignored both page
  // switches and frame switches (#205).

  test('eval and find follow a page switch', () => {
    execSync(`${VIBIUM} go ${baseURL}/`, { encoding: 'utf-8', timeout: 30000 });
    execSync(`${VIBIUM} page new ${baseURL}/login`, { encoding: 'utf-8', timeout: 30000 });

    const onNewPage = execSync(`${VIBIUM} eval "location.pathname"`, {
      encoding: 'utf-8',
      timeout: 30000,
    }).trim();
    assert.match(onNewPage, /\/login/, 'eval should run on the newly opened page');

    execSync(`${VIBIUM} page switch 0`, { encoding: 'utf-8', timeout: 30000 });
    const backOnHome = execSync(`${VIBIUM} eval "location.pathname"`, {
      encoding: 'utf-8',
      timeout: 30000,
    }).trim();
    assert.strictEqual(backOnHome, '/', 'eval should follow the switch back to page 0');

    execSync(`${VIBIUM} page switch 1`, { encoding: 'utf-8', timeout: 30000 });
    const found = execSync(`${VIBIUM} find h2`, { encoding: 'utf-8', timeout: 30000 });
    assert.match(found, /Login/, 'find should search the active page');

    execSync(`${VIBIUM} page close`, { encoding: 'utf-8', timeout: 30000 });
  });

  test('eval follows a frame switch and persists to the next command (#205)', () => {
    execSync(`${VIBIUM} go ${baseURL}/frames`, { encoding: 'utf-8', timeout: 30000 });

    const outer = execSync(`${VIBIUM} eval "document.querySelector('h1').id"`, {
      encoding: 'utf-8',
      timeout: 30000,
    }).trim();
    assert.strictEqual(outer, 'outer', 'setup: should start on the top-level document');

    execSync(`${VIBIUM} frame myframe`, { encoding: 'utf-8', timeout: 30000 });

    // Separate process — the frame only persists if the daemon recorded it.
    const inner = execSync(`${VIBIUM} eval "document.querySelector('h1').id"`, {
      encoding: 'utf-8',
      timeout: 30000,
    }).trim();
    assert.strictEqual(inner, 'inner', 'eval should run inside the switched frame');
  });
});
