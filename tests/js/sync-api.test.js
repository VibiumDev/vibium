/**
 * JS Library Tests: Sync API
 * Tests browserSync.launch() and sync Vibe methods
 */

const { test, describe } = require('node:test');
const assert = require('node:assert');
const fs = require('node:fs');

// Import from built library
const { browserSync } = require('../../clients/javascript/dist');

describe('JS Sync API', () => {
  test('browserSync.launch() and vibe.quit() work', () => {
    const vibe = browserSync.launch({ headless: true });
    assert.ok(vibe, 'Should return a VibeSync instance');
    vibe.quit();
  });

  test('vibe.go() navigates to URL (sync)', () => {
    const vibe = browserSync.launch({ headless: true });
    try {
      vibe.go('https://the-internet.herokuapp.com/');
      // If we get here without error, navigation worked
      assert.ok(true);
    } finally {
      vibe.quit();
    }
  });

  test('vibe.screenshot() returns PNG buffer (sync)', () => {
    const vibe = browserSync.launch({ headless: true });
    try {
      vibe.go('https://the-internet.herokuapp.com/');
      const screenshot = vibe.screenshot();

      assert.ok(Buffer.isBuffer(screenshot), 'Should return a Buffer');
      assert.ok(screenshot.length > 1000, 'Screenshot should have reasonable size');

      // Check PNG magic bytes
      assert.strictEqual(screenshot[0], 0x89, 'Should be valid PNG');
      assert.strictEqual(screenshot[1], 0x50, 'Should be valid PNG');
    } finally {
      vibe.quit();
    }
  });

  test('vibe.evaluate() executes JavaScript (sync)', () => {
    const vibe = browserSync.launch({ headless: true });
    try {
      vibe.go('https://the-internet.herokuapp.com/');
      const title = vibe.evaluate('return document.title');
      assert.match(title, /The Internet/i, 'Should return page title');
    } finally {
      vibe.quit();
    }
  });

  test('vibe.find() locates element (sync)', () => {
    const vibe = browserSync.launch({ headless: true });
    try {
      vibe.go('https://the-internet.herokuapp.com/');
      const heading = vibe.find('h1.heading');

      assert.ok(heading, 'Should return an ElementSync');
      assert.ok(heading.info, 'Element should have info');
      assert.match(heading.info.tag, /^h1$/i, 'Should be an h1 tag');
    } finally {
      vibe.quit();
    }
  });

  test('element.click() works (sync)', () => {
    const vibe = browserSync.launch({ headless: true });
    try {
      vibe.go('https://the-internet.herokuapp.com/');
      const link = vibe.find('a[href="/add_remove_elements/"]');
      link.click();

      // After click, should have navigated
      const heading = vibe.find('h3');
      assert.match(heading.info.text, /Add\/Remove Elements/i, 'Should have navigated');
    } finally {
      vibe.quit();
    }
  });

  test('element.type() enters text (sync)', () => {
    const vibe = browserSync.launch({ headless: true });
    try {
      vibe.go('https://the-internet.herokuapp.com/inputs');
      const input = vibe.find('input');
      input.type('12345');

      // Verify the value was entered
      const value = vibe.evaluate(`
        return document.querySelector('input').value;
      `);
      assert.strictEqual(value, '12345', 'Input should have typed value');
    } finally {
      vibe.quit();
    }
  });

  test('element.type() enters text with carriage returns (sync)', () => {
    const vibe = browserSync.launch({ headless: true });
    try {
      vibe.go('https://seleniumbase.io/demo_page');
      const textarea = vibe.find('textarea');
      textarea.type('123\r456');

      // Verify the value was entered
      const value = vibe.evaluate(`
        return document.querySelector('textarea').value;
      `);
      assert.strictEqual(value, '123\n456', 'Textarea should have typed value with newline (via carriage return)');
    } finally {
      vibe.quit();
    }
  });

  test('element.type() enters text with newlines (sync)', () => {
    const vibe = browserSync.launch({ headless: true });
    try {
      vibe.go('https://seleniumbase.io/demo_page');
      const textarea = vibe.find('textarea');
      textarea.type('123\n456');

      // Verify the value was entered
      const value = vibe.evaluate(`
        return document.querySelector('textarea').value;
      `);
      assert.strictEqual(value, '123\n456', 'Textarea should have typed value with newline');
    } finally {
      vibe.quit();
    }
  });
});
