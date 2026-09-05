/** First Verify slice: real Chrome and CLI/daemon/recording, with either a
 * deterministic OpenAI-compatible fixture (default) or a real BYO model
 * (VIBIUM_VERIFY_LIVE=1, OPENAI_API_KEY, VIBIUM_VERIFIER_MODEL). */
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
const live = process.env.VIBIUM_VERIFY_LIVE === '1';

function listen(server) {
  return new Promise(resolve => server.listen(0, '127.0.0.1', () => resolve(`http://127.0.0.1:${server.address().port}`)));
}

async function acceptance(t, broken = false) {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'vibium-verify-'));
  const record = process.env.VIBIUM_VERIFY_RECORD || path.join(dir, 'record.zip');
  const session = `verify-${process.pid}-${broken ? 'fail' : 'pass'}`;
  const output = path.join(dir, 'verification.zip');
  const report = path.join(dir, 'verdict.json');
  const claim = 'changing my display name persists after refresh';
  const env = { ...process.env, VIBIUM_SESSION: session, VIBIUM_CONNECT_URL: '', VIBIUM_ENGINE: process.env.VIBIUM_VERIFY_ENGINE || 'chrome', VIBIUM_ENGINE_PATH: '', VIBIUM_ENGINE_CHANNEL: process.env.VIBIUM_VERIFY_ENGINE === 'firefox' ? 'beta' : '' };
  const cli = async (...args) => {
    const { stdout } = await exec(VIBIUM, ['--json', '--headless', ...args], { env, timeout: 230000, maxBuffer: 4 * 1024 * 1024 });
    const out = JSON.parse(stdout); assert.equal(out.ok, true); return out.result;
  };
  const app = http.createServer((req, res) => {
    res.setHeader('Content-Type', 'text/html');
    res.setHeader('X-API-Key', 'APP-HEADER-SECRET');
    res.end(`<!doctype html><title>Account settings</title><h1>Account</h1>
      <form><label for="display-name">Display name</label><input id="display-name" name="displayName">
      <button type="submit">Save</button></form><p role="status" id="status"></p>
      <script>
        const input = document.querySelector('input');
        input.value = localStorage.getItem('displayName') || 'Jason';
        localStorage.setItem('loadHistory', JSON.stringify([...JSON.parse(localStorage.getItem('loadHistory') || '[]'), input.value]));
        document.querySelector('form').onsubmit = e => {
          e.preventDefault();
          ${broken ? '' : "localStorage.setItem('displayName', input.value);"}
          document.querySelector('#status').textContent = 'Saved display name: ' + input.value;
          console.log('Display name saved');
        };
      </script>`);
  });
  const base = await listen(app);
  let modelRequests = 0;
  let recordingActive = false;
  const provider = http.createServer(async (req, res) => {
    try {
      let data = ''; for await (const chunk of req) data += chunk;
      const body = JSON.parse(data);
      assert.equal(req.url, '/v1/chat/completions');
      assert.equal(body.model, 'fixture-model');
      assert.equal(req.headers.authorization, 'Bearer fixture-secret');
      assert.ok(!data.includes('builder-conversation-marker'));
      assert.ok(!data.includes('PRIVATE-REASONING-MARKER'));
      if (modelRequests === 0) {
        assert.equal(body.messages.length, 5);
        assert.equal(body.messages[0].role, 'system');
        assert.equal(body.messages[1].content, claim);
        assert.ok(data.includes(base));
      }
      const tools = body.tools.map(t => t.function.name);
      assert.ok(!tools.includes('browser_evaluate') && !tools.includes('browser_stop'));
      const steps = [
        ['browser_press', { key: 'x' }],
        ['browser_get_value', { selector: '#password' }],
        ['browser_get_value', { selector: '#display-name' }],
        ['browser_fill', { selector: '#display-name', text: 'Jason Test' }],
        ['browser_click', { selector: 'button' }],
        ['browser_reload', {}],
        ['browser_get_value', { selector: '#display-name' }],
        ['browser_screenshot', {}],
        ['browser_console', {}],
        ['browser_network', {}],
      ];
      const step = steps[modelRequests++];
      let message;
      if (step) {
        message = { role: 'assistant', content: 'PRIVATE-REASONING-MARKER', reasoning_content: 'PRIVATE-REASONING-MARKER', tool_calls: [{ id: `call${modelRequests}`, type: 'function', function: { name: step[0], arguments: JSON.stringify(step[1]) } }] };
      } else {
        const observations = body.messages.filter(m => m.role === 'tool');
        assert.match(observations[0].content, /Password fields are unavailable/);
        assert.match(observations[1].content, /Password fields are unavailable/);
        const observed = observations[6].content;
        assert.ok(observed.includes(broken ? 'Jason' : 'Jason Test'));
        message = { role: 'assistant', content: JSON.stringify({ status: broken ? 'failed' : 'passed', summary: broken ? 'The display name reverted after refresh.' : 'The new display name persisted after refresh.', evidence: [{ type: 'observation', summary: `Display name before edit: Jason. After refresh: ${broken ? 'Jason' : 'Jason Test'}.` }] }) };
      }
      res.setHeader('Content-Type', 'application/json');
      res.end(JSON.stringify({ choices: [{ finish_reason: step ? 'tool_calls' : 'stop', message }] }));
    } catch (error) { res.writeHead(500); res.end('Fixture failed'); t.diagnostic(error.message); }
  });
  if (!live) {
    env.VIBIUM_VERIFIER_PROVIDER = 'openai-compatible';
    env.VIBIUM_VERIFIER_MODEL = 'fixture-model';
    env.OPENAI_API_KEY = 'fixture-secret';
    env.VIBIUM_VERIFIER_BASE_URL = `${await listen(provider)}/v1`;
  } else {
    assert.ok(env.OPENAI_API_KEY && env.VIBIUM_VERIFIER_MODEL, 'Real-model test needs configured key/model');
    env.VIBIUM_VERIFIER_PROVIDER ||= 'openai';
  }
  try {
    // Start the daemon without provider config, as if configured after building.
    await exec(VIBIUM, ['--headless', 'start'], { env: { ...env, OPENAI_API_KEY: '', VIBIUM_VERIFIER_MODEL: '', VIBIUM_VERIFIER_PROVIDER: '' }, timeout: 90000 });
    await cli('go', `${base}/account`);
    await cli('eval', `window.name='existing-vibium-tab'; sessionStorage.setItem('builderSession','session-sentinel'); localStorage.setItem('builderState','storage-sentinel'); document.cookie='builderCookie=cookie-sentinel'; 'ready'`);
    if (!live) {
      await cli('eval', `const password = document.createElement('input'); password.id='password'; password.type='password'; password.value='PASSWORD-SENTINEL'; document.body.append(password); password.focus(); 'ready'`);
    }
    const pagesBefore = await cli('pages');
    if (!broken) {
      await cli('record', 'start', '-o', record);
      recordingActive = true;
      await cli('record', 'group', 'start', 'Builder workflow');
    }
    const result = await cli('verify', claim, '-o', output, '--report', report);
    assert.deepEqual(JSON.parse(fs.readFileSync(report, 'utf8')), result);
    assert.ok(fs.statSync(output).size > 0);
    const { stdout: exported } = await exec('unzip', ['-p', output, 'trace.trace'], { maxBuffer: 12 * 1024 * 1024 });
    const outputEvents = exported.trim().split('\n').map(JSON.parse);
    const outputParent = outputEvents.find(e => e.type === 'before' && e.params?.method === 'vibium:verify.run');
    assert.ok(outputParent);
    assert.deepEqual(outputEvents.find(e => e.type === 'after' && e.callId === outputParent.callId).result, result);
    t.diagnostic(`${live ? 'Real model' : 'Fixture'}: ${result.status}: ${result.summary}`);
    assert.equal(result.status, broken ? 'failed' : 'passed');
    assert.equal(result.claim, claim);
    assert.ok(result.summary && result.evidence.length);
    assert.equal(await cli('eval', 'window.name'), 'existing-vibium-tab');
    assert.equal(await cli('eval', "sessionStorage.getItem('builderSession')"), 'session-sentinel');
    assert.equal(await cli('eval', "localStorage.getItem('builderState')"), 'storage-sentinel');
    assert.ok((await cli('eval', 'document.cookie')).includes('builderCookie=cookie-sentinel'));
    assert.equal(await cli('pages'), pagesBefore);
    if (!broken) {
      const current = await cli('value', '#display-name');
      // A real verifier may restore the original name after testing. Assert
      // that a changed value actually survived a page load, not that it left
      // the test edit in place at the end.
      const loaded = JSON.parse(await cli('eval', "localStorage.getItem('loadHistory')"));
      assert.ok(loaded.some(name => name !== 'Jason'), 'A changed display name must survive a page load');
      assert.equal(await cli('eval', "localStorage.getItem('displayName')"), current);
    }
    if (!broken) {
      await cli('record', 'group', 'stop');
      await cli('record', 'stop');
      recordingActive = false;
    } else {
      // Verify must release the recorder it started.
      await cli('record', 'start', '--video=false', '-o', record);
      recordingActive = true;
      await cli('record', 'stop');
      recordingActive = false;
    }
    const traceFile = broken ? output : record;
    const { stdout: trace } = await exec('unzip', ['-p', traceFile, 'trace.trace'], { maxBuffer: 12 * 1024 * 1024 });
    const { stdout: network } = await exec('unzip', ['-p', traceFile, 'trace.network'], { maxBuffer: 12 * 1024 * 1024 });
    assert.ok(!network.includes('APP-HEADER-SECRET'), 'Response API key must be masked');
    assert.ok(!network.includes('cookie-sentinel'), 'Session cookie must be masked');
    const events = trace.trim().split('\n').map(JSON.parse);
    assert.equal(events[0].version, 8);
    const parent = events.find(e => e.type === 'before' && e.params?.method === 'vibium:verify.run');
    assert.ok(parent);
    if (!broken) {
      assert.ok(parent.parentId, 'Verify nested under builder group');
      assert.ok(events.length > outputEvents.length, 'Existing recording continues collecting actions after Verify output');
      assert.ok(events.some(e => e.type === 'after' && e.callId === parent.parentId), 'Builder group remains open until caller closes it');
    } else {
      assert.ok(!parent.parentId);
    }
    assert.equal(parent.params.claim, claim);
    assert.equal(parent.method, 'tracingGroup');
    const children = events.filter(e => e.type === 'before' && e.parentId === parent.callId);
    assert.ok(children.some(e => e.method === 'vibium:element.fill'));
    assert.ok(children.some(e => e.method === 'vibium:element.click'));
    assert.ok(children.some(e => /reload|navigate/.test(e.method)));
    if (live) assert.ok(events.some(e => e.type === 'screencast-frame'));
    else {
      assert.ok(!events.some(e => e.type === 'screencast-frame' || e.type === 'frame-snapshot'));
      assert.ok(events[0].vibiumPrivacy, 'Sensitive form fields suppress visual artifacts');
    }
    const end = events.find(e => e.type === 'after' && e.callId === parent.callId);
    assert.deepEqual(end.result, result);
    assert.ok(parent.title.includes(broken ? 'FAIL' : 'PASS'));
    assert.ok(!trace.includes(env.OPENAI_API_KEY), 'Provider key must not appear in trace');
    assert.ok(!trace.includes('PASSWORD-SENTINEL'), 'Password must not appear in trace');
    assert.ok(!trace.includes('PRIVATE-REASONING-MARKER'), 'Reasoning must not appear in trace');
    assert.ok(children.every(c => events.some(e => e.type === 'after' && e.callId === c.callId)));
    if (process.env.VIBIUM_VERIFY_RECORD) t.diagnostic(`Recording: ${record}`);
  } finally {
    if (recordingActive) await cli('record', 'stop').catch(() => {});
    await exec(VIBIUM, ['daemon', 'stop'], { env, timeout: 15000 }).catch(() => {});
    app.closeAllConnections(); await new Promise(resolve => app.close(resolve));
    if (!live) { provider.closeAllConnections(); await new Promise(resolve => provider.close(resolve)); }
    fs.rmSync(dir, { recursive: true, force: true });
  }
}

test(`Verify reuses live ${process.env.VIBIUM_VERIFY_ENGINE || 'chrome'} and records persistence evidence`, { timeout: 300000 }, t => acceptance(t));
test('Verify reports a persistence regression', { timeout: 300000, skip: live }, t => acceptance(t, true));

test('Verify saves error evidence and preserves recording ownership', { timeout: 120000, skip: live }, async t => {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'vibium-verify-error-'));
  const env = { ...process.env, VIBIUM_SESSION: `verify-error-${process.pid}`, VIBIUM_CONNECT_URL: '', VIBIUM_ENGINE: 'chrome', VIBIUM_ENGINE_PATH: '', VIBIUM_ENGINE_CHANNEL: '', VIBIUM_VERIFIER_PROVIDER: 'openai-compatible', VIBIUM_VERIFIER_MODEL: 'fixture', OPENAI_API_KEY: 'fixture-secret' };
  let calls = 0;
  const provider = http.createServer((req, res) => { calls++; req.resume(); res.writeHead(503); res.end('SECRET-ERROR-BODY'); });
  env.VIBIUM_VERIFIER_BASE_URL = `${await listen(provider)}/v1`;
  const app = http.createServer((req, res) => { res.setHeader('Content-Type', 'text/html'); res.end('<h1>Account</h1>'); });
  const base = await listen(app);
  const cli = async (...args) => JSON.parse((await exec(VIBIUM, ['--json', '--headless', ...args], { env, timeout: 45000 })).stdout).result;
  try {
    await cli('go', base);
    for (const active of [false, true]) {
      const original = path.join(dir, `workflow-${active}.zip`);
      const output = path.join(dir, `verify-${active}.zip`);
      const report = path.join(dir, `verdict-${active}.json`);
      if (active) {
        await cli('record', 'start', '--video=false', '-o', original);
        const before = calls;
        await assert.rejects(cli('verify', 'claim', '-o', original), e => { assert.match(e.stdout, /active recording/); return true; });
        assert.equal(calls, before);
        assert.ok(!fs.existsSync(original));
      }
      await assert.rejects(cli('verify', 'The Account heading is visible', '-o', output, '--report', report), e => {
        assert.match(e.stdout, /503/);
        assert.ok(!e.stdout.includes('SECRET-ERROR-BODY')); return true;
      });
      assert.ok(!fs.existsSync(report), 'No verdict report for an operational error');
      const { stdout } = await exec('unzip', ['-p', output, 'trace.trace']);
      const events = stdout.trim().split('\n').map(JSON.parse);
      const parent = events.find(e => e.type === 'before' && e.params?.method === 'vibium:verify.run');
      assert.ok(parent);
      const end = events.find(e => e.type === 'after' && e.callId === parent.callId);
      assert.ok(end.error && !end.result, 'Error is recorded separately from a verdict');
      assert.ok(events.some(e => e.type === 'before' && e.parentId === parent.callId), 'Initial inspection remains available');
      assert.ok(!stdout.includes('fixture-secret') && !stdout.includes('SECRET-ERROR-BODY'));
      // The caller's recorder remains active, or Verify's recorder is released.
      if (!active) await cli('record', 'start', '--video=false', '-o', original);
      await cli('record', 'stop');
      assert.ok(fs.existsSync(original));
    }
  } finally {
    await exec(VIBIUM, ['daemon', 'stop'], { env, timeout: 15000 }).catch(() => {});
    for (const server of [app, provider]) { server.closeAllConnections(); await new Promise(resolve => server.close(resolve)); }
    fs.rmSync(dir, { recursive: true, force: true });
  }
});
