/**
 * JS Library Tests: Error types
 * Engine errors map onto the exported error classes (ElementNotFoundError,
 * TimeoutError, BiDiError), matching the Python client's contract.
 */

const { test, describe, before, after } = require("../../../helpers/capabilities").suite("core");
const assert = require('node:assert');

const { browser, ElementNotFoundError, TimeoutError, BiDiError } = require('../../../../clients/javascript/dist');

let bro, vibe;

before(async () => {
  bro = await browser.start({ headless: true });
  vibe = await bro.page();
});

after(async () => {
  if (bro) await bro.stop();
});

describe('Error types', () => {
  test('error classes are exported and structured', () => {
    for (const cls of [ElementNotFoundError, TimeoutError, BiDiError]) {
      assert.strictEqual(typeof cls, 'function', 'error class should be exported');
    }
    const err = new BiDiError('invalid argument', 'bad value');
    assert.ok(err instanceof Error);
    assert.strictEqual(err.error, 'invalid argument');
    assert.match(err.message, /invalid argument: bad value/);
  });

  test('find() on a missing element rejects with ElementNotFoundError', async () => {
    await vibe.setContent('<p>nothing else here</p>');
    await assert.rejects(
      () => vibe.find({ css: '#never-existed' }, { timeout: 500 }),
      (err) => {
        assert.ok(
          err instanceof ElementNotFoundError,
          `expected ElementNotFoundError, got ${err.constructor.name}: ${err.message}`
        );
        return true;
      }
    );
  });

  test('a pure wait timeout rejects with TimeoutError', async () => {
    await vibe.setContent('<p>static page</p>');
    await assert.rejects(
      () => vibe.waitUntil('() => false', { timeout: 500 }),
      (err) => {
        assert.ok(
          err instanceof TimeoutError,
          `expected TimeoutError, got ${err.constructor.name}: ${err.message}`
        );
        return true;
      }
    );
  });
});
