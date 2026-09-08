const { test } = require('node:test');
const assert = require('node:assert/strict');
const { execFile } = require('node:child_process');
const { promisify } = require('node:util');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const { fixture } = require('./fixture.cjs');
const { ENGINE } = require('../helpers');
const exec = promisify(execFile), root = path.resolve(__dirname, '../..');

test('Run surfaces share tools, recording, and shared AI configuration', { timeout: 300000 }, async t => {
  const f = await fixture(), dir = fs.mkdtempSync(path.join(os.tmpdir(), 'run-surfaces-'));
  const base = { ...process.env, ...f.env, VIBIUM_BIN_PATH: path.join(root, 'clicker/bin/vibium'), VIBIUM_ENGINE: ENGINE, VIBIUM_ENGINE_CHANNEL: '', VIBIUM_ENGINE_PATH: '', VIBIUM_CONNECT_URL: '', RUN_TEST_URL: f.url };
  try {
    const cases = [
      ['JavaScript async', process.execPath, ['sdk-js.cjs'], '0', 'anthropic'],
      ['JavaScript privacy', process.execPath, ['sdk-js.cjs'], '0', 'google'],
      ['JavaScript sync', process.execPath, ['sdk-js.cjs'], '1', 'google'],
      ['Python async', path.join(root, process.platform === 'win32' ? 'clients/python/.venv/Scripts/python.exe' : 'clients/python/.venv/bin/python'), ['sdk-python.py'], '0', 'openai-compatible'],
      ['Python sync', path.join(root, process.platform === 'win32' ? 'clients/python/.venv/Scripts/python.exe' : 'clients/python/.venv/bin/python'), ['sdk-python.py'], '1', 'anthropic'],
      ['Java', 'java', ['-cp', path.join(root, 'clients/java/build/libs/*') + path.delimiter + path.join(root, 'clients/java/build/dependencies/*'), 'RunSDK.java'], '0', 'google'],
      ['MCP', process.execPath, ['mcp.cjs'], '0', 'anthropic'],
    ];
    for (const [name, bin, args, sync, provider] of cases) await t.test(name, async () => {
      const output = path.join(dir, `${name}.zip`);
      const env = { ...base, RUN_TEST_OUTPUT: output, RUN_TEST_SYNC: sync, RUN_TEST_PRIVACY: name.includes('privacy') ? '1' : '0', RUN_TEST_ENGINE: ENGINE, VIBIUM_AI_PROVIDER: provider };
      try { await exec(bin, args.map(a => /\.(cjs|py|java)$/.test(a) ? path.join(__dirname, a) : a), { env, timeout: 120000 }); }
      catch (err) { t.diagnostic(f.errors.join('\n')); throw err; }
      const trace = (await exec('unzip', ['-p', output, 'trace.trace'], { maxBuffer: 16*1024*1024 })).stdout;
      const events = trace.trim().split('\n').map(JSON.parse);
      const parent = events.find(e => e.type === 'before' && e.params?.method === 'vibium:run.run');
      assert.ok(parent);
      const configs = events.filter(e => e.type === 'before' && e.params?.method === 'vibium:run.run').map(e => e.params.modelConfig);
      assert.equal(configs[0].provider, provider);
      assert.equal(configs[0].model, 'check-model');
      assert.equal(configs[0].maxActions, 24);
      assert.equal(configs[0].apiKey, undefined);
      assert.equal(configs[0].baseURL, undefined);
      if (!name.includes('privacy')) {
        assert.equal(configs[1].provider, provider === 'anthropic' ? 'google' : 'anthropic');
        assert.equal(configs.at(-1).provider, provider); // Override did not mutate runtime defaults.
      }

      assert.equal(events.find(e => e.type === 'after' && e.callId === parent.callId).result.status, 'completed');
      assert.ok(events.some(e => e.type === 'before' && e.parentId === parent.callId && e.method === 'vibium:element.fill'));
      if (name.includes('privacy')) {
        assert.ok(events[0].vibiumPrivacy);
        assert.ok(!trace.includes('TEST-PASSWORD-SECRET'));
        const entries = (await exec('unzip', ['-Z1', output])).stdout;
        assert.ok(!entries.includes('resources/') && !entries.includes('video/'));
      } else assert.ok(events.some(e => e.type === 'before' && e.params?.method === 'vibium:check.run'));
      assert.ok(!trace.includes('native-key') && !trace.includes('PRIVATE-REASONING') && !trace.includes('opaque-'));
    });
  } finally { fs.rmSync(dir, { recursive: true, force: true }); await f.close(); }
});
