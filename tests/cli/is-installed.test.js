/**
 * CLI Tests: is-installed command
 * Tests that `vibium is-installed` correctly checks for Chrome and chromedriver.
 */

const { test, describe } = require('node:test');
const assert = require('node:assert');
const { execFileSync } = require('node:child_process');
const fs = require('node:fs');
const path = require('node:path');
const os = require('node:os');
const { VIBIUM } = require('../helpers');

/**
 * Run `vibium is-installed` with VIBIUM_CACHE_DIR pointing to a custom dir.
 * Returns the exit code.
 */
function runIsInstalled(cacheDir) {
  try {
    execFileSync(VIBIUM, ['is-installed'], {
      timeout: 10000,
      // Pin the channel: the fake cache seeds the default-channel layout,
      // and the Beta Watch workflow exports VIBIUM_ENGINE_CHANNEL=beta
      // job-wide, which sent resolution into a beta/ subdirectory nobody
      // created. Same class as #479.
      env: { ...process.env, VIBIUM_CACHE_DIR: cacheDir, VIBIUM_ENGINE_CHANNEL: '' },
    });
    return 0;
  } catch (err) {
    return err.status;
  }
}

/**
 * Create a fake Chrome for Testing directory with only the specified binaries.
 */
function createFakeCacheDir(opts = {}) {
  const cacheDir = fs.mkdtempSync(path.join(os.tmpdir(), 'vibium-test-'));
  const versionDir = path.join(cacheDir, 'chrome-for-testing', '999.0.0.0');

  if (opts.chrome) {
    let chromePath;
    if (process.platform === 'darwin') {
      chromePath = path.join(versionDir, 'Google Chrome for Testing.app', 'Contents', 'MacOS', 'Google Chrome for Testing');
    } else if (process.platform === 'win32') {
      chromePath = path.join(versionDir, 'chrome.exe');
    } else {
      chromePath = path.join(versionDir, 'chrome');
    }
    fs.mkdirSync(path.dirname(chromePath), { recursive: true });
    fs.writeFileSync(chromePath, 'fake', { mode: 0o755 });
  }

  if (opts.chromedriver) {
    const driverName = process.platform === 'win32' ? 'chromedriver.exe' : 'chromedriver';
    const driverPath = path.join(versionDir, driverName);
    fs.mkdirSync(path.dirname(driverPath), { recursive: true });
    fs.writeFileSync(driverPath, 'fake', { mode: 0o755 });
  }

  return cacheDir;
}

describe('CLI: is-installed', () => {
  test('exits 0 when Chrome and chromedriver are installed', () => {
    execFileSync(VIBIUM, ['is-installed'], { timeout: 10000 });
  });

  test('produces no stdout output', () => {
    const output = execFileSync(VIBIUM, ['is-installed'], {
      encoding: 'utf-8',
      timeout: 10000,
    });
    assert.strictEqual(output, '', 'is-installed should produce no output');
  });

  test('exits 1 when nothing is installed', () => {
    const cacheDir = createFakeCacheDir();
    assert.strictEqual(runIsInstalled(cacheDir), 1);
    fs.rmSync(cacheDir, { recursive: true });
  });

  test('exits 1 when only Chrome is installed (no chromedriver)', () => {
    const cacheDir = createFakeCacheDir({ chrome: true });
    assert.strictEqual(runIsInstalled(cacheDir), 1);
    fs.rmSync(cacheDir, { recursive: true });
  });

  test('exits 1 when only chromedriver is installed (no Chrome)', () => {
    const cacheDir = createFakeCacheDir({ chromedriver: true });
    assert.strictEqual(runIsInstalled(cacheDir), 1);
    fs.rmSync(cacheDir, { recursive: true });
  });

  test('exits 0 when both Chrome and chromedriver are present', () => {
    const cacheDir = createFakeCacheDir({ chrome: true, chromedriver: true });
    assert.strictEqual(runIsInstalled(cacheDir), 0);
    fs.rmSync(cacheDir, { recursive: true });
  });
});
