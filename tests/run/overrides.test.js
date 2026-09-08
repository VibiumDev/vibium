const { test } = require('node:test');
const assert = require('node:assert/strict');
const { execFile } = require('node:child_process');
const { promisify } = require('node:util');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const { fixture } = require('./fixture.cjs');
const { VIBIUM } = require('../helpers');
const exec = promisify(execFile);

test('CLI overrides are per call across Run, Check, archives, and ready ai', { timeout: 120000 }, async () => {
  const f = await fixture(), dir = fs.mkdtempSync(path.join(os.tmpdir(), 'model-overrides-'));
  const env = { ...process.env, ...f.env, VIBIUM_SESSION: `overrides-${process.pid}`, VIBIUM_ENGINE: 'chrome', VIBIUM_ENGINE_PATH: '', VIBIUM_ENGINE_CHANNEL: '', VIBIUM_CONNECT_URL: '', VIBIUM_AI_MODEL: 'error-model' };
  const cli = async (...args) => JSON.parse((await exec(VIBIUM, ['--headless', '--json', ...args], { env, timeout: 90000, maxBuffer: 8*1024*1024 })).stdout).result;
  const options = (provider, model) => ['--provider', provider, '--model', model, '--base-url', f.endpoint + '/v1', '--reasoning-effort', ''];
  try {
    for (const cmd of ['run', 'check']) await assert.rejects(cli(cmd, 'claim', '--provider', 'anthropic'), err => /MODEL is required/.test(err.stdout));
    assert.match(await cli('stop'), /No browser session/);
    assert.equal((await cli('ready', 'ai', ...options('anthropic', 'run-model'))).ready, true);
    assert.equal((await cli('ready', 'ai', ...options('google', 'check-model'))).ready, true);
    assert.match(await cli('stop'), /No browser session/);
    await cli('go', f.url);
    const output = path.join(dir, 'run.zip');
    await cli('record', 'start', '-o', output);
    assert.equal((await cli('run', 'change name', ...options('anthropic', 'run-model'))).status, 'completed');
    assert.equal((await cli('check', 'the name persisted', ...options('google', 'check-model'))).status, 'passed');
    // A later call still uses the original model, which deliberately returns 401.
    await assert.rejects(cli('run', 'change name'), err => /HTTP 401/.test(err.stdout));
    await assert.rejects(cli('check', 'the name persisted'), err => /HTTP 401/.test(err.stdout));
    await cli('record', 'stop');
    await cli('stop');
    const input = path.resolve(__dirname, '../fixtures/check/playwright-checkout.zip');
    assert.equal((await cli('check', 'archive evidence', '-i', input, ...options('google', 'check-model'))).status, 'inconclusive');
    assert.match(await cli('stop'), /No browser session/);
    const trace = (await exec('unzip', ['-p', output, 'trace.trace'], { maxBuffer: 16*1024*1024 })).stdout;
    const parents = trace.trim().split('\n').map(JSON.parse).filter(e => e.type === 'before' && e.params?.modelConfig);
    assert.deepEqual(parents.slice(0, 2).map(e => e.params.modelConfig.provider), ['anthropic', 'google']);
    assert.deepEqual(parents.slice(0, 2).map(e => e.params.modelConfig.model), ['run-model', 'check-model']);
    assert.ok(parents.every(e => e.params.modelConfig.apiKey === undefined && e.params.modelConfig.baseURL === undefined));
    assert.ok(!trace.includes('native-key'));
  } finally {
    await exec(VIBIUM, ['daemon', 'stop'], { env }).catch(() => {});
    fs.rmSync(dir, { recursive: true, force: true });
    await f.close();
  }
});
