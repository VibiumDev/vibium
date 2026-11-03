/**
 * Daemon CLI: Codegen Tests
 * Records a short flow, then checks `vibium codegen` output for both languages.
 */

const { test, describe, before, after } = require('node:test');
const assert = require('node:assert');
const { execSync, spawn } = require('node:child_process');
const fs = require('node:fs');
const path = require('node:path');
const os = require('node:os');
const { VIBIUM } = require('../helpers');

function clicker(args, opts = {}) {
  const result = execSync(`${VIBIUM} ${args}`, {
    encoding: 'utf-8',
    timeout: opts.timeout || 60000,
    env: { ...process.env, ...opts.env },
  });
  return result.trim();
}

function clickerJSON(args, opts = {}) {
  const result = clicker(`--json ${args}`, opts);
  return JSON.parse(result);
}

function stopDaemon() {
  try {
    execSync(`${VIBIUM} daemon stop`, { encoding: 'utf-8', timeout: 10000 });
  } catch (e) {
    // Daemon may not be running
  }
}

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

describe('Daemon CLI: Codegen', () => {
  const tmpFiles = [];
  const tmpDirs = [];
  let zipPath;

  function tmpPath(name) {
    const p = path.join(os.tmpdir(), `vibium-daemon-codegen-${Date.now()}-${name}`);
    tmpFiles.push(p);
    return p;
  }

  before(() => {
    stopDaemon();
    clicker('daemon start --headless');
    clicker(`go ${baseURL}/login`);

    // One shared recording: navigate, fill, fill, click inside a group.
    clickerJSON('record start');
    clickerJSON(`go ${baseURL}/login`);
    clickerJSON('record group start "log in"');
    clickerJSON('fill "#username" "tom\'s name"');
    clickerJSON('fill "#password" "SuperSecretPassword!"');
    clickerJSON('click "button[type=submit]"');
    clickerJSON('record group stop');
    zipPath = tmpPath('flow.zip');
    clickerJSON(`record stop -o "${zipPath}"`);
  });

  after(() => {
    stopDaemon();
    for (const f of tmpFiles) {
      try { fs.unlinkSync(f); } catch {}
    }
    for (const d of tmpDirs) {
      try { fs.rmSync(d, { recursive: true, force: true }); } catch {}
    }
  });

  test('codegen emits a runnable JavaScript script', () => {
    const script = clicker(`codegen -i "${zipPath}"`);
    assert.ok(script.includes("import { browser } from 'vibium';"), 'Should import the JS client');
    assert.ok(script.includes('const bro = await browser.start();'), 'Should start a browser');
    assert.ok(script.includes(`await vibe.go('${baseURL}/login');`), 'Should replay the navigation');
    assert.ok(script.includes('// group: log in'), 'Should carry the group as a comment');
    assert.ok(script.includes(".fill('tom\\'s name')"), 'Should escape the quoted username');
    assert.ok(script.includes(".fill('[REDACTED]')"), 'Password values stay redacted');
    assert.ok(script.includes("find('button[type=submit]').click()"), 'Should replay the click');
    assert.ok(script.includes('await bro.stop();'), 'Should stop the browser');
  });

  test('codegen --language python emits the sync Python form', () => {
    const script = clicker(`codegen -i "${zipPath}" --language python`);
    assert.ok(script.includes('from vibium import browser'), 'Should import the Python client');
    assert.ok(script.includes('bro = browser.start()'), 'Should start a browser');
    assert.ok(script.includes(`vibe.go('${baseURL}/login')`), 'Should replay the navigation');
    assert.ok(script.includes("vibe.find('#username').fill('tom\\'s name')"), 'Should replay the fill');
    assert.ok(script.includes('# group: log in'), 'Should carry the group as a comment');
    assert.ok(script.includes('bro.stop()'), 'Should stop the browser');
    assert.ok(!script.includes('await'), 'Sync Python has no await');
  });

  test('codegen -o saves the script as a new file', () => {
    const outPath = tmpPath('replay.mjs');
    const result = clicker(`codegen -i "${zipPath}" -o "${outPath}"`);
    assert.ok(result.includes('Script saved'), 'Should confirm the save');
    const script = fs.readFileSync(outPath, 'utf-8');
    assert.ok(script.includes("import { browser } from 'vibium';"), 'File should hold the script');

    assert.throws(
      () => clicker(`codegen -i "${zipPath}" -o "${outPath}"`),
      /already exists/,
      'Should refuse to overwrite an existing file'
    );
  });

  test('codegen rejects an unknown language', () => {
    assert.throws(
      () => clicker(`codegen -i "${zipPath}" --language ruby`),
      /unsupported language/,
      'Should name the problem'
    );
  });

  test('codegen rejects a non-recording input', () => {
    const bogus = tmpPath('bogus.zip');
    fs.writeFileSync(bogus, 'not a zip');
    assert.throws(() => clicker(`codegen -i "${bogus}"`), /invalid recording zip/);
  });

  // Executes the generated script with node. Needs the built JS client; the
  // daemon suite does not build it, so self-skip when dist/ is absent.
  test('generated JavaScript script replays against the fixture', (t) => {
    const clientDir = path.resolve(__dirname, '../../clients/javascript');
    if (!fs.existsSync(path.join(clientDir, 'dist/index.mjs'))) {
      t.skip('clients/javascript/dist is not built');
      return;
    }

    const runDir = fs.mkdtempSync(path.join(os.tmpdir(), 'vibium-codegen-run-'));
    tmpDirs.push(runDir);
    fs.mkdirSync(path.join(runDir, 'node_modules'));
    fs.symlinkSync(clientDir, path.join(runDir, 'node_modules', 'vibium'), 'dir');

    // The record→codegen→edit→run workflow: the one edit is headless mode.
    let script = clicker(`codegen -i "${zipPath}"`);
    script = script.replace('browser.start()', 'browser.start({ headless: true })');
    const scriptPath = path.join(runDir, 'replay.mjs');
    fs.writeFileSync(scriptPath, script);

    execSync(`node "${scriptPath}"`, {
      encoding: 'utf-8',
      timeout: 90000,
      env: { ...process.env, VIBIUM_BIN_PATH: VIBIUM },
    });
  });
});
