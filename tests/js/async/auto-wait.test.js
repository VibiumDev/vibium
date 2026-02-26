/**
 * JS Library Tests: Auto-Wait Behavior
 * Tests that actions wait for elements to be actionable
 */

const { test, describe, before, after } = require('node:test');
const assert = require('node:assert');

const { browser } = require('../../../clients/javascript/dist');
const { createTestServer } = require('../../helpers/test-server');

let server, baseURL;

before(async () => {
  ({ server, baseURL } = await createTestServer());
});

after(() => {
  if (server) server.close();
});

describe('JS Auto-Wait', () => {
  test('find() waits for element to appear', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(baseURL + '/dynamic_loading/1');

      // Click the start button to trigger dynamic loading
      const startBtn = await vibe.find('#start button', { timeout: 5000 });
      await startBtn.click();

      // find() should wait for the dynamically loaded element
      const result = await vibe.find('#finish h4', { timeout: 10000 });
      assert.ok(result, 'Should find the dynamically loaded element');
      assert.strictEqual(result.info.text, 'Hello World!', 'Should have correct text');
    } finally {
      await bro.close();
    }
  });

  test('click() waits for element to be actionable', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(baseURL + '/add_remove_elements/');

      // Click the "Add Element" button
      const addBtn = await vibe.find('button[onclick="addElement()"]', { timeout: 5000 });
      await addBtn.click({ timeout: 5000 });

      // Verify the delete button appeared
      const deleteBtn = await vibe.find('.added-manually', { timeout: 5000 });
      assert.ok(deleteBtn, 'Delete button should have appeared after click');
    } finally {
      await bro.close();
    }
  });

  test('find() times out for non-existent element', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(baseURL + '/');

      await assert.rejects(
        async () => {
          await vibe.find('#does-not-exist', { timeout: 1000 });
        },
        /timeout/i,
        'Should throw timeout error'
      );
    } finally {
      await bro.close();
    }
  });

  test('timeout error message is clear', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
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
      await bro.close();
    }
  });

  test('navigation error message is clear', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await assert.rejects(
        async () => {
          await vibe.go('https://test.invalid');
        },
        /error/i,
        'Should throw error for invalid domain'
      );
    } finally {
      await bro.close();
    }
  });
});
