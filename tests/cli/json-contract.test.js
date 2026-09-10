/**
 * CLI Tests: the --json output contract, swept over every command path
 *
 * `--json` kept regressing one command at a time (#241, #392, #451, #517)
 * because each fix covered only the commands named in the report. This suite
 * closes the class: every path `vibium commands` lists is declared in the
 * table below, and a tripwire fails the moment a new command ships without
 * declaring its class. Declaring it is also joining it, because the declared
 * class is asserted by running the command.
 *
 * Classes:
 *   envelope     stdout is exactly one JSON object with a boolean "ok".
 *                Holds on failure too: a browser command that cannot launch
 *                must answer {"ok":false,...}, not plain text. Exit codes
 *                are the command's own business (is-installed says no with
 *                ok:true and exit 1), so they are not asserted here.
 *   usage-error  called with no arguments the command refuses: stdout stays
 *                empty, the complaint goes to stderr, exit is non-zero. A
 *                usage error is conventionally plain text, so this class is
 *                about keeping stdout clean for parsers.
 *   exempt       never run, reason documented inline.
 *
 * Everything runs in a sandbox: temp HOME and config dir, an empty cache
 * with downloads disabled so browser launches fail fast instead of
 * downloading Chrome, and a private session so an auto-started daemon never
 * touches a real one. No browser is ever launched; `install` runs against a
 * seeded fake cache and takes its already-installed path.
 */

const { test, describe, before, after } = require('node:test');
const assert = require('node:assert');
const { spawnSync } = require('node:child_process');
const fs = require('node:fs');
const path = require('node:path');
const os = require('node:os');
const { VIBIUM } = require('../helpers');

// Short session and cache names: the daemon socket path must stay under the
// 103-byte macOS unix-socket limit.
const SESSION = `jc${process.pid}`;

const CLASSES = {
  'a11y-tree': 'envelope',
  'add-skill': 'envelope',
  'attr': 'usage-error',
  'back': 'envelope',
  'bidi-test': 'exempt', // five-step diagnostic transcript, not a result
  'check': 'usage-error',
  'click': 'usage-error',
  'config init': 'envelope',
  'content': 'usage-error',
  'cookies': 'envelope',
  'cookies clear': 'envelope',
  'count': 'usage-error',
  'daemon start': 'envelope',
  'daemon status': 'envelope',
  'daemon stop': 'envelope',
  'dblclick': 'usage-error',
  'dialog accept': 'envelope',
  'dialog dismiss': 'envelope',
  'diff map': 'envelope',
  'download dir': 'usage-error',
  'drag': 'usage-error',
  'eval': 'usage-error',
  'fill': 'usage-error',
  'find': 'usage-error',
  'find alt': 'usage-error',
  'find label': 'usage-error',
  'find placeholder': 'usage-error',
  'find role': 'usage-error',
  'find testid': 'usage-error',
  'find text': 'usage-error',
  'find title': 'usage-error',
  'find xpath': 'usage-error',
  'focus': 'usage-error',
  'forward': 'envelope',
  'frame': 'usage-error',
  'frames': 'envelope',
  'geolocation': 'usage-error',
  'go': 'usage-error',
  'highlight': 'usage-error',
  'hover': 'usage-error',
  'html': 'envelope',
  'install': 'envelope',
  'is actionable': 'usage-error',
  'is enabled': 'usage-error',
  'is set': 'usage-error',
  'is visible': 'usage-error',
  'is-installed': 'envelope',
  'keys': 'usage-error',
  'launch-test': 'exempt', // runs until Ctrl+C
  'map': 'envelope',
  'mcp': 'exempt', // server: banner on stderr, exits when stdin closes
  'media': 'usage-error',
  'mouse click': 'envelope',
  'mouse down': 'envelope',
  'mouse move': 'usage-error',
  'mouse up': 'envelope',
  'page close': 'envelope',
  'page new': 'envelope',
  'page switch': 'usage-error',
  'pages': 'envelope',
  'paths': 'envelope',
  'pdf': 'envelope',
  'pipe': 'exempt', // stdout carries the BiDi protocol stream by design
  'press': 'usage-error',
  'ready': 'envelope',
  'ready ai': 'envelope',
  'ready browser': 'envelope',
  'record chunk start': 'envelope',
  'record chunk stop': 'envelope',
  'record group start': 'usage-error',
  'record group stop': 'envelope',
  'record start': 'envelope',
  'record stop': 'envelope',
  'reload': 'envelope',
  'run': 'usage-error',
  'screenshot': 'envelope',
  'scroll': 'envelope',
  'scroll into-view': 'usage-error',
  'select': 'usage-error',
  'serve': 'exempt', // runs until interrupted
  'set': 'usage-error',
  'sleep': 'usage-error',
  'start': 'envelope',
  'stop': 'envelope',
  'storage': 'envelope',
  'storage restore': 'usage-error',
  'text': 'envelope',
  'title': 'envelope',
  'type': 'usage-error',
  'unset': 'usage-error',
  'upload': 'usage-error',
  'url': 'envelope',
  'value': 'usage-error',
  'version': 'envelope',
  'viewport': 'envelope',
  'wait': 'usage-error',
  'wait fn': 'usage-error',
  'wait load': 'envelope',
  'wait text': 'usage-error',
  'wait url': 'usage-error',
  'window': 'envelope',
  'ws-test': 'usage-error',
};

let tmpHome;
let emptyCache;
let fakeCache;

function run(args, extraEnv = {}) {
  const result = spawnSync(VIBIUM, args, {
    encoding: 'utf-8',
    timeout: 30000,
    stdio: ['ignore', 'pipe', 'pipe'],
    env: {
      ...process.env,
      HOME: tmpHome,
      VIBIUM_CONFIG_DIR: path.join(tmpHome, 'config'),
      VIBIUM_CACHE_DIR: emptyCache,
      VIBIUM_SESSION: SESSION,
      VIBIUM_ENGINE_CHANNEL: '',
      VIBIUM_SKIP_BROWSER_DOWNLOAD: '1',
      ...extraEnv,
    },
  });
  assert.strictEqual(result.error, undefined, `spawn failed: ${result.error}`);
  return result;
}

// install is envelope-class via its already-installed path; give it a seeded
// cache and let it "install" (Install() checks the download switch before
// the cache, so the switch must be off).
const ENV_OVERRIDES = {
  'install': () => ({ VIBIUM_CACHE_DIR: fakeCache, VIBIUM_SKIP_BROWSER_DOWNLOAD: '' }),
};

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

describe('CLI: --json output contract on every command path', () => {
  before(() => {
    tmpHome = fs.mkdtempSync(path.join(os.tmpdir(), 'vibium-jc-home-'));
    emptyCache = fs.mkdtempSync(path.join(os.tmpdir(), 'jc-'));
    fakeCache = fs.mkdtempSync(path.join(os.tmpdir(), 'jcf-'));
    seedFakeChromeCache(fakeCache);
  });

  after(() => {
    // Some envelope commands auto-start a daemon on the private session.
    run(['daemon', 'stop']);
    for (const dir of [tmpHome, emptyCache, fakeCache]) {
      fs.rmSync(dir, { recursive: true, force: true });
    }
  });

  test('every command path declares its class (tripwire)', () => {
    const result = run(['commands']);
    assert.strictEqual(result.status, 0, `vibium commands failed:\n${result.stderr}`);
    const listed = JSON.parse(result.stdout)[''];
    const declared = Object.keys(CLASSES);

    const missing = listed.filter((p) => !(p in CLASSES));
    const stale = declared.filter((p) => !listed.includes(p));
    assert.deepStrictEqual(missing, [],
      'new command paths must declare a --json class in this table (envelope, usage-error, or exempt with a reason)');
    assert.deepStrictEqual(stale, [],
      'table entries for command paths that no longer exist');
  });

  for (const [cmd, cls] of Object.entries(CLASSES)) {
    if (cls === 'exempt') continue;

    test(`${cmd} [${cls}]`, () => {
      const extraEnv = ENV_OVERRIDES[cmd] ? ENV_OVERRIDES[cmd]() : {};
      const result = run(['--json', ...cmd.split(' ')], extraEnv);

      if (cls === 'usage-error') {
        assert.notStrictEqual(result.status, 0,
          `expected a non-zero exit with no arguments\nstdout: ${result.stdout}`);
        assert.strictEqual(result.stdout, '',
          `a usage error must leave stdout empty for parsers\nstdout: ${result.stdout}`);
        assert.notStrictEqual(result.stderr.trim(), '',
          'the usage error itself went missing from stderr');
        return;
      }

      const lines = result.stdout.trim().split('\n');
      assert.strictEqual(lines.length, 1,
        `expected a single JSON line on stdout\nstdout: ${result.stdout}\nstderr: ${result.stderr}`);
      let parsed;
      assert.doesNotThrow(() => { parsed = JSON.parse(lines[0]); },
        `stdout is not JSON\nstdout: ${result.stdout}\nstderr: ${result.stderr}`);
      assert.strictEqual(typeof parsed.ok, 'boolean',
        `envelope has no boolean ok\nstdout: ${result.stdout}`);
    });
  }
});
