/**
 * JS Library Tests: Auto-Wait Behavior
 * Tests that actions wait for elements to be actionable
 */

const { test, describe, before, after } = require('node:test');
const assert = require('node:assert');

const { browser } = require('../../../clients/javascript/dist');
const { createTestServer } = require('../../helpers/test-server');

let server, baseURL, bro;

before(async () => {
  ({ server, baseURL } = await createTestServer());
  bro = await browser.start({ headless: true });
});

after(async () => {
  if (bro) await bro.stop();
  if (server) server.close();
});

describe('JS Auto-Wait', () => {
  test('find() waits for element to appear', async () => {
    const vibe = await bro.newPage();
    try {
      await vibe.go(baseURL + '/dynamic_loading/1');

      // Click the start button to trigger dynamic loading
      const startBtn = await vibe.find('#start button', { timeout: 5000 });
      await startBtn.click();

      // find() should wait for the dynamically loaded element
      const result = await vibe.find('#finish h4', { timeout: 10000 });
      assert.ok(result, 'Should find the dynamically loaded element');
      assert.strictEqual(result.info.text, 'Hello World!', 'Should have correct text');
    } finally {
      await vibe.close();
    }
  });

  test('click() waits for element to be actionable', async () => {
    const vibe = await bro.newPage();
    try {
      await vibe.go(baseURL + '/add_remove_elements/');

      // Click the "Add Element" button
      const addBtn = await vibe.find('button[onclick="addElement()"]', { timeout: 5000 });
      await addBtn.click({ timeout: 5000 });

      // Verify the delete button appeared
      const deleteBtn = await vibe.find('.added-manually', { timeout: 5000 });
      assert.ok(deleteBtn, 'Delete button should have appeared after click');
    } finally {
      await vibe.close();
    }
  });

  test('find() times out for non-existent element', async () => {
    const vibe = await bro.newPage();
    try {
      await vibe.go(baseURL + '/');

      await assert.rejects(
        async () => {
          await vibe.find('#does-not-exist', { timeout: 1000 });
        },
        /timeout/i,
        'Should throw timeout error'
      );
    } finally {
      await vibe.close();
    }
  });

  test('timeout error message is clear', async () => {
    const vibe = await bro.newPage();
    try {
      await vibe.go(baseURL + '/');

      try {
        await vibe.find('#nonexistent-element-xyz', { timeout: 1000 });
        assert.fail('Should have thrown');
      } catch (err) {
        // Error should mention the selector or timeout
        assert.ok(
          err.message.includes('timeout') || err.message.includes('#nonexistent-element-xyz'),
          `Error message should be clear: ${err.message}`
        );
      }
    } finally {
      await vibe.close();
    }
  });

  test('navigation error message is clear', async () => {
    const vibe = await bro.newPage();
    try {
      // Use a guaranteed-fast failure (port 1, connection refused) rather than
      // a fake DNS name — DNS lookup for .invalid is fast on most systems but
      // can stall for tens of seconds on networks with custom resolvers.
      await assert.rejects(
        async () => {
          await vibe.go('http://127.0.0.1:1');
        },
        /error|refused|fail/i,
        'Should throw error for unreachable URL'
      );
    } finally {
      await vibe.close();
    }
  });
});
