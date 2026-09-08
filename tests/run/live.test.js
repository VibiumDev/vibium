/** Opt-in provider acceptance. The normal suite uses deterministic provider fixtures. */
const { test } = require('node:test');
const assert = require('node:assert/strict');
const { execFile } = require('node:child_process');
const { promisify } = require('node:util');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const { VIBIUM } = require('../helpers');
const { fixture } = require('./fixture.cjs');
const exec = promisify(execFile);

async function acceptance(t, existing) {
  const app = await fixture();
  const dir = process.env.VIBIUM_AI_ARTIFACT_DIR || fs.mkdtempSync(path.join(os.tmpdir(), 'run-live-'));
  fs.mkdirSync(dir, { recursive: true });
  const env = { ...process.env, VIBIUM_SESSION: `run-live-${process.pid}`, VIBIUM_ENGINE: 'chrome', VIBIUM_ENGINE_PATH: '', VIBIUM_ENGINE_CHANNEL: '', VIBIUM_CONNECT_URL: '' };
  const overrides = process.env.VIBIUM_AI_LIVE_OVERRIDES === '1' ? [
    '--provider', env.VIBIUM_AI_PROVIDER, '--model', env.VIBIUM_AI_MODEL,
    '--base-url', env.VIBIUM_AI_BASE_URL || '', '--reasoning-effort', env.VIBIUM_AI_REASONING_EFFORT || '',
  ] : [];
  if (overrides.length) {
    env.VIBIUM_AI_PROVIDER = 'invalid-default';
    env.VIBIUM_AI_MODEL = 'invalid-default';
    env.VIBIUM_AI_BASE_URL = 'http://127.0.0.1:1/v1';
    env.VIBIUM_AI_REASONING_EFFORT = 'invalid-default';
  }
  const cli = async (...args) => JSON.parse((await exec(VIBIUM, ['--json', '--headless', ...args], { env, timeout: 230000, maxBuffer: 8*1024*1024 })).stdout).result;
  const output = path.join(dir, existing ? 'existing.zip' : 'standalone.zip');
  try {
    assert.ok(env.VIBIUM_AI_PROVIDER && env.VIBIUM_AI_MODEL, 'Configure the chosen Run provider and model');
    assert.equal((await cli('ready', 'ai', ...overrides)).ready, true);
    if (existing) {
      await cli('go', app.url);
      await cli('eval', "window.name='original';sessionStorage.setItem('builder','preserved');'ready'");
    }
    const goal = `${existing ? 'On the current Account page' : `Open ${app.url}`}, change the display name to Phase Two Test and save it. Confirm the page reports the saved name.`;
    const result = await cli('run', goal, '-o', output, ...overrides);
    fs.writeFileSync(path.join(dir, existing ? 'existing-result.json' : 'standalone-result.json'), JSON.stringify(result, null, 2));
    assert.equal(result.status, 'completed'); assert.equal(result.goal, goal); assert.ok(result.evidence.length);
    if (existing) {
      assert.equal(await cli('eval', "localStorage.getItem('name')"), 'Phase Two Test');
      assert.equal(await cli('eval', 'window.name'), 'original');
      assert.equal(await cli('eval', "sessionStorage.getItem('builder')"), 'preserved');
    }
    const trace = (await exec('unzip', ['-p', output, 'trace.trace'], { maxBuffer: 16*1024*1024 })).stdout;
    const events = trace.trim().split('\n').map(JSON.parse);
    const parent = events.find(e => e.type === 'before' && e.params?.method === 'vibium:run.run');
    assert.ok(parent); assert.equal(events.find(e => e.type === 'after' && e.callId === parent.callId).result.status, 'completed');
    assert.equal(parent.params.modelConfig.provider, process.env.VIBIUM_AI_PROVIDER);
    assert.equal(parent.params.modelConfig.model, process.env.VIBIUM_AI_MODEL);
    assert.ok(events.some(e => e.type === 'before' && e.parentId === parent.callId && /fill|type/.test(e.method)));
    for (const key of ['OPENAI_API_KEY', 'ANTHROPIC_API_KEY', 'GOOGLE_API_KEY']) if (env[key]) assert.ok(!trace.includes(env[key]), 'provider credential appeared in trace');
    assert.match(await cli('stop'), existing ? /Browser session closed/ : /No browser session to close/);
    t.diagnostic(`${result.status}: ${result.summary}`);
  } finally {
    await exec(VIBIUM, ['daemon', 'stop'], { env, timeout: 15000 }).catch(() => {});
    await app.close();
    if (!process.env.VIBIUM_AI_ARTIFACT_DIR) fs.rmSync(dir, { recursive: true, force: true });
  }
}
for (const provider of ['openai', 'anthropic', 'google', 'local']) {
  const enabled = process.env.VIBIUM_AI_LIVE === '1' && process.env.VIBIUM_AI_PROVIDER === provider;
  test(`Real ${provider} Run reuses and records the current browser`, { timeout: 300000, skip: enabled ? false : 'Pending opt-in provider environment' }, t => acceptance(t, true));
  if (provider === 'openai') test('Real OpenAI standalone Run saves evidence and closes its browser', { timeout: 300000, skip: enabled ? false : 'Pending opt-in OpenAI environment' }, t => acceptance(t, false));
}
