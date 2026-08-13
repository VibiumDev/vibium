/**
 * CLI Tests: Element Finding, Click, and Type
 * Tests the vibium binary directly
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
});

after(() => {
  if (serverProcess) serverProcess.kill();
});

describe('CLI: Elements', () => {
  test('find command locates element and returns @ref', () => {
    const result = execSync(`${VIBIUM} find ${baseURL}/example "a"`, {
      encoding: 'utf-8',
      timeout: 30000,
    });
    assert.match(result, /@e1/, 'Should return @e1 ref');
    assert.match(result, /\[a\]/, 'Should show [a] tag label');
    // Link text may be "More information..." or "Learn more" depending on page version
    assert.match(result, /(More information|Learn more)/i, 'Should show link text');
  });

  test('find xpath subcommand locates element', () => {
    execSync(`${VIBIUM} go ${baseURL}/inputs`, { encoding: 'utf-8', timeout: 30000 });
    const result = execSync(`${VIBIUM} find xpath "//input"`, {
      encoding: 'utf-8',
      timeout: 30000,
    });
    assert.match(result, /@e\d+/, 'Should return an element ref');
    assert.match(result, /\[input/, 'Should show [input] tag label');
  });

  test('click command navigates via link', () => {
    const result = execSync(`${VIBIUM} click ${baseURL}/example "a"`, {
      encoding: 'utf-8',
      timeout: 30000,
    });
    assert.match(result, /clicked/i, 'Should confirm element was clicked');
  });

  test('type command enters text into input', () => {
    const result = execSync(
      `${VIBIUM} type ${baseURL}/inputs "input" "12345"`,
      {
        encoding: 'utf-8',
        timeout: 30000,
      }
    );
    assert.match(result, /typed/i, 'Should confirm text was typed');
  });

  test('type enters text containing a carriage return (#61)', () => {
    execSync(`${VIBIUM} go ${baseURL}/inputs`, { encoding: 'utf-8', timeout: 30000 });
    execSync(`${VIBIUM} type "#multiline" "$(printf '123\\r456')"`, {
      encoding: 'utf-8',
      timeout: 30000,
      shell: '/bin/bash',
    });

    const value = execSync(`${VIBIUM} eval "document.querySelector('#multiline').value"`, {
      encoding: 'utf-8',
      timeout: 30000,
    }).trim();
    assert.strictEqual(value, '123\n456');
  });

  test('type enters text containing a newline (#61)', () => {
    // A literal newline is not a key. It used to be sent as a key value the
    // browser ignored, so "123\n456" arrived as "123456".
    execSync(`${VIBIUM} go ${baseURL}/inputs`, { encoding: 'utf-8', timeout: 30000 });
    execSync(`${VIBIUM} type "#multiline" "123
456"`, { encoding: 'utf-8', timeout: 30000 });

    const value = execSync(`${VIBIUM} eval "document.querySelector('#multiline').value"`, {
      encoding: 'utf-8',
      timeout: 30000,
    }).trim();
    assert.strictEqual(value, '123\n456');
  });

});

describe('CLI: shadow DOM pierce', () => {
  test('>> crosses one shadow boundary, >>> crosses nested ones (#118)', () => {
    execSync(`${VIBIUM} go ${baseURL}/shadow`, { encoding: 'utf-8', timeout: 30000 });

    // Plain CSS cannot cross the boundary — the baseline the issue documents.
    assert.throws(
      () => execSync(`${VIBIUM} find "my-card p" --timeout 2s`, {
        encoding: 'utf-8', timeout: 30000, stdio: 'pipe',
      }),
      /not found/,
    );

    assert.match(execSync(`${VIBIUM} find "my-card >> p"`, { encoding: 'utf-8', timeout: 30000 }), /shadow text/);
    assert.match(execSync(`${VIBIUM} find "my-card >>> #deep"`, { encoding: 'utf-8', timeout: 30000 }), /Deep Button/);
  });

  test('actions work on a pierced element', () => {
    execSync(`${VIBIUM} go ${baseURL}/shadow`, { encoding: 'utf-8', timeout: 30000 });

    execSync(`${VIBIUM} fill "my-card >> #i" hello`, { encoding: 'utf-8', timeout: 30000 });
    execSync(`${VIBIUM} click "my-card >> #b"`, { encoding: 'utf-8', timeout: 30000 });

    // The component's own handler read the value, so fill and click both landed
    // inside the shadow root — including the elementFromPoint hit test.
    const out = execSync(`${VIBIUM} text "#out"`, { encoding: 'utf-8', timeout: 30000 });
    assert.match(out, /clicked:hello/);
  });

  test('map lists elements inside shadow roots (#203)', () => {
    execSync(`${VIBIUM} go ${baseURL}/shadow`, { encoding: 'utf-8', timeout: 30000 });
    const out = execSync(`${VIBIUM} map`, { encoding: 'utf-8', timeout: 30000 });
    assert.match(out, /Shadow Button/, 'shadow-root elements should be mapped');
    assert.match(out, /Deep Button/, 'nested shadow roots too');
  });
});
