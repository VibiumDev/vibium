const { test } = require('node:test');
const assert = require('node:assert/strict');
const { execFile } = require('node:child_process');
const { promisify } = require('node:util');
const http = require('node:http');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const exec = promisify(execFile);
const root = path.resolve(__dirname, '../..');
const { ENGINE } = require('../helpers');
const listen = s => new Promise(r => s.listen(0, '127.0.0.1', () => r(`http://127.0.0.1:${s.address().port}`)));

test('Check SDK and MCP surfaces share the native runtime', { timeout: 300000 }, async t => {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'check-surfaces-'));
  const errors = [];
  const provider = http.createServer(async (req, res) => {
    try {
      let raw = ''; for await (const data of req) raw += data;
      const body = JSON.parse(raw);
      const archived = body.tools.every(t => t.function.name.startsWith('trace_'));
      const obs = body.messages.filter(m => m.role === 'tool');
      let message, step;
      if (archived) {
        assert.equal(body.model, 'archive-override');
        assert.equal(body.messages.length, 3);
        message = { role: 'assistant', content: JSON.stringify({ status: 'inconclusive', summary: 'Archive evidence has not established this broad claim.', evidence: [] }) };
      } else {
        assert.ok(!raw.includes('BUILDER-TRANSCRIPT'));
        if (!obs.length) {
          assert.equal(body.messages.length, 5);
          assert.ok(raw.includes('Account'), 'Initial observation must use the pinned page');
          assert.ok(!raw.includes('Other page'), 'Other page must not leak into pinned context');
        }
        const steps = [['browser_get_value', { selector: '#name' }], ['browser_fill', { selector: '#name', text: 'Updated' }], ['browser_click', { selector: 'button' }], ['browser_reload', {}], ['browser_get_value', { selector: '#name' }], ['browser_screenshot', {}], ['browser_console', {}], ['browser_network', {}]];
        step = steps[obs.length];
        if (!step) {
          assert.ok(obs[4].content.includes('Updated'), 'Saved value must survive reload');
          assert.ok(body.messages.some(m => Array.isArray(m.content) && m.content.some(p => p.type === 'image_url')), 'Screenshot tool must return an image');
          message = { role: 'assistant', content: JSON.stringify({ status: 'passed', summary: 'The changed display name persisted after refresh.', evidence: [{ type: 'observation', summary: 'The value was Updated after reload.' }] }) };
        }
      }
      if (step) message = { role: 'assistant', content: null, tool_calls: [{ id: `call${obs.length}`, type: 'function', function: { name: step[0], arguments: JSON.stringify(step[1]) } }] };
      res.setHeader('Content-Type', 'application/json');
      res.end(JSON.stringify({ choices: [{ finish_reason: step ? 'tool_calls' : 'stop', message }] }));
    } catch (err) { errors.push(err.stack); res.writeHead(500); res.end('Fixture assertion failed'); }
  });
  const app = http.createServer((req, res) => {
    res.setHeader('Content-Type', 'text/html');
    if (req.url === '/other') { res.end('<h1>Other page</h1><input id="name" value="Other">'); return; }
    res.end(`<!doctype html><title>Account</title><h1>Account</h1><form><label>Display name<input id="name"></label><button>Save</button></form><script>const input=document.querySelector('input');input.value=localStorage.getItem('name')||'Original';document.querySelector('form').onsubmit=e=>{e.preventDefault();localStorage.setItem('name',input.value);console.log('Saved')};</script>`);
  });
  const env = { ...process.env, VIBIUM_BIN_PATH: path.join(root, 'clicker/bin/vibium'), VIBIUM_CONNECT_URL: '', VIBIUM_ENGINE: ENGINE, VIBIUM_ENGINE_PATH: '', VIBIUM_ENGINE_CHANNEL: '', VIBIUM_AI_PROVIDER: 'openai-compatible', VIBIUM_AI_MODEL: 'fixture', OPENAI_API_KEY: 'fixture-secret', VIBIUM_AI_BASE_URL: `${await listen(provider)}/v1`, CHECK_TEST_URL: await listen(app), CHECK_TEST_INPUT: path.join(root, 'tests/fixtures/check/playwright-checkout.zip') };
  try {
    const runners = [
      ['JavaScript async', process.execPath, [path.join(__dirname, 'sdk-js.cjs')], '0'],
      ['JavaScript sync', process.execPath, [path.join(__dirname, 'sdk-js.cjs')], '1'],
      ['Python async', path.join(root, process.platform === 'win32' ? 'clients/python/.venv/Scripts/python.exe' : 'clients/python/.venv/bin/python'), [path.join(__dirname, 'sdk-python.py')], '0'],
      ['Python sync', path.join(root, process.platform === 'win32' ? 'clients/python/.venv/Scripts/python.exe' : 'clients/python/.venv/bin/python'), [path.join(__dirname, 'sdk-python.py')], '1'],
      ['Java', 'java', ['-cp', path.join(root, 'clients/java/build/libs/*') + path.delimiter + path.join(root, 'clients/java/build/dependencies/*'), path.join(__dirname, 'CheckSDK.java')], '0'],
      ['MCP', process.execPath, [path.join(__dirname, 'mcp.cjs')], '0'],
    ];
    for (const [label, binary, args, sync] of runners) {
      for (const archived of ['0', '1']) {
        await t.test(`${label} ${archived === '1' ? 'archive without browser' : 'live pinned page'}`, async () => {
          const output = path.join(dir, `result-${label.replaceAll(" ", "-")}-${archived}.zip`);
          const localEnv = { ...env, CHECK_TEST_ENGINE: ENGINE, CHECK_TEST_SYNC: sync, CHECK_TEST_ARCHIVE_ONLY: archived, CHECK_TEST_OUTPUT: output };
          if (archived === '1') { localEnv.VIBIUM_ENGINE = 'firefox'; localEnv.VIBIUM_ENGINE_PATH = '/browser-must-not-launch'; }
          try { await exec(binary, args, { env: localEnv, timeout: 90000 }); }
          catch (err) { t.diagnostic(errors.join('\n')); throw err; }
          if (archived === '0') {
            const { stdout } = await exec('unzip', ['-p', output, 'trace.trace'], { maxBuffer: 16*1024*1024 });
            const events = stdout.trim().split('\n').map(JSON.parse);
            const parent = events.find(e => e.type === 'before' && e.params?.method === 'vibium:check.run');
            assert.ok(parent);
            assert.equal(events.find(e => e.type === 'after' && e.callId === parent.callId).result.status, 'passed');
            assert.ok(events.some(e => e.type === 'before' && e.parentId === parent.callId && e.method === 'vibium:element.fill'));
            assert.ok(!stdout.includes('fixture-secret'));
          }
        });
      }
    }
  } finally {
    for (const s of [provider, app]) { s.closeAllConnections(); await new Promise(r => s.close(r)); }
    fs.rmSync(dir, { recursive: true, force: true });
  }
});
