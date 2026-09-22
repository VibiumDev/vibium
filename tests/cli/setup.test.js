/** setup writes AI settings and installs a missing browser without launching one. */
const { test } = require('node:test');
const assert = require('node:assert/strict');
const { execFile } = require('node:child_process');
const { promisify } = require('node:util');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const { VIBIUM } = require('../helpers');
const exec = promisify(execFile);

function environment(t, extra = {}) {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'vs-setup-'));
  t.after(() => fs.rmSync(dir, { recursive: true, force: true }));
  const env = {
    ...process.env, HOME: dir, USERPROFILE: dir, VIBIUM_CACHE_DIR: path.join(dir, 'cache'),
    VIBIUM_SESSION: 'setup-cli', VIBIUM_ENGINE: 'chrome', VIBIUM_ENGINE_PATH: '',
    VIBIUM_ENGINE_CHANNEL: '', VIBIUM_ENGINE_VERSION: '', VIBIUM_CONNECT_URL: '',
    ...extra,
  };
  for (const key of ['VIBIUM_AI_PROVIDER', 'VIBIUM_AI_MODEL', 'OPENAI_API_KEY',
    'ANTHROPIC_API_KEY', 'GOOGLE_API_KEY', 'VIBIUM_AI_BASE_URL', 'VIBIUM_AI_REASONING_EFFORT']) {
    if (!(key in extra)) delete env[key];
  }
  return env;
}

async function run(env, args) {
  try { return { ...(await exec(VIBIUM, args, { env, timeout: 15000 })), code: 0 }; }
  catch (error) { if (typeof error.code !== 'number') throw error; return error; }
}

function seedChrome(env) {
  const version = path.join(env.VIBIUM_CACHE_DIR, 'chrome-for-testing', '150.0.0.1');
  const chrome = path.join(version, process.platform === 'darwin'
    ? 'Google Chrome for Testing.app/Contents/MacOS/Google Chrome for Testing'
    : process.platform === 'win32' ? 'chrome.exe' : 'chrome');
  const driver = path.join(version, process.platform === 'win32' ? 'chromedriver.exe' : 'chromedriver');
  env.READY_LAUNCH_MARKER = path.join(env.HOME, 'unexpected-browser-launch');
  for (const executable of [chrome, driver]) {
    fs.mkdirSync(path.dirname(executable), { recursive: true });
    fs.writeFileSync(executable, '#!/bin/sh\nprintf launched > "$READY_LAUNCH_MARKER"\nexit 91\n', { mode: 0o700 });
  }
  return { chrome, driver };
}

test('setup --non-interactive --json with a seeded browser cache returns an ok envelope', async t => {
  const env = environment(t);
  seedChrome(env);
  const result = await run(env, ['setup', '--non-interactive', '--json']);
  assert.equal(result.code, 0, result.stdout + result.stderr);
  const body = JSON.parse(result.stdout);
  assert.equal(body.ok, true);
  assert.deepEqual(body.result.sections.map(s => s.name), ['ai', 'skills', 'tasks', 'browser'],
    'prompt-driven sections must run before the browser download');
  const sections = Object.fromEntries(body.result.sections.map(s => [s.name, s]));
  assert.equal(sections.browser.status, 'done');
  assert.equal(sections.ai.status, 'skipped');
  assert.equal(sections.skills.status, 'skipped');
  assert.equal(sections.tasks.status, 'skipped');
  assert.equal(fs.existsSync(env.READY_LAUNCH_MARKER), false, 'launched a browser');
});

test('setup without an API key ends on the next step, not a failure', async t => {
  const env = environment(t);
  seedChrome(env);
  const settings = path.join(env.HOME, '.config', 'vibium', 'ai.env');
  fs.mkdirSync(path.dirname(settings), { recursive: true });
  fs.writeFileSync(settings, 'export VIBIUM_AI_PROVIDER=openai\nexport VIBIUM_AI_MODEL=gpt-test\n', { mode: 0o600 });
  const result = await run(env, ['setup', '--non-interactive', '--json']);
  assert.equal(result.code, 0, result.stdout + result.stderr);
  const body = JSON.parse(result.stdout);
  assert.equal(body.ok, true);
  const cred = body.result.ready.checks.find(c => c.name === 'OPENAI_API_KEY');
  assert.equal(cred.status, 'skipped');
  assert.match(cred.message, /add it to .*ai\.env/);
  const provider = body.result.ready.checks.find(c => c.name === 'provider');
  assert.equal(provider.status, 'skipped');
  assert.match(provider.message, /No API key; provider was not contacted/);
  assert.equal(body.result.ready.checks.filter(c => c.status === 'failed').length, 0);
  assert.match(body.result.ready.summary, /Add your API key to .*ai\.env, then run vibium ready ai/);
});

test('setup rejects an unsupported channel instead of ignoring it', async t => {
  const env = environment(t);
  seedChrome(env);
  const result = await run(env, ['setup', '--channel', 'bogus', '--non-interactive', '--json']);
  assert.notEqual(result.code, 0);
  assert.match(result.stdout + result.stderr, /unsupported channel "bogus"/);
});

test('setup does not overwrite an existing ai.env', async t => {
  const env = environment(t);
  seedChrome(env);
  const settings = path.join(env.HOME, '.config', 'vibium', 'ai.env');
  fs.mkdirSync(path.dirname(settings), { recursive: true });
  const original = 'export VIBIUM_AI_PROVIDER=openai\nexport VIBIUM_AI_MODEL=keep-me\n';
  fs.writeFileSync(settings, original, { mode: 0o600 });
  const result = await run(env, ['setup', 'ai', '--non-interactive', '--json']);
  assert.equal(result.code, 0, result.stdout + result.stderr);
  assert.equal(fs.readFileSync(settings, 'utf8'), original);
  assert.equal(fs.existsSync(settings + '.bak'), false);
  const body = JSON.parse(result.stdout);
  assert.equal(body.result.sections.find(s => s.name === 'ai').status, 'skipped');
});
