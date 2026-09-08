const { test } = require('node:test');
const assert = require('node:assert/strict');
const { execFile } = require('node:child_process');
const { promisify } = require('node:util');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const { VIBIUM, ENGINE } = require('../helpers');
const { fixture } = require('./fixture.cjs');
const exec = promisify(execFile);

async function run(t, options = {}) {
  const f = await fixture(), dir = fs.mkdtempSync(path.join(os.tmpdir(), 'run-cli-'));
  const env = { ...process.env, ...f.env, VIBIUM_SESSION: `run-${process.pid}`, VIBIUM_ENGINE: ENGINE, VIBIUM_ENGINE_CHANNEL: '', VIBIUM_ENGINE_PATH: '', VIBIUM_CONNECT_URL: '', VIBIUM_AI_PROVIDER: options.provider || 'openai-compatible' };
  if (env.VIBIUM_AI_PROVIDER === 'openai') env.OPENAI_API_KEY = 'openai-test-key';
  const cli = async (...args) => JSON.parse((await exec(VIBIUM, ['--json', '--headless', ...args], { env, timeout: 230000, maxBuffer: 8*1024*1024 })).stdout).result;
  const output = path.join(dir, 'run.zip'), original = path.join(dir, 'builder.zip');
  try {
    assert.equal((await cli('ready', 'ai')).ready, true);
    if (options.existing) {
      await cli('go', f.url);
      await cli('eval', "window.name='original';sessionStorage.setItem('builder','preserved');localStorage.setItem('builder','preserved');document.cookie='sentinel=preserved';'ready'");
      await cli('record', 'start', '-o', original);
      await cli('record', 'group', 'start', 'Builder');
    }
    if (options.privacy) await cli('eval', "const p=document.createElement('input');p.type='password';p.id='password';document.body.append(p);'ready'");
    if (options.error) env.VIBIUM_AI_MODEL = 'error-model';
    const args = ['run', options.privacy ? 'credential test' : options.incomplete ? 'not possible' : 'change name', '-o', output];
    if (options.keepOpen) args.push('--keep-open');
    if (options.error) await assert.rejects(cli(...args), e => /HTTP 401/.test(e.stdout) && !e.stdout.includes('SECRET-ERROR-BODY'));
    else {
      const result = await cli(...args);
      assert.equal(result.status, options.incomplete ? 'not_completed' : 'completed');
      assert.equal(result.goal, args[1]); assert.equal(result.claim, undefined);
    }
    if (options.existing) {
      assert.equal(await cli('eval', 'window.name'), 'original');
      assert.equal(await cli('eval', "sessionStorage.getItem('builder')"), 'preserved');
      assert.equal(await cli('eval', "localStorage.getItem('builder')"), 'preserved');
      assert.match(await cli('eval', 'document.cookie'), /sentinel=preserved/);
      assert.equal(fs.existsSync(original), false);
      await cli('record', 'group', 'stop'); await cli('record', 'stop');
      if (!options.error && !options.privacy) {
        env.VIBIUM_AI_PROVIDER = options.provider === 'anthropic' ? 'google' : 'openai-compatible';
        assert.equal((await cli('check', 'the name persisted')).status, 'passed');
      }
    }
    const trace = (await exec('unzip', ['-p', output, 'trace.trace'], { maxBuffer: 12*1024*1024 })).stdout;
    const events = trace.trim().split('\n').map(JSON.parse);
    const parent = events.find(e => e.type === 'before' && e.params?.method === 'vibium:run.run');
    assert.ok(parent); assert.ok(events.some(e => e.type === 'after' && e.callId === parent.callId));
    if (!options.error && !options.incomplete) assert.ok(events.some(e => e.type === 'before' && e.parentId === parent.callId && /fill/.test(e.method)));
    assert.ok(!trace.includes('native-key') && !trace.includes('PRIVATE-REASONING'));
    const names = (await exec('unzip', ['-Z1', output])).stdout;
    if (options.privacy) { assert.ok(!trace.includes('TEST-PASSWORD-SECRET')); assert.ok(events[0].vibiumPrivacy); assert.ok(!names.includes('video/')); }
    else if (ENGINE === 'firefox' && !options.existing) assert.match(names, /video\/.+\.webm/);
    assert.match(await cli('stop'), options.existing || options.keepOpen ? /Browser session closed/ : /No browser session to close/);
  } finally {
    await exec(VIBIUM, ['daemon', 'stop'], { env, timeout: 15000 }).catch(() => {});
    fs.rmSync(dir, { recursive: true, force: true }); await f.close();
  }
}
for (const provider of ['openai', 'anthropic', 'google', 'openai-compatible', 'local']) test(`Run CLI uses ${provider} browser tools and shared AI configuration`, { timeout: 120000 }, t => run(t, { provider, existing: true }));
for (const state of [{}, { incomplete: true }, { error: true }, { keepOpen: true }, { keepOpen: true, error: true }, { existing: true, error: true }, { existing: true, privacy: true }]) test(`Run ownership and evidence ${JSON.stringify(state)}`, { timeout: 120000 }, t => run(t, state));
test('Run rejects archive/report flags and requires shared AI configuration', async () => {
  // Old per-feature variables must not silently configure the shared model loop.
  for (const args of [['run', 'goal', '-i', 'record.zip'], ['run', 'goal', '--report', 'result.json'], ['run', 'goal']]) {
    await assert.rejects(exec(VIBIUM, args, { env: { ...process.env, VIBIUM_AI_PROVIDER: '', VIBIUM_AI_MODEL: '', VIBIUM_VERIFIER_PROVIDER: 'openai', VIBIUM_PERFORM_MODEL: 'should-not-be-used' } }), e => /unknown (shorthand )?flag|VIBIUM_AI_PROVIDER/.test(e.stderr));
  }
});
