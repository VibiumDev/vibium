/**
 * JS Library Tests: Async API
 * Tests browser.launch() and async Vibe methods
 */

const { test, describe, after } = require('node:test');
const assert = require('node:assert');
const fs = require('node:fs');
const path = require('node:path');

// Import from built library
const { browser } = require('../../clients/javascript/dist');

describe('JS Async API', () => {
  test('browser.launch() and vibe.quit() work', async () => {
    const vibe = await browser.launch({ headless: true });
    assert.ok(vibe, 'Should return a Vibe instance');
    await vibe.quit();
  });

  test('vibe.go() navigates to URL', async () => {
    const vibe = await browser.launch({ headless: true });
    try {
      await vibe.go('https://the-internet.herokuapp.com/');
      // If we get here without error, navigation worked
      assert.ok(true);
    } finally {
      await vibe.quit();
    }
  });

  test('vibe.screenshot() returns PNG buffer', async () => {
    const vibe = await browser.launch({ headless: true });
    try {
      await vibe.go('https://the-internet.herokuapp.com/');
      const screenshot = await vibe.screenshot();

      assert.ok(Buffer.isBuffer(screenshot), 'Should return a Buffer');
      assert.ok(screenshot.length > 1000, 'Screenshot should have reasonable size');

      // Check PNG magic bytes
      assert.strictEqual(screenshot[0], 0x89, 'Should be valid PNG');
      assert.strictEqual(screenshot[1], 0x50, 'Should be valid PNG');
      assert.strictEqual(screenshot[2], 0x4E, 'Should be valid PNG');
      assert.strictEqual(screenshot[3], 0x47, 'Should be valid PNG');
    } finally {
      await vibe.quit();
    }
  });

  test('vibe.evaluate() executes JavaScript', async () => {
    const vibe = await browser.launch({ headless: true });
    try {
      await vibe.go('https://the-internet.herokuapp.com/');
      const title = await vibe.evaluate('return document.title');
      assert.match(title, /The Internet/i, 'Should return page title');
    } finally {
      await vibe.quit();
    }
  });

  test('vibe.find() locates element', async () => {
    const vibe = await browser.launch({ headless: true });
    try {
      await vibe.go('https://the-internet.herokuapp.com/');
      const heading = await vibe.find('h1.heading');

      assert.ok(heading, 'Should return an Element');
      assert.ok(heading.info, 'Element should have info');
      assert.match(heading.info.tag, /^h1$/i, 'Should be an h1 tag');
      assert.match(heading.info.text, /Welcome to the-internet/i, 'Should have heading text');
    } finally {
      await vibe.quit();
    }
  });

  test('element.click() works', async () => {
    const vibe = await browser.launch({ headless: true });
    try {
      await vibe.go('https://the-internet.herokuapp.com/');
      const link = await vibe.find('a[href="/add_remove_elements/"]');
      await link.click();

      // After click, we should have navigated
      await new Promise(resolve => setTimeout(resolve, 1000));

      const heading = await vibe.find('h3');
      assert.match(heading.info.text, /Add\/Remove Elements/i, 'Should have navigated to new page');
    } finally {
      await vibe.quit();
    }
  });

  test('element.type() enters text', async () => {
    const vibe = await browser.launch({ headless: true });
    try {
      await vibe.go('https://the-internet.herokuapp.com/inputs');
      const input = await vibe.find('input');
      await input.type('12345');

      // Verify the value was entered
      const value = await vibe.evaluate(`
        return document.querySelector('input').value;
      `);
      assert.strictEqual(value, '12345', 'Input should have typed value');
    } finally {
      await vibe.quit();
    }
  });

  test('element.type() enters text with carriage returns', async () => {
    const vibe = await browser.launch({ headless: true });
    try {
      await vibe.go('https://seleniumbase.io/demo_page');
      const textarea = await vibe.find('textarea');
      await textarea.type('123\r456');

      // Verify the value was entered
      const value = await vibe.evaluate(`
        return document.querySelector('textarea').value;
      `);
      assert.strictEqual(value, '123\n456', 'Textarea should have typed value with newline (via carriage return)');
    } finally {
      await vibe.quit();
    }
  });

  test('element.type() enters text with newlines', async () => {
    const vibe = await browser.launch({ headless: true });
    try {
      await vibe.go('https://seleniumbase.io/demo_page');
      const textarea = await vibe.find('textarea');
      await textarea.type('123\n456');

      // Verify the value was entered
      const value = await vibe.evaluate(`
        return document.querySelector('textarea').value;
      `);
      assert.strictEqual(value, '123\n456', 'Textarea should have typed value with newline');
    } finally {
      await vibe.quit();
    }
  });

  test('element.text() returns element text', async () => {
    const vibe = await browser.launch({ headless: true });
    try {
      await vibe.go('https://the-internet.herokuapp.com/');
      const heading = await vibe.find('h1.heading');
      const text = await heading.text();
      assert.match(text, /Welcome to the-internet/i, 'Should return heading text');
    } finally {
      await vibe.quit();
    }
  });
});
