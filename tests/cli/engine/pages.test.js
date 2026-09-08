/**
 * CLI Tests: Page Commands
 * Tests page management via the daemon.
 */

const { test, describe } = require("../../helpers/capabilities").suite("core");
const assert = require('node:assert');
const { execSync } = require('node:child_process');
const { VIBIUM } = require("../../helpers");

describe('CLI: Page Commands', () => {
  test('pages lists open pages', () => {
    const result = execSync(`${VIBIUM} pages`, {
      encoding: 'utf-8',
      timeout: 30000,
    });
    assert.match(result, /\[0\]/, 'Should list page 0');
  });

  test('page new creates a new page', () => {
    const result = execSync(`${VIBIUM} page new`, {
      encoding: 'utf-8',
      timeout: 30000,
    });
    assert.match(result, /created|new page/i, 'Should confirm new page created');
  });

  test('page switch switches to a page', () => {
    const result = execSync(`${VIBIUM} page switch 0`, {
      encoding: 'utf-8',
      timeout: 30000,
    });
    assert.match(result, /switched|page 0/i, 'Should confirm page switch');
  });

  test('page close closes a page', () => {
    const result = execSync(`${VIBIUM} page close`, {
      encoding: 'utf-8',
      timeout: 30000,
    });
    assert.match(result, /closed/i, 'Should confirm page closed');
  });
});
