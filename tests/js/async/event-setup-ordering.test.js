/**
 * JS Library Tests: event-setup ordering (#351)
 *
 * page.onWebSocket() is a void callback API, so its install command had nothing
 * for the caller to await and was fired and forgotten — a socket opened by the
 * very next command could beat the install and its one-shot event was lost.
 *
 * helpers/fake-engine.js stands in for the binary: it installs slowly and only
 * reports a socket once the install is answered, so this asserts ordering rather
 * than how fast a browser happens to be.
 */

const { test, describe } = require('node:test');
const assert = require('node:assert');
const path = require('path');

const { browser } = require('../../../clients/javascript/dist');

const FAKE_ENGINE = path.join(__dirname, '..', 'helpers', 'fake-engine.js');

// The stand-in is a #! script, which Windows cannot spawn as a binary.
const describeFn = process.platform === 'win32' ? describe.skip : describe;

async function withFakeEngine(fn) {
  process.env.VIBIUM_BIN_PATH = FAKE_ENGINE;
  const bro = await browser.start({ headless: true });
  try {
    await fn(await bro.page());
  } finally {
    await bro.stop();
    delete process.env.VIBIUM_BIN_PATH;
  }
}

describeFn('Event setup ordering: page.onWebSocket', () => {
  test('a socket opened by the very next command is still seen', async () => {
    await withFakeEngine(async (vibe) => {
      const seen = [];
      vibe.onWebSocket((ws) => seen.push(ws.url()));
      // No await in between — the exact shape that used to lose the event.
      await vibe.evaluate('openSocket()');

      assert.deepStrictEqual(seen, ['ws://127.0.0.1:1/live'],
        'the command overtook the monitor install, so its socket went unseen');
    });
  });

  test('registering twice installs the monitor once', async () => {
    await withFakeEngine(async (vibe) => {
      const seen = [];
      vibe.onWebSocket(() => seen.push('a'));
      vibe.onWebSocket(() => seen.push('b'));
      await vibe.evaluate('openSocket()');

      // A second install would have re-armed the gate and restarted the delay.
      assert.deepStrictEqual(seen, ['a', 'b']);
    });
  });
});
