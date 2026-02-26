/**
 * CLI Tests: Actionability Checks
 * Tests auto-wait and actionability behavior
 */

const { test, describe } = require('node:test');
const assert = require('node:assert');
const { execSync } = require('node:child_process');
const { VIBIUM } = require('../helpers');

describe('CLI: Actionability', () => {
  test('check-actionable reports visibility status', () => {
    const result = execSync(`${VIBIUM} check-actionable https://example.com "a"`, {
      encoding: 'utf-8',
      timeout: 30000,
    });
    assert.match(result, /Visible.*true/i, 'Link should be visible');
    assert.match(result, /Stable.*true/i, 'Link should be stable');
    assert.match(result, /ReceivesEvents.*true/i, 'Link should receive events');
    assert.match(result, /Enabled.*true/i, 'Link should be enabled');
  });

  test('click with short timeout fails on non-existent element', () => {
    assert.throws(
      () => {
        execSync(`${VIBIUM} click https://example.com "#does-not-exist" --timeout 1s`, {
          encoding: 'utf-8',
          timeout: 10000,
        });
      },
      /timeout|not found/i,
      'Should timeout or report not found'
    );
  });
});
