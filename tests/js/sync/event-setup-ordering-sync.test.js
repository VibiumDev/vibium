/**
 * Sync JS Library Tests: event-setup ordering (#351)
 *
 * The sync API has no promise for the caller to hold, so its blocking
 * onWebSocket() must not return until the engine has acknowledged the install —
 * and must raise if the install was rejected, the only place a sync caller can
 * see it. See ../helpers/fake-engine.js.
 */

const { test, describe } = require('node:test');
const assert = require('node:assert');
const path = require('path');

const { browser } = require('../../../clients/javascript/dist/sync');

const FAKE_ENGINE = path.join(__dirname, '..', 'helpers', 'fake-engine.js');

// The stand-in is a #! script, which Windows cannot spawn as a binary.
const describeFn = process.platform === 'win32' ? describe.skip : describe;

function withFakeEngine(fn) {
  process.env.VIBIUM_BIN_PATH = FAKE_ENGINE;
  const bro = browser.start({ headless: true });
  try {
    fn(bro.page());
  } finally {
    bro.stop();
    delete process.env.VIBIUM_BIN_PATH;
  }
}

describeFn('Sync event setup ordering: page.onWebSocket', () => {
  test('a socket opened by the very next call is still seen', () => {
    withFakeEngine((vibe) => {
      const seen = [];
      vibe.onWebSocket((ws) => seen.push(ws.url()));
      vibe.evaluate('openSocket()');
      // Drain: the event reached the worker with the response above.
      vibe.evaluate('1');

      assert.deepStrictEqual(seen, ['ws://127.0.0.1:1/live'],
        'onWebSocket returned before the install was acknowledged, so the socket went unseen');
    });
  });

  test('a rejected install raises from onWebSocket', () => {
    process.env.FAKE_ENGINE_FAIL_SETUP = '1';
    try {
      withFakeEngine((vibe) => {
        assert.throws(() => vibe.onWebSocket(() => {}), /no preload scripts here/,
          'a rejected install must reach the sync caller instead of being swallowed');
      });
    } finally {
      delete process.env.FAKE_ENGINE_FAIL_SETUP;
    }
  });
});
