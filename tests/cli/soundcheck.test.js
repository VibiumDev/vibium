/** Verifier setup checks must work without a daemon or browser installation. */
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
    VIBIUM_SESSION: 'soundcheck', VIBIUM_ENGINE: 'chrome', VIBIUM_ENGINE_PATH: '',
    VIBIUM_ENGINE_CHANNEL: '', VIBIUM_CONNECT_URL: '',
    VIBIUM_VERIFIER_PROVIDER: '', VIBIUM_VERIFIER_MODEL: '', OPENAI_API_KEY: '',
    VIBIUM_VERIFIER_BASE_URL: '', VIBIUM_VERIFIER_REASONING_EFFORT: '', ...extra,
  };
}
async function run(env, args = ['soundcheck', '--json']) {
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

test('soundcheck lists missing settings and detects an unsourced env file without reading it', async t => {
  const env = environment(t);
  const settings = path.join(env.HOME, '.config', 'vibium', 'verifier.env');
  fs.mkdirSync(path.dirname(settings), { recursive: true });
  fs.writeFileSync(settings, 'OPENAI_API_KEY=secret-file-marker\nVIBIUM_VERIFIER_PROVIDER=openai\n', { mode: 0o600 });
  const before = fs.readFileSync(settings);
  for (const args of [['soundcheck'], ['soundcheck', '--json']]) {
    const result = await run(env, args);
    assert.equal(result.code, 1);
    assert.match(result.stdout, /VIBIUM_VERIFIER_PROVIDER/);
    assert.match(result.stdout, /VIBIUM_VERIFIER_MODEL/);
    assert.match(result.stdout, /source ~\//);
    assert.doesNotMatch(result.stdout + result.stderr, /secret-file-marker/);
    if (args.includes('--json')) {
      const body = JSON.parse(result.stdout);
      assert.equal(body.ok, false);
      assert.equal(body.result.ready, false);
      assert.equal(body.result.checks.find(c => c.name === 'provider').status, 'skipped');
    }
    noBrowser(env);
  }
  assert.deepEqual(fs.readFileSync(settings), before);
});

test('soundcheck exercises the actual provider transport and reports readiness in text and JSON', async t => {
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
  const env = environment(t, { VIBIUM_VERIFIER_PROVIDER: 'openai-compatible',
    VIBIUM_VERIFIER_BASE_URL: fixture.url, VIBIUM_VERIFIER_MODEL: 'fixture-model',
    VIBIUM_VERIFIER_REASONING_EFFORT: 'none', OPENAI_API_KEY: 'secret-api-marker' });
  for (const args of [['soundcheck'], ['soundcheck', '--json']]) {
    const result = await run(env, args);
    assert.equal(result.code, 0, result.stdout + result.stderr);
    assert.doesNotMatch(result.stdout + result.stderr, /secret-api-marker|PRIVATE-REASONING/);
    if (args.includes('--json')) {
      const body = JSON.parse(result.stdout);
      assert.equal(body.ok, true);
      assert.equal(body.result.ready, true);
      assert.ok(body.result.checks.every(c => c.status === 'passed'));
    } else { assert.match(result.stdout, /READY: verifier configuration/); }
    noBrowser(env);
  }
  assert.equal(fixture.requests(), 4);
});

test('provider authentication failure gives a fix, exit 1, and no leaked response body', async t => {
  const fixture = await provider(t, (req, res) => {
    res.writeHead(401);
    res.end(JSON.stringify({ error: { message: 'secret-api-marker', code: 'secret-api-marker', param: 'secret-api-marker' } }));
  });
  const env = environment(t, { VIBIUM_VERIFIER_PROVIDER: 'openai-compatible',
    VIBIUM_VERIFIER_BASE_URL: fixture.url, VIBIUM_VERIFIER_MODEL: 'fixture-model' });
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

test('verify setup errors point to soundcheck and help runs without setup', async t => {
  const env = environment(t);
  const failure = await run(env, ['verify', 'the cart works', '--json']);
  assert.equal(failure.code, 1);
  assert.match(JSON.parse(failure.stdout).error, /vibium soundcheck/);
  const help = await run(env, ['soundcheck', '--help']);
  assert.equal(help.code, 0);
  assert.match(help.stdout, /API charges may apply/);
  noBrowser(env);
});
