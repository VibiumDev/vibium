// CLI archive acceptance uses a genuine Playwright fixture and a recording
// generated through Vibium's existing local browser and recorder.
const { test } = require('node:test');
const assert = require('node:assert/strict');
const { execFile } = require('node:child_process');
const { promisify } = require('node:util');
const { createHash } = require('node:crypto');
const http = require('node:http');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const { VIBIUM } = require('../helpers');
const exec = promisify(execFile);
const listen = server => new Promise(resolve => server.listen(0, '127.0.0.1', () => resolve(`http://127.0.0.1:${server.address().port}`)));
const hash = file => createHash('sha256').update(fs.readFileSync(file)).digest('hex');

test('Check input reads Vibium and Playwright archives without a browser and writes only explicit reports', { timeout: 120000 }, async t => {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'vibium-check-archive-'));
  const env = { ...process.env, VIBIUM_SESSION: `check-archives-${process.pid}`, VIBIUM_CONNECT_URL: '', VIBIUM_ENGINE: 'chrome', VIBIUM_ENGINE_PATH: '', VIBIUM_ENGINE_CHANNEL: '', VIBIUM_AI_PROVIDER: 'openai-compatible', VIBIUM_AI_MODEL: 'fixture', OPENAI_API_KEY: 'fixture-secret' };
  let calls = 0;
  let failure = false;
  const provider = http.createServer(async (req, res) => {
    try {
      calls++;
      if (failure) { res.writeHead(503); res.end('SECRET-PROVIDER-ERROR'); return; }
      let raw = ''; for await (const chunk of req) raw += chunk;
      const body = JSON.parse(raw);
      const live = body.tools.some(t => t.function.name.startsWith('browser_'));
      if (!live) assert.ok(body.tools.every(t => t.function.name.startsWith('trace_')), 'Archive tools must be read-only');
      assert.ok(!raw.includes('fixture-secret'));
      const observations = body.messages.filter(m => m.role === 'tool');
      const claim = body.messages[1].content;
      let message;
      let step;
      if (live) {
        assert.ok(body.messages[1].content.startsWith('fixture generation'));
        if (observations.length === 0) step = ['browser_get_text', { selector: 'h1' }];
        else message = { role: 'assistant', content: JSON.stringify({ status: 'passed', summary: 'Order confirmed is visible.', evidence: [{ type: 'observation', summary: observations[0].content }] }) };
      } else if (observations.length === 0) {
        assert.equal(body.messages.length, 3, 'Fresh archive context has only instructions, claim, and summary');
        assert.ok(!raw.includes('Order confirmed'), 'Page contents are not included before inspection');
        step = [claim.startsWith('vibium:') ? 'trace_list_actions' : 'trace_list_snapshots', {}];
      } else if (observations.length === 1) {
        const items = JSON.parse(observations[0].content).items;
        assert.ok(items.length);
        if (claim.startsWith('vibium:')) {
          const action = items.find(e => e.method === 'vibium:element.text');
          assert.ok(action, 'Vibium records the verifier text observation');
          step = ['trace_inspect_action', { id: action.id }];
        } else step = ['trace_inspect_snapshot', { id: items.at(-1).id }];
      } else {
        assert.ok(observations[1].content.includes('Order confirmed'), 'Inspection resolves real recorded DOM evidence');
        const status = claim.includes('missing') ? 'inconclusive' : claim.includes('failed') ? 'failed' : 'passed';
        message = { role: 'assistant', content: JSON.stringify({ status, summary: status === 'inconclusive' ? 'Payment settlement is not captured.' : 'The recorded page shows Order confirmed.', evidence: [{ type: 'observation', summary: 'The final recorded DOM contains the Order confirmed heading.' }] }) };
      }
      if (step) message = { role: 'assistant', content: null, tool_calls: [{ id: `call-${observations.length}`, type: 'function', function: { name: step[0], arguments: JSON.stringify(step[1]) } }] };
      res.setHeader('Content-Type', 'application/json');
      res.end(JSON.stringify({ choices: [{ finish_reason: step ? 'tool_calls' : 'stop', message }] }));
    } catch (e) { t.diagnostic(e.stack); res.writeHead(500); res.end('Fixture failed'); }
  });
  env.VIBIUM_AI_BASE_URL = `${await listen(provider)}/v1`;
  const app = http.createServer((req, res) => {
    res.setHeader('Content-Type', 'text/html');
    res.end('<!doctype html><h1>Checkout</h1><button onclick="document.querySelector(\'h1\').textContent=\'Order confirmed\'">Place order</button>');
  });
  const base = await listen(app);
  const cli = async (...args) => JSON.parse((await exec(VIBIUM, ['--json', '--headless', ...args], { env, cwd: dir, timeout: 45000 })).stdout).result;
  const reject = async (args, pattern) => {
    const before = calls;
    await assert.rejects(cli(...args), e => { assert.match(e.stdout + e.stderr, pattern); return true; });
    assert.equal(calls, before, 'Invalid input must fail before model inference');
  };
  try {
    const vibium = path.join(dir, 'vibium.zip');
    await cli('go', base);
    await cli('record', 'start', '--video=false', '--snapshots', '-o', vibium);
    await cli('click', 'button');
    await cli('check', 'fixture generation: the order confirmation heading is visible');
    await cli('record', 'stop');
    await exec(VIBIUM, ['daemon', 'stop'], { env, timeout: 15000 });
    // Accepted engine settings with an impossible executable: any attempt to
    // install or start a live browser during archive verification would fail.
    env.VIBIUM_ENGINE = 'firefox';
    env.VIBIUM_ENGINE_PATH = path.join(dir, 'browser-must-not-launch');
    env.VIBIUM_SKIP_BROWSER_DOWNLOAD = '1';
    const playwright = path.resolve(__dirname, '../fixtures/check/playwright-checkout.zip');
    for (const input of [vibium, playwright]) {
      const originalHash = hash(input);
      for (const [flag, status] of [['-i', 'passed'], ['--input', 'failed'], ['--input', 'inconclusive']]) {
        const report = path.join(dir, `report-${path.basename(input)}-${status}.json`);
        const claim = `${input === vibium ? 'vibium:' : 'playwright:'} ${status === 'inconclusive' ? 'missing payment settlement' : status + ': inspect the confirmation heading'}`;
        const before = fs.readdirSync(dir).sort();
        const result = await cli('check', claim, flag, input, '--report', report);
        assert.equal(result.status, status);
        assert.deepEqual(JSON.parse(fs.readFileSync(report)), result);
        assert.deepEqual(fs.readdirSync(dir).sort(), [...before, path.basename(report)].sort());
        assert.equal(hash(input), originalHash);
      }
      const before = fs.readdirSync(dir).sort();
      await cli('check', `${input === vibium ? 'vibium:' : 'playwright:'} inspect the confirmation heading`, '-i', input);
      assert.deepEqual(fs.readdirSync(dir).sort(), before, 'No artifact written by default');
      await reject(['check', 'claim', '-i', input, '-o', path.join(dir, 'derived.zip')], /cannot be combined/);
      await reject(['check', 'claim', '-i', input, '--report', input], /already exists/);
      assert.equal(hash(input), originalHash);
    }
    const invalid = path.join(dir, 'invalid.zip'); fs.writeFileSync(invalid, 'not a zip');
    await reject(['check', 'claim', '-i', invalid], /archive|zip/);
    await reject(['check', 'claim', '--record', vibium], /unknown flag/);
    await reject(['check', 'claim', '--trace', playwright], /unknown flag/);
    failure = true;
    const report = path.join(dir, 'failed-report.json');
    await assert.rejects(cli('check', 'claim', '-i', playwright, '--report', report), e => {
      assert.ok(!e.stdout.includes('SECRET-PROVIDER-ERROR'));
      assert.match(e.stdout, /503/); return true;
    });
    assert.ok(!fs.existsSync(report), 'Provider error must not leave an empty verdict report');
  } finally {
    await exec(VIBIUM, ['daemon', 'stop'], { env, timeout: 15000 }).catch(() => {});
    for (const server of [app, provider]) { server.closeAllConnections(); await new Promise(resolve => server.close(resolve)); }
    fs.rmSync(dir, { recursive: true, force: true });
  }
});
