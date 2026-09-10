/**
 * CLI Tests: --json envelope on commands that answer without a browser
 *
 * Regression tests for #517: version, install, add-skill, daemon start,
 * daemon stop and config init ignored --json entirely, daemon status emitted
 * JSON without ok, and is-installed printed nothing at all. Every command
 * here must put exactly one parseable JSON object with an "ok" key on
 * stdout, and nothing else.
 *
 * All state is sandboxed: a temp HOME and config dir for the file-writing
 * commands, a temp cache dir for the install ones, and a private session so
 * the daemon cases never touch a real daemon. No browser is launched; the
 * install case hits the already-installed path of a seeded fake cache.
 */

const { test, describe, before, after } = require('node:test');
const assert = require('node:assert');
const { spawnSync } = require('node:child_process');
const fs = require('node:fs');
const path = require('node:path');
const os = require('node:os');
const { VIBIUM } = require('../helpers');

// Short on purpose: the daemon socket lands under the cache dir, and unix
// socket paths are capped at 103 bytes on macOS.
const SESSION = `j${process.pid}`;
let tmpHome;
let shortCache;

function run(args, extraEnv = {}) {
  const result = spawnSync(VIBIUM, args, {
    encoding: 'utf-8',
    timeout: 30000,
    env: {
      ...process.env,
      HOME: tmpHome,
      VIBIUM_CONFIG_DIR: path.join(tmpHome, 'config'),
      VIBIUM_CACHE_DIR: shortCache,
      VIBIUM_SESSION: SESSION,
      VIBIUM_ENGINE_CHANNEL: '',
      ...extraEnv,
    },
  });
  assert.strictEqual(result.error, undefined, `spawn failed: ${result.error}`);
  return result;
}

// Asserts stdout is exactly one JSON object with ok, and returns it parsed.
function parseEnvelope(result) {
  const lines = result.stdout.trim().split('\n');
  assert.strictEqual(lines.length, 1,
    `expected a single JSON line on stdout, got:\n${result.stdout}`);
  let parsed;
  assert.doesNotThrow(() => { parsed = JSON.parse(lines[0]); },
    `stdout is not JSON:\n${result.stdout}`);
  assert.strictEqual(typeof parsed.ok, 'boolean',
    `envelope has no boolean ok:\n${result.stdout}`);
  return parsed;
}

// Chrome fake cache in the platform layout paths.GetChromeExecutable expects,
// same shape as is-installed.test.js.
function seedFakeChromeCache(cacheDir) {
  const versionDir = path.join(cacheDir, 'chrome-for-testing', '999.0.0.0');
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
  const driverName = process.platform === 'win32' ? 'chromedriver.exe' : 'chromedriver';
  fs.writeFileSync(path.join(versionDir, driverName), 'fake', { mode: 0o755 });
}

describe('CLI: --json envelope on no-browser commands (#517)', () => {
  before(() => {
    tmpHome = fs.mkdtempSync(path.join(os.tmpdir(), 'vibium-json-test-'));
    shortCache = fs.mkdtempSync(path.join(os.tmpdir(), 'vjt-'));
  });

  after(() => {
    // The daemon cases stop their own daemon, but a failed assertion can
    // leave one behind on the private session; sweep it before cleanup.
    run(['daemon', 'stop']);
    fs.rmSync(tmpHome, { recursive: true, force: true });
    fs.rmSync(shortCache, { recursive: true, force: true });
  });

  test('version --json wraps name and version', () => {
    const result = run(['--json', 'version']);
    assert.strictEqual(result.status, 0);
    const env = parseEnvelope(result);
    assert.strictEqual(env.ok, true);
    assert.strictEqual(typeof env.result.version, 'string');
    assert.strictEqual(typeof env.result.name, 'string');
  });

  test('version without --json is unchanged', () => {
    const result = run(['version']);
    assert.strictEqual(result.status, 0);
    assert.match(result.stdout, /^\S+ v\S+\n$/);
  });

  test('is-installed --json answers installed:true from a seeded cache', () => {
    const cacheDir = path.join(tmpHome, 'cache-full');
    seedFakeChromeCache(cacheDir);
    const result = run(['--json', 'is-installed'], { VIBIUM_CACHE_DIR: cacheDir });
    assert.strictEqual(result.status, 0);
    const env = parseEnvelope(result);
    assert.strictEqual(env.ok, true);
    assert.deepStrictEqual(env.result, { engine: 'chrome', installed: true });
  });

  test('is-installed --json answers installed:false and still exits 1', () => {
    const cacheDir = path.join(tmpHome, 'cache-empty');
    fs.mkdirSync(cacheDir, { recursive: true });
    const result = run(['--json', 'is-installed'], { VIBIUM_CACHE_DIR: cacheDir });
    // ok means the check ran; the answer and the exit code both say no.
    assert.strictEqual(result.status, 1);
    const env = parseEnvelope(result);
    assert.strictEqual(env.ok, true);
    assert.deepStrictEqual(env.result, { engine: 'chrome', installed: false });
  });

  test('install --json emits only the envelope when already installed', () => {
    const cacheDir = path.join(tmpHome, 'cache-full-install');
    seedFakeChromeCache(cacheDir);
    const result = run(['--json', 'install'], { VIBIUM_CACHE_DIR: cacheDir });
    assert.strictEqual(result.status, 0);
    const env = parseEnvelope(result);
    assert.strictEqual(env.ok, true);
    assert.strictEqual(env.result.engine, 'chrome');
    assert.ok(env.result.chrome, 'result should carry the chrome path');
    // The "already installed" line is progress, not result: stderr only.
    assert.match(result.stderr, /already installed/);
  });

  test('install without --json keeps its human transcript on stdout', () => {
    const cacheDir = path.join(tmpHome, 'cache-full-install2');
    seedFakeChromeCache(cacheDir);
    const result = run(['install'], { VIBIUM_CACHE_DIR: cacheDir });
    assert.strictEqual(result.status, 0);
    assert.match(result.stdout, /already installed/);
    assert.match(result.stdout, /Installation complete!/);
  });

  test('add-skill --json wraps the install location', () => {
    const result = run(['--json', 'add-skill']);
    assert.strictEqual(result.status, 0);
    const env = parseEnvelope(result);
    assert.strictEqual(env.ok, true);
    assert.strictEqual(env.result.skill, 'browser');
    assert.ok(env.result.dir.startsWith(tmpHome), 'dir should be inside the sandbox HOME');
    assert.strictEqual(env.result.files.length, 1);
  });

  test('config init --json reports the files written', () => {
    const result = run(['--json', 'config', 'init', 'all']);
    assert.strictEqual(result.status, 0);
    const env = parseEnvelope(result);
    assert.strictEqual(env.ok, true);
    assert.strictEqual(env.result.mode, '0600');
    assert.strictEqual(env.result.written.length, 2);
    for (const file of env.result.written) {
      assert.ok(fs.existsSync(file), `reported file missing: ${file}`);
    }
  });

  test('daemon status --json carries ok beside its existing keys', () => {
    const result = run(['--json', 'daemon', 'status']);
    // Not running: exit 1 by design, but the JSON must still say ok.
    assert.strictEqual(result.status, 1);
    const env = parseEnvelope(result);
    assert.strictEqual(env.ok, true);
    assert.strictEqual(env.running, false);
  });

  test('daemon stop --json reports stopped:false when nothing runs', () => {
    const result = run(['--json', 'daemon', 'stop']);
    assert.strictEqual(result.status, 0);
    const env = parseEnvelope(result);
    assert.strictEqual(env.ok, true);
    assert.deepStrictEqual(env.result, { running: false, stopped: false });
  });

  test('daemon start/stop --json round-trip on a private session', () => {
    const started = run(['--json', 'daemon', 'start']);
    assert.strictEqual(started.status, 0, `daemon start failed:\n${started.stderr}`);
    const startEnv = parseEnvelope(started);
    assert.strictEqual(startEnv.ok, true);
    assert.strictEqual(startEnv.result.started, true);
    assert.strictEqual(typeof startEnv.result.pid, 'number');

    const status = run(['--json', 'daemon', 'status']);
    assert.strictEqual(status.status, 0);
    const statusEnv = parseEnvelope(status);
    assert.strictEqual(statusEnv.ok, true);
    assert.strictEqual(statusEnv.running, true);

    const stopped = run(['--json', 'daemon', 'stop']);
    assert.strictEqual(stopped.status, 0);
    const stopEnv = parseEnvelope(stopped);
    assert.strictEqual(stopEnv.ok, true);
    assert.deepStrictEqual(stopEnv.result, { running: false, stopped: true });
  });
});
