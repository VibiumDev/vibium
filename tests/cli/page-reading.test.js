/**
 * CLI Tests: Page Reading Tools
 * Tests text, html, find-all commands in oneshot mode
 */

const { test, describe } = require('node:test');
const assert = require('node:assert');
const { execSync } = require('node:child_process');
const { VIBIUM } = require('../helpers');

describe('CLI: Page Reading', () => {
  test('text command returns page text', () => {
    const result = execSync(`${VIBIUM} text https://example.com`, {
      encoding: 'utf-8',
      timeout: 30000,
    });
    assert.match(result, /Example Domain/, 'Should contain page text');
  });

  test('text command with selector returns element text', () => {
    const result = execSync(`${VIBIUM} text https://example.com "h1"`, {
      encoding: 'utf-8',
      timeout: 30000,
    });
    assert.match(result, /Example Domain/, 'Should contain h1 text');
  });

  test('html command returns page HTML', () => {
    const result = execSync(`${VIBIUM} html https://example.com "h1"`, {
      encoding: 'utf-8',
      timeout: 30000,
    });
    assert.match(result, /Example Domain/, 'Should contain HTML');
  });

  test('html command with --outer returns outer HTML', () => {
    const result = execSync(`${VIBIUM} html https://example.com "h1" --outer`, {
      encoding: 'utf-8',
      timeout: 30000,
    });
    assert.match(result, /<h1>/, 'Should contain h1 tag');
    assert.match(result, /Example Domain/, 'Should contain text');
  });

  test('find-all command returns multiple @refs', () => {
    const result = execSync(`${VIBIUM} find-all https://example.com "p"`, {
      encoding: 'utf-8',
      timeout: 30000,
    });
    assert.match(result, /@e1/, 'Should contain @e1 ref');
    assert.match(result, /\[p\]/, 'Should contain [p] tag label');
  });

  test('find-all command with --limit', () => {
    const result = execSync(`${VIBIUM} find-all https://example.com "p" --limit 1`, {
      encoding: 'utf-8',
      timeout: 30000,
    });
    assert.match(result, /@e1/, 'Should contain @e1 ref');
    assert.ok(!result.includes('@e2'), 'Should not contain @e2 ref');
  });
});
