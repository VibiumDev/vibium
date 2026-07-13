/**
 * CLI Tests: Page Reading Tools
 * Tests text, html, find --all commands
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

  test('find --all returns multiple @refs', () => {
    const result = execSync(`${VIBIUM} find https://example.com "p" --all`, {
      encoding: 'utf-8',
      timeout: 30000,
    });
    assert.match(result, /@e1/, 'Should contain @e1 ref');
    assert.match(result, /\[p\]/, 'Should contain [p] tag label');
  });

  test('find --all with --limit', () => {
    const result = execSync(`${VIBIUM} find https://example.com "p" --all --limit 1`, {
      encoding: 'utf-8',
      timeout: 30000,
    });
    assert.match(result, /@e1/, 'Should contain @e1 ref');
    assert.ok(!result.includes('@e2'), 'Should not contain @e2 ref');
  });

  // Regression test for #209: the daemon socket transport used a fixed-size
  // scanner buffer, so any response over 1MB (large pages, JSON APIs) crashed
  // with "read response: bufio.Scanner: token too long".
  test('text command handles multi-megabyte page text', () => {
    const size = 3 * 1024 * 1024;
    execSync(`${VIBIUM} go https://example.com`, {
      encoding: 'utf-8',
      timeout: 30000,
    });
    execSync(
      `${VIBIUM} eval "document.body.textContent = 'issue-209-marker ' + 'x'.repeat(${size}); 'big page ready'"`,
      { encoding: 'utf-8', timeout: 30000 }
    );
    const result = execSync(`${VIBIUM} text`, {
      encoding: 'utf-8',
      timeout: 30000,
      maxBuffer: 64 * 1024 * 1024,
    });
    assert.match(result, /issue-209-marker/, 'Should contain marker text');
    assert.ok(
      result.length > size,
      `Should return full page text, >${size} chars (got ${result.length})`
    );
  });
});
