/** Live browsers with a deterministic provider: ownership, recording, and errors. */
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

async function lifecycle(t, { status = 'passed', keepOpen = false, existing = false, idleDaemon = false, mismatch = false, record = true } = {}) {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'check-lifecycle-'));
  let requests = 0;
  const provider = http.createServer(async (req, res) => {
    try {
      if (req.method === 'GET') { res.setHeader('Content-Type', 'text/html'); res.end('<h1>Lifecycle fixture</h1>'); return; }
      let data = ''; for await (const chunk of req) data += chunk;
      const body = JSON.parse(data);
      requests++;
      assert.equal(body.messages[1].content, 'the fixture is visible');
      assert.equal(body.tools.some(tool => /stop|launch|start/.test(tool.function.name)), false);
      if (status === 'error') { res.writeHead(401); res.end('{}'); return; }
      res.setHeader('Content-Type', 'application/json');
      res.end(JSON.stringify({ choices: [{ finish_reason: 'stop', message: {
        role: 'assistant', content: JSON.stringify({ status, summary: 'Fixture verdict.',
          evidence: [{ type: 'observation', summary: 'Browser observations received.' }] }),
      } }] }));
    } catch (error) { t.diagnostic(error.stack); res.writeHead(500); res.end('{}'); }
  });
  await new Promise(resolve => provider.listen(0, '127.0.0.1', resolve));
  const env = { ...process.env, VIBIUM_SESSION: `vl-${process.pid}`,
    VIBIUM_ENGINE: process.env.VIBIUM_CHECK_ENGINE || 'chrome', VIBIUM_ENGINE_PATH: '',
    VIBIUM_ENGINE_CHANNEL: process.env.VIBIUM_CHECK_ENGINE === 'firefox' ? 'beta' : '',
    VIBIUM_CONNECT_URL: '', VIBIUM_AI_PROVIDER: 'openai-compatible',
    VIBIUM_AI_MODEL: 'fixture', VIBIUM_AI_REASONING_EFFORT: '',
    VIBIUM_AI_BASE_URL: `http://127.0.0.1:${provider.address().port}/v1`, OPENAI_API_KEY: '' };
  const cli = async (...args) => JSON.parse((await exec(VIBIUM, ['--json', '--headless', ...args], { env, timeout: 120000 })).stdout);
  try {
    // A bare daemon must not be mistaken for an already-started browser.
    if (idleDaemon) await exec(VIBIUM, ['--headless', 'daemon', 'start'], { env, timeout: 15000 });
    let pages;
    const original = path.join(dir, 'builder.zip');
    if (existing) {
      await cli('go', `http://127.0.0.1:${provider.address().port}/fixture`);
      await cli('eval', "window.name='owned-by-builder'; sessionStorage.setItem('sentinel','preserved'); 'ready'");
      pages = (await cli('pages')).result;
      await cli('record', 'start', '-o', original);
      await cli('record', 'group', 'start', 'Builder');
    }
    const recording = path.join(dir, 'verification.zip');
    const report = path.join(dir, 'verdict.json');
    const args = ['check', 'the fixture is visible', '--report', report];
    if (record) args.push('-o', recording);
    if (keepOpen) args.push('--keep-open');
    if (mismatch) args.push('--headless=false');
    let result;
    if (status === 'error' || mismatch) {
      await assert.rejects(cli(...args), error => {
        const response = JSON.parse(error.stdout);
        assert.equal(response.ok, false);
        assert.match(response.error, mismatch ? /already running/ : /HTTP 401/);
        return true;
      });
      assert.equal(fs.existsSync(report), false);
    } else {
      result = (await cli(...args)).result;
      assert.equal(result.status, status);
      assert.deepEqual(JSON.parse(fs.readFileSync(report, 'utf8')), result);
    }
    assert.equal(requests, mismatch ? 0 : 1);
    if (!mismatch && record) {
      // Evidence must be finalized before Check returns and before cleanup.
      const trace = (await exec('unzip', ['-p', recording, 'trace.trace'], { maxBuffer: 12 * 1024 * 1024 })).stdout;
      const events = trace.trim().split('\n').map(JSON.parse);
      const parent = events.find(e => e.type === 'before' && e.params?.method === 'vibium:check.run');
      assert.ok(parent, 'missing Check recording group');
      const end = events.find(e => e.type === 'after' && e.callId === parent.callId);
      assert.ok(end, 'recording not finalized');
      if (status !== 'error') assert.equal(end.result.status, status);
      const entries = (await exec('unzip', ['-Z1', recording])).stdout.trim().split('\n');
      const videos = entries.filter(name => /^video\/.+\.webm$/.test(name));
      if (env.VIBIUM_ENGINE === 'firefox' && !existing) {
        assert.ok(videos.length > 0, 'Check-owned Firefox recording lost its WebM track');
        const manifest = JSON.parse((await exec('unzip', ['-p', recording, 'video/index.json'])).stdout);
        assert.equal(manifest.version, 1);
        assert.ok(manifest.videos.length > 0, 'missing video manifest entries');
        assert.ok(manifest.videos.every(track => track.mimeType === 'video/webm' && entries.includes(track.file)), 'video manifest must point to delivered WebM files');
      } else {
        assert.equal(videos.length, 0, 'Chrome or an ongoing recording chunk should not contain native video');
      }
    }
    if (existing) {
      assert.equal((await cli('eval', 'window.name')).result, 'owned-by-builder');
      assert.equal((await cli('eval', "sessionStorage.getItem('sentinel')")).result, 'preserved');
      assert.equal((await cli('pages')).result, pages);
      assert.equal(fs.existsSync(original), false, 'stopped the builder recording');
      await cli('record', 'group', 'stop');
      await cli('record', 'stop');
      assert.ok(fs.statSync(original).size > 0);
    }
    // stop does not autostart a browser. Its response distinguishes an
    // already-cleaned browser from one deliberately kept alive.
    const stopped = (await cli('stop')).result;
    assert.match(stopped, existing || keepOpen ? /Browser session closed/ : /No browser session to close/);
  } finally {
    await exec(VIBIUM, ['daemon', 'stop'], { env, timeout: 15000 }).catch(() => {});
    await new Promise(resolve => provider.close(resolve));
    fs.rmSync(dir, { recursive: true, force: true });
  }
}

for (const status of ['passed', 'failed', 'inconclusive', 'error']) {
  test(`Check closes its new browser after ${status} and saves evidence`, t => lifecycle(t, { status, idleDaemon: status === 'inconclusive' }));
  test(`Check --keep-open retains its new browser after ${status}`, t => lifecycle(t, { status, keepOpen: true }));
}
for (const status of ['passed', 'error']) {
  test(`Check preserves an existing browser and recording after ${status}`, t => lifecycle(t, { status, existing: true }));
}
test('Check preserves the existing browser when launch flags conflict', t => lifecycle(t, { existing: true, mismatch: true }));

test('Check also closes its new browser without --output', t => lifecycle(t, { record: false }));
