/**
 * CLI Tests: Element Finding, Click, and Type
 * Tests the vibium binary directly
 */

const { test, describe, before, after } = require('node:test');
const assert = require('node:assert');
const { execSync, spawn } = require('node:child_process');
const path = require('path');
const { VIBIUM } = require('../helpers');

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
});

after(() => {
  if (serverProcess) serverProcess.kill();
});

describe('CLI: Elements', () => {
  test('find command locates element and returns @ref', () => {
    const result = execSync(`${VIBIUM} find https://example.com "a"`, {
      encoding: 'utf-8',
      timeout: 30000,
    });
    assert.match(result, /@e1/, 'Should return @e1 ref');
    assert.match(result, /\[a\]/, 'Should show [a] tag label');
    // Link text may be "More information..." or "Learn more" depending on page version
    assert.match(result, /(More information|Learn more)/i, 'Should show link text');
  });

  test('click command navigates via link', () => {
    const result = execSync(`${VIBIUM} click https://example.com "a"`, {
      encoding: 'utf-8',
      timeout: 30000,
    });
    assert.match(result, /iana\.org/i, 'Should navigate to IANA after clicking link');
  });

  test('type command enters text into input', () => {
    const result = execSync(
      `${VIBIUM} type ${baseURL}/inputs "input" "12345"`,
      {
        encoding: 'utf-8',
        timeout: 30000,
      }
    );
    assert.match(result, /12345/, 'Should show typed text in result');
  });
});
