/**
 * CLI Tests: Viewport & Window Commands
 * Tests viewport and window management via the daemon.
 */

const { test, describe } = require('node:test');
const assert = require('node:assert');
const { execSync } = require('node:child_process');
const { VIBIUM } = require('../helpers');

describe('CLI: Viewport & Window Commands', () => {
  test('viewport returns width and height', () => {
    const result = execSync(`${VIBIUM} viewport`, {
      encoding: 'utf-8',
      timeout: 30000,
    });
    const vp = JSON.parse(result);
    assert.ok(typeof vp.width === 'number', 'Should have numeric width');
    assert.ok(typeof vp.height === 'number', 'Should have numeric height');
  });

  test('viewport with args sets dimensions', () => {
    execSync(`${VIBIUM} viewport 800 600`, {
      encoding: 'utf-8',
      timeout: 30000,
    });
    const result = execSync(`${VIBIUM} viewport`, {
      encoding: 'utf-8',
      timeout: 30000,
    });
    const vp = JSON.parse(result);
    assert.strictEqual(vp.width, 800);
    assert.strictEqual(vp.height, 600);
  });

  test('window returns state and dimensions', () => {
    const result = execSync(`${VIBIUM} window`, {
      encoding: 'utf-8',
      timeout: 30000,
    });
    const win = JSON.parse(result);
    assert.ok(typeof win.width === 'number', 'Should have numeric width');
    assert.ok(typeof win.height === 'number', 'Should have numeric height');
  });

  test('window with args sets dimensions', () => {
    execSync(`${VIBIUM} window 900 700`, {
      encoding: 'utf-8',
      timeout: 30000,
    });
    const result = execSync(`${VIBIUM} window`, {
      encoding: 'utf-8',
      timeout: 30000,
    });
    const win = JSON.parse(result);
    assert.strictEqual(win.width, 900);
    assert.strictEqual(win.height, 700);
  });

  test('window resize survives closing the active page (#346)', () => {
    execSync(`${VIBIUM} page new`, { encoding: 'utf-8', timeout: 30000 });
    execSync(`${VIBIUM} page switch 0`, { encoding: 'utf-8', timeout: 30000 });
    execSync(`${VIBIUM} page close`, { encoding: 'utf-8', timeout: 30000 });
    execSync(`${VIBIUM} window 910 710`, {
      encoding: 'utf-8',
      timeout: 30000,
    });
    const result = execSync(`${VIBIUM} window`, {
      encoding: 'utf-8',
      timeout: 30000,
    });
    const win = JSON.parse(result);
    assert.strictEqual(win.width, 910);
    assert.strictEqual(win.height, 710);
  });
});
