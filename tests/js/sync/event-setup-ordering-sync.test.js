/**
 * Sync JS Library Tests: event-setup ordering (#351)
 *
 * The sync API has no promise for the caller to hold, so its blocking
 * onWebSocket() must not return until the engine has acknowledged the
 * install, and must raise if the install was rejected, the only place a
 * sync caller can see it. See ../helpers/fake-engine.js.
 */

const { test, describe } = require('node:test');
const assert = require('node:assert');
const path = require('path');

const { browser } = require('../../../clients/javascript/dist/sync');

const FAKE_ENGINE = path.join(__dirname, '..', 'helpers', 'fake-engine.js');

// The stand-in is a shebang script, which Windows cannot spawn as a binary.
const describeFn = process.platform === 'win32' ? describe.skip : describe;

function withFakeEngine(fn) {
  const bro = browser.start({ headless: true, executablePath: FAKE_ENGINE });
  try {
    fn(bro.page());
  } finally {
    bro.stop();
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
    // The stand-in reads this at startup, so set it before launching. Safe to
    // mutate here: node --test gives each test file its own process.
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

  test('catching a rejected install and retrying works', () => {
    process.env.FAKE_ENGINE_FAIL_SETUP = 'once';
    try {
      withFakeEngine((vibe) => {
        const seen = [];
        const callback = (ws) => seen.push(ws.url());
        assert.throws(() => vibe.onWebSocket(callback), /no preload scripts here/);
        vibe.onWebSocket(callback);
        vibe.evaluate('openSocket()');
        // Drain: the event reached the worker with the response above.
        vibe.evaluate('1');

        // Exactly once: the retry re-sent the install, and the raised call
        // left no duplicate registration behind.
        assert.deepStrictEqual(seen, ['ws://127.0.0.1:1/live']);
        assert.strictEqual(vibe.evaluate('__installCount'), 2,
          'the retry must re-send the install command');
      });
    } finally {
      delete process.env.FAKE_ENGINE_FAIL_SETUP;
    }
  });
});
