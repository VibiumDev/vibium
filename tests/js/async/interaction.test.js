/**
 * JS Library Tests: Element Interaction
 * Tests click, dblclick, fill, type, press, clear, check, uncheck,
 * selectOption, hover, focus, tap, scrollIntoView, dispatchEvent.
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

// --- Checkpoint test: Login flow ---

describe('Interaction: Checkpoint', () => {
  test('login flow: fill + click', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(baseURL + '/login');

      const username = await vibe.find('#username');
      await username.fill('tomsmith');

      const password = await vibe.find('#password');
      await password.fill('SuperSecretPassword!');

      const loginBtn = await vibe.find('button[type="submit"]');
      await loginBtn.click();

      await vibe.waitUntil.url('**/secure');
      const url = await vibe.url();
      assert.ok(url.includes('/secure'), `Should be on /secure page, got: ${url}`);
    } finally {
      await bro.close();
    }
  });

  test('checkbox: check and uncheck', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(baseURL + '/checkboxes');

      const checkboxes = await vibe.findAll('input[type="checkbox"]');
      const first = checkboxes.first();

      // First checkbox starts unchecked
      await first.check();
      const checked = await vibe.evaluate(`
        return document.querySelectorAll('input[type="checkbox"]')[0].checked;
      `);
      assert.strictEqual(checked, true, 'First checkbox should be checked');

      // Uncheck it
      await first.uncheck();
      const unchecked = await vibe.evaluate(`
        return document.querySelectorAll('input[type="checkbox"]')[0].checked;
      `);
      assert.strictEqual(unchecked, false, 'First checkbox should be unchecked');
    } finally {
      await bro.close();
    }
  });

  test('hover reveals hidden content', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(baseURL + '/hovers');

      const figures = await vibe.findAll('.figure');
      const first = figures.first();
      await first.hover();

      // After hover, the caption should be visible
      // Wait briefly for CSS transition
      await new Promise(resolve => setTimeout(resolve, 500));
      const visible = await vibe.evaluate(`
        const caption = document.querySelector('.figure .figcaption');
        const style = window.getComputedStyle(caption);
        return style.opacity !== '0' && style.display !== 'none';
      `);
      assert.ok(visible, 'Hovering should reveal caption');
    } finally {
      await bro.close();
    }
  });
});

// --- Individual interaction tests ---

describe('Interaction: Click variants', () => {
  test('click navigates via link', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(baseURL);

      const link = await vibe.find('a[href="/login"]');
      await link.click();

      await vibe.waitUntil.url('**/login');
      const url = await vibe.url();
      assert.ok(url.includes('/login'), `Should navigate to /login, got: ${url}`);
    } finally {
      await bro.close();
    }
  });

  test('dblclick selects text', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go('https://example.com');

      const h1 = await vibe.find('h1');
      await h1.dblclick();

      // Double-clicking on text should select it
      const selectedText = await vibe.evaluate(`
        return window.getSelection().toString();
      `);
      assert.ok(selectedText.length > 0, 'Double-click should select text');
    } finally {
      await bro.close();
    }
  });
});

describe('Interaction: Input methods', () => {
  test('fill clears and enters text', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(baseURL + '/login');

      const username = await vibe.find('#username');
      await username.fill('firstvalue');
      await username.fill('secondvalue');

      const value = await vibe.evaluate(`
        return document.getElementById('username').value;
      `);
      assert.strictEqual(value, 'secondvalue', 'fill() should clear and replace text');
    } finally {
      await bro.close();
    }
  });

  test('type appends text', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(baseURL + '/inputs');

      const input = await vibe.find('input');
      await input.type('123');
      await input.type('45');

      const value = await vibe.evaluate(`
        return document.querySelector('input').value;
      `);
      assert.strictEqual(value, '12345', 'type() should append text');
    } finally {
      await bro.close();
    }
  });

  test('clear removes text', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(baseURL + '/login');

      const username = await vibe.find('#username');
      await username.fill('sometext');
      await username.clear();

      const value = await vibe.evaluate(`
        return document.getElementById('username').value;
      `);
      assert.strictEqual(value, '', 'clear() should empty the input');
    } finally {
      await bro.close();
    }
  });

  test('press sends key events', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(baseURL + '/login');

      const username = await vibe.find('#username');
      await username.fill('tomsmith');

      const password = await vibe.find('#password');
      await password.fill('SuperSecretPassword!');

      // Press Enter to submit instead of clicking
      await password.press('Enter');

      await vibe.waitUntil.url('**/secure');
      const url = await vibe.url();
      assert.ok(url.includes('/secure'), `Enter should submit form, got: ${url}`);
    } finally {
      await bro.close();
    }
  });
});

describe('Interaction: Select', () => {
  test('selectOption changes dropdown value', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(baseURL + '/dropdown');

      const dropdown = await vibe.find('#dropdown');
      await dropdown.selectOption('2');

      const value = await vibe.evaluate(`
        return document.getElementById('dropdown').value;
      `);
      assert.strictEqual(value, '2', 'selectOption should set dropdown value');
    } finally {
      await bro.close();
    }
  });
});

describe('Interaction: Focus', () => {
  test('focus sets active element', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(baseURL + '/login');

      const username = await vibe.find('#username');
      await username.focus();

      const activeId = await vibe.evaluate(`
        return document.activeElement ? document.activeElement.id : '';
      `);
      assert.strictEqual(activeId, 'username', 'focus() should set active element');
    } finally {
      await bro.close();
    }
  });
});

describe('Interaction: Scroll', () => {
  test('scrollIntoView scrolls to element', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(baseURL);

      // The footer is below the fold
      const footer = await vibe.find('#page-footer');
      await footer.scrollIntoView();

      const inView = await vibe.evaluate(`
        const footer = document.getElementById('page-footer');
        const rect = footer.getBoundingClientRect();
        return rect.top >= 0 && rect.top < window.innerHeight;
      `);
      assert.ok(inView, 'scrollIntoView should bring element into viewport');
    } finally {
      await bro.close();
    }
  });
});

describe('Interaction: findAll index bug fix', () => {
  test('findAll().nth(1).click() acts on correct element', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go(baseURL + '/checkboxes');

      const checkboxes = await vibe.findAll('input[type="checkbox"]');
      assert.strictEqual(checkboxes.count(), 2, 'Should find 2 checkboxes');

      // Second checkbox starts checked
      const secondChecked = await vibe.evaluate(`
        return document.querySelectorAll('input[type="checkbox"]')[1].checked;
      `);
      assert.strictEqual(secondChecked, true, 'Second checkbox starts checked');

      // Click the second one (index 1) to uncheck it
      await checkboxes.nth(1).click();

      const afterClick = await vibe.evaluate(`
        return document.querySelectorAll('input[type="checkbox"]')[1].checked;
      `);
      assert.strictEqual(afterClick, false, 'nth(1).click() should toggle second checkbox');

      // First checkbox should be unchanged
      const firstUnchanged = await vibe.evaluate(`
        return document.querySelectorAll('input[type="checkbox"]')[0].checked;
      `);
      assert.strictEqual(firstUnchanged, false, 'First checkbox should be unchanged');
    } finally {
      await bro.close();
    }
  });
});

describe('Interaction: dispatchEvent', () => {
  test('dispatchEvent fires custom event', async () => {
    const bro = await browser.launch({ headless: true });
    try {
      const vibe = await bro.page();
      await vibe.go('https://example.com');

      // Set up an event listener
      await vibe.evaluate(`
        window.__eventFired = false;
        document.querySelector('h1').addEventListener('click', () => {
          window.__eventFired = true;
        });
      `);

      const h1 = await vibe.find('h1');
      await h1.dispatchEvent('click', { bubbles: true });

      const fired = await vibe.evaluate(`
        return window.__eventFired;
      `);
      assert.strictEqual(fired, true, 'dispatchEvent should fire the event');
    } finally {
      await bro.close();
    }
  });
});
