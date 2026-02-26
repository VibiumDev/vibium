/**
 * JS Library Tests: Keyboard, Mouse, Screenshots, Evaluation
 * Tests page.keyboard, page.mouse, page.screenshot (options),
 * page.pdf, page.eval, page.evalHandle, page.addScript, page.addStyle, page.expose.
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

// --- Keyboard, Mouse ---

describe('Keyboard: page-level input', () => {
  test('keyboard.type() types text into focused input', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(`${baseURL}/login`);

      // Click the input to focus it
      const input = await vibe.find('#username');
      await input.click();

      // Type via page.keyboard
      await vibe.keyboard.type('tomsmith');

      const val = await input.value();
      assert.strictEqual(val, 'tomsmith');
    } finally {
      await bro.close();
    }
  });

  test('keyboard.press() sends a key press', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(`${baseURL}/login`);

      const input = await vibe.find('#username');
      await input.click();
      await vibe.keyboard.type('hello');

      // Press Backspace to delete last character
      await vibe.keyboard.press('Backspace');

      const val = await input.value();
      assert.strictEqual(val, 'hell');
    } finally {
      await bro.close();
    }
  });

  test('keyboard.down() and keyboard.up() hold and release keys', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(`${baseURL}/login`);

      const input = await vibe.find('#username');
      await input.click();
      await vibe.keyboard.type('hello');

      // Hold shift, press Home to select all, release shift, then delete
      await vibe.keyboard.down('Shift');
      await vibe.keyboard.press('Home');
      await vibe.keyboard.up('Shift');
      await vibe.keyboard.press('Backspace');

      const val = await input.value();
      assert.strictEqual(val, '');
    } finally {
      await bro.close();
    }
  });
});

describe('Mouse: page-level input', () => {
  test('mouse.click() clicks at coordinates', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(`${baseURL}/login`);

      // Find the username input bounds and click it via mouse
      const input = await vibe.find('#username');
      const bounds = await input.bounds();
      const cx = bounds.x + bounds.width / 2;
      const cy = bounds.y + bounds.height / 2;

      await vibe.mouse.click(cx, cy);
      await vibe.keyboard.type('mouseuser');

      const val = await input.value();
      assert.strictEqual(val, 'mouseuser');
    } finally {
      await bro.close();
    }
  });

  test('mouse.move() moves to coordinates', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(`${baseURL}/hovers`);

      // Get first figure position
      const figure = await vibe.find('.figure');
      const bounds = await figure.bounds();
      const cx = bounds.x + bounds.width / 2;
      const cy = bounds.y + bounds.height / 2;

      // Move mouse to trigger hover
      await vibe.mouse.move(cx, cy);

      // Wait briefly for CSS transition
      await vibe.wait(500);

      const visible = await vibe.evaluate(`
        const caption = document.querySelector('.figure .figcaption');
        const style = window.getComputedStyle(caption);
        return style.opacity !== '0';
      `);
      assert.ok(visible, 'Hover caption should be visible after mouse.move');
    } finally {
      await bro.close();
    }
  });

  test('mouse.wheel() scrolls the page', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go('data:text/html,<body style="margin:0"><div style="height:5000px;background:linear-gradient(red,blue)">Tall</div></body>');

      // Scroll down
      await vibe.mouse.wheel(0, 500);
      await vibe.wait(300);

      const scrollY = await vibe.evaluate('return window.scrollY;');
      assert.ok(scrollY > 0, `Page should have scrolled down, scrollY: ${scrollY}`);
    } finally {
      await bro.close();
    }
  });
});

// --- Screenshots & PDF ---

describe('Screenshots: options', () => {
  test('screenshot() returns a PNG buffer', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go('https://example.com');

      const buf = await vibe.screenshot();
      assert.ok(Buffer.isBuffer(buf), 'screenshot() should return a Buffer');
      assert.ok(buf.length > 100, 'Screenshot should have meaningful content');

      // Check PNG magic bytes
      assert.strictEqual(buf[0], 0x89);
      assert.strictEqual(buf[1], 0x50); // P
      assert.strictEqual(buf[2], 0x4e); // N
      assert.strictEqual(buf[3], 0x47); // G
    } finally {
      await bro.close();
    }
  });

  test('screenshot({ fullPage: true }) captures full page', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go('https://example.com');

      const viewportShot = await vibe.screenshot();
      const fullShot = await vibe.screenshot({ fullPage: true });

      assert.ok(Buffer.isBuffer(fullShot), 'fullPage screenshot should return a Buffer');
      assert.ok(fullShot.length > 100, 'fullPage screenshot should have meaningful content');
    } finally {
      await bro.close();
    }
  });

  test('screenshot({ clip }) captures a specific region', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go('https://example.com');

      const clipShot = await vibe.screenshot({
        clip: { x: 0, y: 0, width: 100, height: 100 },
      });

      assert.ok(Buffer.isBuffer(clipShot), 'clip screenshot should return a Buffer');
      assert.ok(clipShot.length > 100, 'clip screenshot should have meaningful content');
    } finally {
      await bro.close();
    }
  });

  test('pdf() returns a PDF buffer', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go('https://example.com');

      const buf = await vibe.pdf();
      assert.ok(Buffer.isBuffer(buf), 'pdf() should return a Buffer');
      assert.ok(buf.length > 100, 'PDF should have meaningful content');

      // Check PDF magic bytes (%PDF)
      const header = buf.subarray(0, 5).toString('ascii');
      assert.ok(header.startsWith('%PDF'), `PDF should start with %PDF, got: ${header}`);
    } finally {
      await bro.close();
    }
  });
});

// --- Evaluation ---

describe('Evaluation: page-level', () => {
  test('eval() evaluates an expression', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go('https://example.com');

      const result = await vibe.eval('1 + 1');
      assert.strictEqual(result, 2);
    } finally {
      await bro.close();
    }
  });

  test('eval() returns strings', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go('https://example.com');

      const result = await vibe.eval('document.title');
      assert.strictEqual(result, 'Example Domain');
    } finally {
      await bro.close();
    }
  });

  test('eval() returns null for undefined', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go('https://example.com');

      const result = await vibe.eval('undefined');
      assert.strictEqual(result, null);
    } finally {
      await bro.close();
    }
  });

  test('evalHandle() returns a handle ID', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go('https://example.com');

      const handle = await vibe.evalHandle('document.body');
      assert.ok(typeof handle === 'string', `evalHandle should return string, got: ${typeof handle}`);
      assert.ok(handle.length > 0, 'Handle ID should not be empty');
    } finally {
      await bro.close();
    }
  });

  test('addScript() injects inline JS', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go('https://example.com');

      await vibe.addScript('window.__testVar = 42;');

      const result = await vibe.eval('window.__testVar');
      assert.strictEqual(result, 42);
    } finally {
      await bro.close();
    }
  });

  test('addStyle() injects inline CSS', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go('https://example.com');

      await vibe.addStyle('body { background-color: rgb(255, 0, 0) !important; }');

      const bg = await vibe.evaluate(`
        return window.getComputedStyle(document.body).backgroundColor;
      `);
      assert.strictEqual(bg, 'rgb(255, 0, 0)');
    } finally {
      await bro.close();
    }
  });

  test('expose() injects a named function on window', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go('https://example.com');

      await vibe.expose('myAdd', '(a, b) => a + b');

      const result = await vibe.eval('window.myAdd(2, 3)');
      assert.strictEqual(result, 5);
    } finally {
      await bro.close();
    }
  });
});

// --- Checkpoint ---

describe('Input & Eval Checkpoint', () => {
  test('keyboard.type, mouse.click, screenshot, eval all work together', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(`${baseURL}/login`);

      // Use keyboard.type via page.keyboard
      const input = await vibe.find('#username');
      await input.click();
      await vibe.keyboard.type('tomsmith');

      // Use mouse.click to click password field
      const pwInput = await vibe.find('#password');
      const pwBounds = await pwInput.bounds();
      await vibe.mouse.click(
        pwBounds.x + pwBounds.width / 2,
        pwBounds.y + pwBounds.height / 2
      );
      await vibe.keyboard.type('SuperSecretPassword!');

      // Verify values using eval
      const username = await vibe.eval('document.querySelector("#username").value');
      assert.strictEqual(username, 'tomsmith');
      const password = await vibe.eval('document.querySelector("#password").value');
      assert.strictEqual(password, 'SuperSecretPassword!');

      // Take screenshot
      const shot = await vibe.screenshot();
      assert.ok(Buffer.isBuffer(shot), 'Screenshot should be a buffer');
      assert.ok(shot.length > 100, 'Screenshot should have content');

      // Submit the form
      const btn = await vibe.find('button[type="submit"]');
      await btn.click();
      await vibe.waitUntil.url('**/secure');

      const url = await vibe.url();
      assert.ok(url.includes('/secure'), `Should be on /secure, got: ${url}`);
    } finally {
      await bro.close();
    }
  });
});
