/** Readiness inspects browser files and probes AI without launching a browser. */
const { test } = require('node:test');
const assert = require('node:assert/strict');
const { execFile } = require('node:child_process');
const { promisify } = require('node:util');
const http = require('node:http');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const { VIBIUM } = require('../helpers');
const exec = promisify(execFile);

function environment(t, extra = {}) {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'vs-'));
  t.after(() => fs.rmSync(dir, { recursive: true, force: true }));
  return {
    ...process.env, HOME: dir, USERPROFILE: dir, VIBIUM_CACHE_DIR: path.join(dir, 'cache'),
    VIBIUM_SESSION: 'ready-ai', VIBIUM_ENGINE: 'chrome', VIBIUM_ENGINE_PATH: '',
    VIBIUM_ENGINE_CHANNEL: '', VIBIUM_ENGINE_VERSION: '', VIBIUM_CONNECT_URL: '',
    VIBIUM_AI_PROVIDER: '', VIBIUM_AI_MODEL: '', OPENAI_API_KEY: '',
    VIBIUM_AI_BASE_URL: '', VIBIUM_AI_REASONING_EFFORT: '', ...extra,
  };
}
async function run(env, args = ['ready', 'ai', '--json']) {
  try { return { ...(await exec(VIBIUM, args, { env, timeout: 15000 })), code: 0 }; }
  catch (error) { if (typeof error.code !== 'number') throw error; return error; }
}
function noBrowser(env) { assert.equal(fs.existsSync(env.VIBIUM_CACHE_DIR), false, 'created daemon/browser files'); }

async function provider(t, handler) {
  let requests = 0;
  const server = http.createServer(async (req, res) => {
    try {
      const chunks = [];
      for await (const chunk of req) chunks.push(chunk);
      const body = JSON.parse(Buffer.concat(chunks));
      requests++;
      handler(req, res, body, requests);
    } catch (err) { t.diagnostic(err.stack); res.writeHead(500); res.end('{}'); }
  });
  await new Promise(resolve => server.listen(0, '127.0.0.1', resolve));
  t.after(() => new Promise(resolve => server.close(resolve)));
  return { url: `http://127.0.0.1:${server.address().port}/v1`, requests: () => requests };
}

function answer(res, content, toolCalls) {
  res.setHeader('Content-Type', 'application/json');
  res.end(JSON.stringify({ choices: [{ finish_reason: toolCalls ? 'tool_calls' : 'stop',
    message: { role: 'assistant', content, tool_calls: toolCalls } }] }));
}

test('ready ai lists missing settings and detects an unsourced env file without reading it', async t => {
  const env = environment(t);
  const settings = path.join(env.HOME, '.config', 'vibium', 'ai.env');
  fs.mkdirSync(path.dirname(settings), { recursive: true });
  fs.writeFileSync(settings, 'OPENAI_API_KEY=secret-file-marker\nVIBIUM_AI_PROVIDER=openai\n', { mode: 0o600 });
  const before = fs.readFileSync(settings);
  for (const args of [['ready', 'ai'], ['ready', 'ai', '--json']]) {
    const result = await run(env, args);
    assert.equal(result.code, 1);
    assert.match(result.stdout, /VIBIUM_AI_PROVIDER/);
    assert.match(result.stdout, /VIBIUM_AI_MODEL/);
    assert.match(result.stdout, /source ~\/\.config\/vibium\/ai\.env/);
    // Both notes contain that source line, so assert on the branch itself:
    // the file is present, so readiness must say it found one.
    assert.match(result.stdout, /Found ~\/\.config\/vibium\/ai\.env/);
    assert.doesNotMatch(result.stdout, /No AI settings file yet/);
    assert.doesNotMatch(result.stdout + result.stderr, /secret-file-marker/);
    if (args.includes('--json')) {
      const body = JSON.parse(result.stdout);
      assert.equal(body.ok, false);
      assert.equal(body.result.ready, false);
      assert.equal(body.result.checks.find(c => c.name === 'provider').status, 'skipped');
      assert.equal(body.result.checks.find(c => c.name === 'credentials').status, 'skipped');
      assert.equal(body.result.checks.find(c => c.name === 'VIBIUM_AI_BASE_URL').status, 'skipped');
      assert.equal(body.result.checks.find(c => c.name === 'VIBIUM_AI_REASONING_EFFORT').status, 'skipped');
      assert.match(body.result.summary, /rerun vibium ready ai\./);
    }
    assert.doesNotMatch(result.stdout, /\[PASSED\]|OPENAI_API_KEY/);
    assert.match(result.stdout, /rerun vibium ready ai\./);
    noBrowser(env);
  }
  assert.deepEqual(fs.readFileSync(settings), before);
});

test('ready ai exercises the actual provider transport and reports readiness in text and JSON', async t => {
  const fixture = await provider(t, (req, res, body, requests) => {
    assert.equal(req.url, '/v1/chat/completions');
    assert.equal(req.headers.authorization, 'Bearer secret-api-marker');
    assert.equal(body.model, 'fixture-model');
    assert.equal(body.reasoning_effort, 'none');
    assert.deepEqual(body.tools.map(t => t.function.name), ['verifier_ping']);
    assert.doesNotMatch(JSON.stringify(body.messages), /secret-api-marker|PRIVATE-REASONING/);
    if (requests % 2 === 1) {
      assert.equal(body.messages.length, 2);
      answer(res, 'PRIVATE-REASONING', [{ id: 'ping', type: 'function', function: { name: 'verifier_ping', arguments: '{}' } }]);
    } else {
      assert.equal(body.messages.at(-1).role, 'tool');
      answer(res, body.messages.at(-1).content);
    }
  });
  const env = environment(t, { VIBIUM_AI_PROVIDER: 'openai-compatible',
    VIBIUM_AI_BASE_URL: fixture.url, VIBIUM_AI_MODEL: 'fixture-model',
    VIBIUM_AI_REASONING_EFFORT: 'none', OPENAI_API_KEY: 'secret-api-marker' });
  for (const args of [['ready', 'ai'], ['ready', 'ai', '--json']]) {
    const result = await run(env, args);
    assert.equal(result.code, 0, result.stdout + result.stderr);
    assert.doesNotMatch(result.stdout + result.stderr, /secret-api-marker|PRIVATE-REASONING/);
    if (args.includes('--json')) {
      const body = JSON.parse(result.stdout);
      assert.equal(body.ok, true);
      assert.equal(body.result.ready, true);
      assert.ok(body.result.checks.every(c => c.status === 'passed'));
    } else { assert.match(result.stdout, /READY: AI configuration/); }
    noBrowser(env);
  }
  assert.equal(fixture.requests(), 4);
});

test('provider authentication failure gives a fix, exit 1, and no leaked response body', async t => {
  const fixture = await provider(t, (req, res) => {
    res.writeHead(401);
    res.end(JSON.stringify({ error: { message: 'secret-api-marker', code: 'secret-api-marker', param: 'secret-api-marker' } }));
  });
  const env = environment(t, { VIBIUM_AI_PROVIDER: 'openai-compatible',
    VIBIUM_AI_BASE_URL: fixture.url, VIBIUM_AI_MODEL: 'fixture-model' });
  const result = await run(env);
  assert.equal(result.code, 1);
  const body = JSON.parse(result.stdout);
  const check = body.result.checks.find(c => c.name === 'provider');
  assert.equal(check.status, 'failed');
  assert.match(check.message, /HTTP 401/);
  assert.match(check.fix, /API key/);
  assert.doesNotMatch(result.stdout + result.stderr, /secret-api-marker/);
  assert.equal(fixture.requests(), 1);
  noBrowser(env);
});

test('check setup errors point to ready ai and help runs without setup', async t => {
  const env = environment(t);
  const failure = await run(env, ['check', 'the cart works', '--json']);
  assert.equal(failure.code, 1);
  assert.match(JSON.parse(failure.stdout).error, /vibium ready ai/);
  const help = await run(env, ['ready', 'ai', '--help']);
  assert.equal(help.code, 0);
  assert.match(help.stdout, /API charges may apply/);
  noBrowser(env);
});

test('ready browser reports missing installation without downloads, daemon, or AI access', async t => {
  const env = environment(t, { VIBIUM_AI_PROVIDER: 'invalid-provider' });
  for (const browser of ['chrome', 'firefox']) {
    const response = await run(env, ['ready', 'browser', browser, '--json']);
    assert.equal(response.code, 1);
    const result = JSON.parse(response.stdout).result;
    assert.equal(result.scope, 'browser');
    assert.equal(result.ready, false);
    assert.equal(result.checks.find(c => c.name === 'browser.installation').status, 'failed');
    assert.match(result.checks[0].fix, new RegExp(`vibium install --engine ${browser}`));
    assert.equal(result.browsers.find(b => b.selected).engine, browser);
    assert.ok(!result.checks.some(c => c.name === 'provider'));
    noBrowser(env);
  }
});

test('plain ready skips unconfigured AI but fails an unavailable browser', async t => {
  const env = environment(t);
  const response = await run(env, ['ready', '--json']);
  assert.equal(response.code, 1);
  const result = JSON.parse(response.stdout).result;
  assert.equal(result.scope, 'all');
  assert.equal(result.checks.find(c => c.name === 'ai').status, 'skipped');
  assert.equal(result.checks.find(c => c.name === 'browser.installation').status, 'failed');
  noBrowser(env);
});

test('ready ai provider selection overrides defaults and ignores broken browser settings', async t => {
  const fixture = await provider(t, (req, res, body, requests) => {
    assert.equal(body.model, 'selected-model');
    assert.equal(body.reasoning_effort, undefined);
    if (requests % 2) answer(res, null, [{ id: 'ping', type: 'function', function: { name: 'verifier_ping', arguments: '{}' } }]);
    else answer(res, body.messages.at(-1).content);
  });
  const env = environment(t, { VIBIUM_AI_PROVIDER: 'anthropic', VIBIUM_AI_MODEL: 'old-model',
    VIBIUM_AI_BASE_URL: 'http://127.0.0.1:1', VIBIUM_AI_REASONING_EFFORT: 'invalid',
    VIBIUM_ENGINE: 'invalid-browser', VIBIUM_ENGINE_CHANNEL: 'invalid-channel', VIBIUM_ENGINE_PATH: '/does-not-exist' });
  const args = ['ready', 'ai', 'local', '--model', 'selected-model', '--base-url', fixture.url, '--json'];
  assert.equal((await run(env, args)).code, 0);
  assert.equal(fixture.requests(), 2);
  assert.equal((await run(env, ['ready', 'ai', 'local', '--json'])).code, 1, 'provider change must require model');
  assert.equal(fixture.requests(), 2);
  noBrowser(env);
});

test('ready rejects ambiguous selections and never treats them as model prompts', async t => {
  const env = environment(t);
  for (const args of [
    ['ready', 'ai', 'local', '--provider', 'openai'],
    ['ready', 'browser', 'firefox', '--engine', 'chrome'],
    ['ready', 'browser', 'chrome', '--channel', 'release'],
    ['ready', 'browser', 'safari'], ['ready', 'unknown'], ['ready', 'ai', 'local', 'extra'],
  ]) {
    assert.equal((await run(env, [...args, '--json'])).code, 1, args.join(' '));
    noBrowser(env);
  }
});

test('plain ready reports browser and AI failures together', async t => {
  const env = environment(t, { VIBIUM_AI_MODEL: 'partially-configured' });
  const response = await run(env, ['ready', '--json']);
  assert.equal(response.code, 1);
  const result = JSON.parse(response.stdout).result;
  assert.equal(result.checks.find(c => c.name === 'browser.installation').status, 'failed');
  assert.equal(result.checks.find(c => c.name === 'VIBIUM_AI_PROVIDER').status, 'failed');
  noBrowser(env);
});

// These executables are intentionally harmless traps. A launch regression fails
// the check (and writes a marker on Unix) without ever starting a real browser.
function installedBrowsers(env) {
  const version = path.join(env.VIBIUM_CACHE_DIR, 'chrome-for-testing', '150.0.0.1');
  const chrome = path.join(version, process.platform === 'darwin'
    ? 'Google Chrome for Testing.app/Contents/MacOS/Google Chrome for Testing'
    : process.platform === 'win32' ? 'chrome.exe' : 'chrome');
  const driver = path.join(version, process.platform === 'win32' ? 'chromedriver.exe' : 'chromedriver');
  const firefox = path.join(env.VIBIUM_CACHE_DIR, 'firefox', 'beta', '150.0', process.platform === 'darwin'
    ? 'Firefox.app/Contents/MacOS/firefox' : process.platform === 'win32' ? 'firefox/firefox.exe' : 'firefox/firefox');
  env.READY_LAUNCH_MARKER = path.join(env.HOME, 'unexpected-browser-launch');
  for (const executable of [chrome, driver, firefox]) {
    fs.mkdirSync(path.dirname(executable), { recursive: true });
    fs.writeFileSync(executable, '#!/bin/sh\nprintf launched > "$READY_LAUNCH_MARKER"\nexit 91\n', { mode: 0o700 });
  }
  return { chrome, driver, firefox };
}

test('ready inspects installed Chrome, driver, and Firefox without executing them or creating sessions', async t => {
  const env = environment(t);
  const binaries = installedBrowsers(env);
  const fixture = await provider(t, (req, res) => { res.writeHead(500); res.end('{}'); });
  env.VIBIUM_CONNECT_URL = fixture.url;
  const before = fs.readdirSync(env.VIBIUM_CACHE_DIR, { recursive: true }).sort();
  for (const args of [
    ['ready'], ['ready', 'browser', 'chrome'], ['ready', 'browser', 'firefox', '--channel', 'beta'],
    ['ready', 'browser', 'firefox', '--channel', 'beta', '--json'], ['ready', '--json'],
    ['ready', 'browser', 'chrome', '--json'],
  ]) {
    const response = await run(env, args);
    assert.equal(response.code, 0, response.stdout + response.stderr);
    assert.equal(fs.existsSync(env.READY_LAUNCH_MARKER), false, 'executed a browser or driver');
    if (args.includes('--json')) {
      const result = JSON.parse(response.stdout).result;
      assert.equal(result.ready, true);
      assert.equal(result.checks.find(c => c.name === 'browser.installation').status, 'passed');
      assert.equal(result.checks.find(c => c.name === 'browser.connection').status, 'skipped');
      const selected = result.browsers.filter(b => b.selected);
      assert.equal(selected.length, 1);
      assert.equal(selected[0].path, args.includes('firefox') ? binaries.firefox : binaries.chrome);
    } else {
      assert.match(response.stdout, /\[SKIPPED\] browser.connection/);
      assert.match(response.stdout, /launch and BiDi connectivity were not tested/);
    }
    assert.deepEqual(fs.readdirSync(env.VIBIUM_CACHE_DIR, { recursive: true }).sort(), before);
  }
  assert.equal(fixture.requests(), 0, 'contacted an existing session');
});

test('plain ready probes configured AI while leaving installed browser executables untouched', async t => {
  const fixture = await provider(t, (req, res, body, requests) => {
    if (requests === 1) answer(res, null, [{ id: 'ping', type: 'function', function: { name: 'verifier_ping', arguments: '{}' } }]);
    else answer(res, body.messages.at(-1).content);
  });
  const env = environment(t, { VIBIUM_AI_PROVIDER: 'local', VIBIUM_AI_MODEL: 'fixture', VIBIUM_AI_BASE_URL: fixture.url });
  installedBrowsers(env);
  const response = await run(env, ['ready', '--json']);
  assert.equal(response.code, 0, response.stdout + response.stderr);
  const result = JSON.parse(response.stdout).result;
  assert.equal(result.checks.find(c => c.name === 'browser.connection').status, 'skipped');
  assert.equal(result.checks.find(c => c.name === 'provider').status, 'passed');
  assert.equal(fixture.requests(), 2);
  assert.equal(fs.existsSync(env.READY_LAUNCH_MARKER), false);
});
