/**
 * What the JavaScript client sends for say and listen.
 *
 * A fake transport stands in for the engine, so no browser, engine or
 * audio is needed.
 */

const { test } = require('node:test');
const assert = require('node:assert/strict');
const path = require('node:path');
const { Browser, Page, BiDiError, SpeechError, TimeoutError } = require('../../clients/javascript/dist');
const sync = require('../../clients/javascript/dist/sync');

function wire(answer = () => ({})) {
  const calls = [];
  return { calls, onEvent() {}, send(method, params, timeout) { calls.push({ method, params, timeout }); return Promise.resolve().then(() => answer(method)); } };
}

test('say and listen go out as the speech wire commands', async () => {
  const transport = wire();
  const browser = new Browser(transport, null), page = new Page(transport, 'page-7');

  await page.say('What time is it?', { voice: 'cedar', speed: 1.5, cache: false, language: undefined });
  assert.deepEqual(transport.calls.pop(), {
    method: 'vibium:speech.say',
    params: { text: 'What time is it?', voice: 'cedar', speed: 1.5, cache: false, context: 'page-7' },
    timeout: 180000,
  });

  await browser.say('Hello');
  assert.deepEqual(transport.calls.pop().params, { text: 'Hello' }, 'a Browser names no page: the engine uses the current one');

  await page.listen({ silence: 0, allowSilence: true, transcribe: false });
  assert.deepEqual(transport.calls.pop(), {
    method: 'vibium:speech.listen',
    params: { silence: 0, allowSilence: true, transcribe: false, context: 'page-7' },
    timeout: 100000,
  });

  await browser.listen({ timeout: 5000, duration: 20000 });
  const listen = transport.calls.pop();
  assert.deepEqual(listen.params, { timeout: 5000, duration: 20000 });
  assert.equal(listen.timeout, 85000, 'the client waits timeout + duration + 60000 ms');
});

test('paths go to the engine absolute, and named targets as they are', async () => {
  const transport = wire();
  const page = new Page(transport, 'page-7');
  const abs = (name) => path.resolve(name);

  await page.say('', { file: 'clip.wav', target: 'out/said.WAV', cacheDir: 'cache' });
  assert.deepEqual(transport.calls.pop().params, { file: abs('clip.wav'), target: abs('out/said.WAV'), cacheDir: abs('cache'), context: 'page-7' });

  await page.listen({ target: 'reply.wav', path: 'heard.wav' });
  assert.deepEqual(transport.calls.pop().params, { target: abs('reply.wav'), path: abs('heard.wav'), context: 'page-7' });

  for (const target of ['session', 'local', 'speakers']) {
    await page.listen({ target });
    assert.equal(transport.calls.pop().params.target, target);
  }
});

test('a failure with a speech code is a SpeechError, and any other failure is not', async () => {
  const failing = (error) => new Page(wire(() => { throw error; }), 'page-7');

  for (const call of [(p) => p.say('Hello'), (p) => p.listen()]) {
    await assert.rejects(call(failing(new BiDiError('no_speech', 'listened for 10.0 s and heard no speech'))), (err) => {
      assert.ok(err instanceof SpeechError);
      assert.equal(err.name, 'SpeechError');
      assert.equal(err.code, 'no_speech');
      assert.equal(err.message, 'listened for 10.0 s and heard no speech');
      return true;
    });
    await assert.rejects(call(failing(new BiDiError('error', 'unsupported argument pitch'))), (err) => {
      assert.ok(err instanceof BiDiError && !(err instanceof SpeechError));
      return true;
    });
    await assert.rejects(call(failing(new TimeoutError("Command 'vibium:speech.listen' timed out after 100000ms"))), TimeoutError);
  }
});

test('the sync client passes say and listen to its worker and outwaits it', () => {
  const calls = [];
  const bridge = { call(method, args, timeout) { calls.push({ method, args, timeout }); return {}; } };
  const browser = new sync.BrowserSync(bridge), page = new sync.PageSync(bridge, 7);

  page.say('Hello', { voice: 'cedar' });
  assert.deepEqual(calls.pop(), { method: 'page.say', args: [7, 'Hello', { voice: 'cedar' }], timeout: 185000 });
  browser.say('Hello');
  assert.deepEqual(calls.pop(), { method: 'browser.say', args: ['Hello', {}], timeout: 185000 });
  page.listen({ timeout: 5000, duration: 20000 });
  assert.deepEqual(calls.pop(), { method: 'page.listen', args: [7, { timeout: 5000, duration: 20000 }], timeout: 90000 });
  browser.listen();
  assert.deepEqual(calls.pop(), { method: 'browser.listen', args: [{}], timeout: 105000 });
});

test('both clients export the same speech surface', () => {
  for (const [B, P] of [[Browser, Page], [sync.BrowserSync, sync.PageSync]]) {
    for (const C of [B, P]) {
      assert.equal(typeof C.prototype.say, 'function');
      assert.equal(typeof C.prototype.listen, 'function');
    }
  }
  for (const lib of [require('../../clients/javascript/dist'), sync]) {
    assert.equal(typeof lib.browser.say, 'function');
    assert.equal(typeof lib.browser.listen, 'function');
    assert.equal(typeof lib.SpeechError, 'function');
  }
});
