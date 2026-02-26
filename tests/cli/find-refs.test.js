/**
 * CLI Tests: Find commands return @refs
 * Tests that find, find-all return @refs in oneshot mode
 */

const { test, describe } = require('node:test');
const assert = require('node:assert');
const { execSync } = require('node:child_process');
const { VIBIUM } = require('../helpers');

describe('CLI: Find @refs', () => {
  test('find CSS selector returns @ref', () => {
    const result = execSync(`${VIBIUM} find https://example.com "a"`, {
      encoding: 'utf-8',
      timeout: 30000,
    });
    assert.match(result, /@e1/, 'Should contain @e1 ref');
    assert.match(result, /\[a\]/, 'Should show [a] tag label');
  });

  test('find-all returns multiple @refs', () => {
    const result = execSync(`${VIBIUM} find-all https://example.com "p"`, {
      encoding: 'utf-8',
      timeout: 30000,
    });
    assert.match(result, /@e1/, 'Should contain @e1');
    assert.match(result, /@e2/, 'Should contain @e2');
  });

  test('find-all --limit 1 returns single @ref', () => {
    const result = execSync(`${VIBIUM} find-all https://example.com "p" --limit 1`, {
      encoding: 'utf-8',
      timeout: 30000,
    });
    assert.match(result, /@e1/, 'Should contain @e1');
    assert.ok(!result.includes('@e2'), 'Should not contain @e2');
  });
});
