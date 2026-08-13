/**
 * JS Library Tests: Sync API
 * Tests browser.start() and BrowserSync → PageSync → ElementSync.
 *
 * The HTTP server runs in a child process because the sync API blocks
 * the main thread with Atomics.wait(), which would deadlock an in-process server.
 */

const { test, describe, before, after } = require("../../../helpers/capabilities").suite("core");
const assert = require('node:assert');
const { fork } = require('child_process');
const path = require('path');

const { browser } = require('../../../../clients/javascript/dist/sync');

// --- Server child process ---

let serverProcess;
let baseURL;
let bro;

before(async () => {
  // Start the HTTP server in a child process
  serverProcess = fork(path.join(__dirname, '../sync-test-server.js'), [], { silent: true });

  // Read the base URL from the server's stdout (first line only)
  baseURL = await new Promise((resolve, reject) => {
    let data = '';
    serverProcess.stdout.on('data', (chunk) => {
      data += chunk.toString();
      const firstLine = data.split('\n')[0].trim();
      if (firstLine.startsWith('http://')) resolve(firstLine);
    });
    serverProcess.on('error', reject);
    setTimeout(() => reject(new Error('Server startup timeout')), 5000);
  });

  bro = browser.start({ headless: true });
});

after(() => {
  bro.stop();
  if (serverProcess) serverProcess.kill();
});

// --- Tests ---

describe('Sync API: Browser lifecycle', () => {
  test('browser.start() and stop()', () => {
    const bro = browser.start({ headless: true });
    assert.ok(bro, 'Should return a BrowserSync instance');
    bro.stop();
  });

  test('browser.page() returns default page', () => {
    const vibe = bro.page();
    assert.ok(vibe, 'Should return a PageSync');
  });
});

describe('Sync API: Multi-page', () => {
  test('newPage() creates a new tab', () => {
    const page1 = bro.page();
    const page2 = bro.newPage();
    assert.ok(page2, 'Should return a new PageSync');
    const allPages = bro.pages();
    assert.ok(allPages.length >= 2, 'Should have at least 2 pages');
    page2.close();
  });
});

describe('Sync API: Navigation', () => {

  test('go() navigates to URL', () => {
    const vibe = bro.page();
    vibe.go(baseURL);
    assert.ok(true, 'Navigation succeeded');
  });

  test('url() returns current URL', () => {
    const vibe = bro.page();
    vibe.go(baseURL);
    const url = vibe.url();
    assert.ok(url.includes('127.0.0.1'), 'Should contain host');
  });

  test('title() returns page title', () => {
    const vibe = bro.page();
    vibe.go(baseURL);
    const title = vibe.title();
    assert.strictEqual(title, 'Test App');
  });

  test('content() returns HTML', () => {
    const vibe = bro.page();
    vibe.go(baseURL);
    const html = vibe.content();
    assert.ok(html.includes('Welcome to test-app'), 'Should contain page content');
  });

  test('back() and forward()', () => {
    const vibe = bro.page();
    vibe.go(baseURL);
    vibe.go(`${baseURL}/subpage`);
    assert.strictEqual(vibe.title(), 'Subpage');

    vibe.back();
    assert.strictEqual(vibe.title(), 'Test App');

    vibe.forward();
    assert.strictEqual(vibe.title(), 'Subpage');
  });

  test('reload()', () => {
    const vibe = bro.page();
    vibe.go(baseURL);
    vibe.reload();
    assert.strictEqual(vibe.title(), 'Test App');
  });
});

describe('Sync API: Screenshots & PDF', () => {

  test('screenshot() returns PNG buffer', () => {
    const vibe = bro.page();
    vibe.go(baseURL);
    const png = vibe.screenshot();

    assert.ok(Buffer.isBuffer(png), 'Should return a Buffer');
    assert.ok(png.length > 100, 'Should have reasonable size');
    assert.strictEqual(png[0], 0x89, 'PNG magic byte 1');
    assert.strictEqual(png[1], 0x50, 'PNG magic byte 2');
  });

  test.requires('pdf')('pdf() returns PDF buffer', () => {
    const vibe = bro.page();
    vibe.go(baseURL);
    const pdf = vibe.pdf();

    assert.ok(Buffer.isBuffer(pdf), 'Should return a Buffer');
    assert.ok(pdf.length > 100, 'Should have reasonable size');
    assert.strictEqual(pdf[0], 0x25, 'PDF magic byte');
    assert.strictEqual(pdf[1], 0x50, 'PDF magic byte');
  });
});

describe('Sync API: Evaluation', () => {

  test('evaluate() executes JavaScript', () => {
    const vibe = bro.page();
    vibe.go(baseURL);
    const title = vibe.evaluate('document.title');
    assert.strictEqual(title, 'Test App');
  });

  test('eval() evaluates expression', () => {
    const vibe = bro.page();
    vibe.go(`${baseURL}/eval`);
    const val = vibe.evaluate('window.testVal');
    assert.strictEqual(val, 42);
  });

  test('eval() returns computed value', () => {
    const vibe = bro.page();
    vibe.go(baseURL);
    const year = vibe.evaluate('new Date().getFullYear()');
    assert.strictEqual(typeof year, 'number');
    assert.ok(year >= 2025);
  });
});

describe('Sync API: Element finding', () => {

  test('find() locates element by CSS selector', () => {
    const vibe = bro.page();
    vibe.go(baseURL);
    const heading = vibe.find('h1.heading');
    assert.ok(heading, 'Should return an ElementSync');
    assert.ok(heading.info, 'Should have info');
    assert.match(heading.info.tag, /^h1$/i);
  });

  test('find() with semantic selector', () => {
    const vibe = bro.page();
    vibe.go(baseURL);
    const link = vibe.find({ role: 'link', text: 'Go to subpage' });
    assert.ok(link, 'Should find link by role+text');
    assert.match(link.info.tag, /^a$/i);
  });

  test('findAll() returns an array', () => {
    const vibe = bro.page();
    vibe.go(`${baseURL}/links`);
    const links = vibe.findAll('a.link');
    assert.ok(Array.isArray(links), 'Should return an array');
    assert.strictEqual(links.length, 4);
  });

  test('find() auto-waits for element', () => {
    const vibe = bro.page();
    vibe.go(baseURL);
    const el = vibe.find('h1.heading');
    assert.ok(el, 'Should return an ElementSync');
  });
});

describe('Sync API: findAll array', () => {

  test('length, [0], .at(-1), [i]', () => {
    const vibe = bro.page();
    vibe.go(`${baseURL}/links`);
    const items = vibe.findAll('a.link');

    assert.strictEqual(items.length, 4);

    const first = items[0];
    assert.ok(first.text().includes('Link 1'));

    const last = items.at(-1);
    assert.ok(last.text().includes('Link 4'));

    const second = items[1];
    assert.ok(second.text().includes('Link 2'));
  });

  test('iteration with for...of', () => {
    const vibe = bro.page();
    vibe.go(`${baseURL}/links`);
    const items = vibe.findAll('a.link');
    let count = 0;
    for (const item of items) {
      count++;
      assert.ok(item, 'Each item should be an ElementSync');
    }
    assert.strictEqual(count, 4);
  });
});

describe('Sync API: Element interaction', () => {

  test('click() navigates via link', () => {
    const vibe = bro.page();
    vibe.go(baseURL);
    const link = vibe.find('a[href="/subpage"]');
    link.click();
    vibe.find('h3'); // wait for subpage to load
    assert.strictEqual(vibe.title(), 'Subpage');
  });

  test('fill() and value()', () => {
    const vibe = bro.page();
    vibe.go(`${baseURL}/inputs`);
    const input = vibe.find('#text-input');
    input.fill('hello world');
    assert.strictEqual(input.value(), 'hello world');
  });

  test('type() appends text', () => {
    const vibe = bro.page();
    vibe.go(`${baseURL}/inputs`);
    const input = vibe.find('#text-input');
    input.type('12345');
    const value = vibe.evaluate("document.querySelector('#text-input').value");
    assert.strictEqual(value, '12345');
  });

  test('check() and uncheck()', () => {
    const vibe = bro.page();
    vibe.go(`${baseURL}/form`);
    const checkbox = vibe.find('#agree');
    checkbox.check();
    assert.strictEqual(checkbox.isChecked(), true);
    checkbox.uncheck();
    assert.strictEqual(checkbox.isChecked(), false);
  });

  test('selectOption()', () => {
    const vibe = bro.page();
    vibe.go(`${baseURL}/form`);
    const select = vibe.find('#color');
    select.selectOption('blue');
    assert.strictEqual(select.value(), 'blue');
  });

  test('hover()', () => {
    const vibe = bro.page();
    vibe.go(baseURL);
    const heading = vibe.find('h1');
    heading.hover();
    assert.ok(true, 'hover completed without error');
  });

  test('press() on element', () => {
    const vibe = bro.page();
    vibe.go(`${baseURL}/inputs`);
    const input = vibe.find('#text-input');
    input.click();
    input.press('a');
    const value = vibe.evaluate("document.querySelector('#text-input').value");
    assert.ok(typeof value === 'string');
  });
});

describe('Sync API: Element state', () => {

  test('text() returns textContent', () => {
    const vibe = bro.page();
    vibe.go(baseURL);
    const heading = vibe.find('h1.heading');
    const text = heading.text();
    assert.ok(text.includes('Welcome to test-app'));
  });

  test('innerText() returns rendered text', () => {
    const vibe = bro.page();
    vibe.go(baseURL);
    const heading = vibe.find('h1.heading');
    const text = heading.innerText();
    assert.ok(text.includes('Welcome to test-app'));
  });

  test('html() returns innerHTML', () => {
    const vibe = bro.page();
    vibe.go(baseURL);
    const info = vibe.find('#info');
    const html = info.html();
    assert.ok(html.includes('Some info text'));
  });

  test('attr() returns attribute value', () => {
    const vibe = bro.page();
    vibe.go(baseURL);
    const link = vibe.find('a[href="/subpage"]');
    assert.strictEqual(link.attr('href'), '/subpage');
  });

  test('bounds() returns bounding box', () => {
    const vibe = bro.page();
    vibe.go(baseURL);
    const heading = vibe.find('h1');
    const box = heading.bounds();
    assert.ok(typeof box.x === 'number');
    assert.ok(typeof box.y === 'number');
    assert.ok(box.width > 0);
    assert.ok(box.height > 0);
  });

  test('isVisible() returns true for visible elements', () => {
    const vibe = bro.page();
    vibe.go(baseURL);
    const heading = vibe.find('h1');
    assert.strictEqual(heading.isVisible(), true);
  });

  test('isEnabled() returns true for enabled elements', () => {
    const vibe = bro.page();
    vibe.go(`${baseURL}/form`);
    const input = vibe.find('#name');
    assert.strictEqual(input.isEnabled(), true);
  });

  test('isEditable() returns true for editable elements', () => {
    const vibe = bro.page();
    vibe.go(`${baseURL}/form`);
    const input = vibe.find('#name');
    assert.strictEqual(input.isEditable(), true);
  });
});

describe('Sync API: Scoped find', () => {

  test('element.find() scoped to parent', () => {
    const vibe = bro.page();
    vibe.go(`${baseURL}/links`);
    const nested = vibe.find('#nested');
    const span = nested.find('.inner');
    assert.ok(span.text().includes('span'));
  });

  test('element.findAll() scoped to parent', () => {
    const vibe = bro.page();
    vibe.go(`${baseURL}/links`);
    const nested = vibe.find('#nested');
    const spans = nested.findAll('.inner');
    assert.strictEqual(spans.length, 2);
  });
});

describe('Sync API: Keyboard, Mouse, Touch', () => {

  test('keyboard.type() types text', () => {
    const vibe = bro.page();
    vibe.go(`${baseURL}/inputs`);
    vibe.find('#text-input').click();
    vibe.keyboard.type('hello');
    const value = vibe.evaluate("document.querySelector('#text-input').value");
    assert.strictEqual(value, 'hello');
  });

  test('keyboard.press() presses a key', () => {
    const vibe = bro.page();
    vibe.go(`${baseURL}/inputs`);
    vibe.find('#text-input').click();
    vibe.keyboard.press('a');
    assert.ok(true, 'press completed');
  });

  test('mouse.click() clicks at coordinates', () => {
    const vibe = bro.page();
    vibe.go(baseURL);
    vibe.mouse.click(100, 100);
    assert.ok(true, 'mouse click completed');
  });
});

describe('Sync API: Clock control', () => {

  test('clock.install() and setFixedTime()', () => {
    const vibe = bro.page();
    vibe.go(`${baseURL}/clock`);
    vibe.clock.install({ time: new Date('2025-06-15T12:00:00Z') });
    vibe.clock.setFixedTime(new Date('2025-06-15T12:00:00Z'));
    const year = vibe.evaluate('new Date().getFullYear()');
    assert.strictEqual(year, 2025);
  });

  test('clock.fastForward()', () => {
    const vibe = bro.newPage();
    vibe.go(`${baseURL}/clock`);
    vibe.clock.install({ time: new Date('2025-06-15T12:00:00Z') });
    vibe.clock.fastForward(60000);
    const time = vibe.evaluate('Date.now()');
    assert.ok(typeof time === 'number');
  });
});

describe('Sync API: Viewport & emulation', () => {

  test('setViewport() and viewport()', () => {
    const vibe = bro.page();
    vibe.setViewport({ width: 375, height: 812 });
    const vp = vibe.viewport();
    assert.strictEqual(vp.width, 375);
    assert.strictEqual(vp.height, 812);
  });

  test('setWindow() and window()', () => {
    const vibe = bro.page();
    vibe.setWindow({ width: 900, height: 700 });
    const win = vibe.window();
    assert.strictEqual(win.width, 900);
    assert.strictEqual(win.height, 700);
  });

  test('setContent() replaces page HTML', () => {
    const vibe = bro.page();
    vibe.setContent('<html><body><h1>Custom</h1></body></html>');
    const heading = vibe.find('h1');
    assert.strictEqual(heading.text(), 'Custom');
  });

  test('a11yTree() returns accessibility tree', () => {
    const vibe = bro.page();
    vibe.go(baseURL);
    const tree = vibe.a11yTree();
    assert.ok(tree, 'Should return a tree');
    assert.ok(tree.role, 'Root should have a role');
  });
});

describe('Sync API: Context isolation', () => {
  test('newContext() creates isolated context', () => {
    const ctx = bro.newContext();
    const vibe = ctx.newPage();
    vibe.go(baseURL);
    assert.strictEqual(vibe.title(), 'Test App');
    ctx.close();
  });

  test('cookies in context', () => {
    const ctx = bro.newContext();
    const vibe = ctx.newPage();
    vibe.go(baseURL);
    ctx.setCookies([{ name: 'test', value: 'val', url: baseURL }]);
    const cookies = ctx.cookies();
    assert.ok(cookies.some(c => c.name === 'test'), 'Should have the test cookie');
    ctx.clearCookies();
    const cleared = ctx.cookies();
    assert.ok(!cleared.some(c => c.name === 'test'), 'Cookie should be cleared');
    ctx.close();
  });
});

describe.requires('dialogs')('Sync API: Dialog auto-handling', () => {

  test('onDialog("accept") auto-accepts alerts', () => {
    const vibe = bro.page();
    vibe.go(`${baseURL}/dialog`);
    vibe.onDialog('accept');
    vibe.find('#alert-btn').click();
    assert.ok(true, 'Dialog was auto-accepted');
  });
});

describe.requires('network')('Sync API: Route handler callback', () => {

  test('route handler can fulfill with custom response', () => {
    const vibe = bro.newPage();
    vibe.go(`${baseURL}/fetch`);
    vibe.route('**/api/data', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ message: 'mocked', count: 99 }),
      });
    });
    vibe.evaluate('doFetch()');
    const text = vibe.find('#result').text();
    const data = JSON.parse(text);
    assert.strictEqual(data.message, 'mocked');
    assert.strictEqual(data.count, 99);
  });

  test('route handler can inspect request properties', () => {
    const vibe = bro.newPage();
    vibe.go(`${baseURL}/fetch`);
    let capturedUrl = '';
    let capturedMethod = '';
    vibe.route('**/api/data', (route) => {
      capturedUrl = route.request.url;
      capturedMethod = route.request.method;
      route.continue();
    });
    vibe.evaluate('doFetch()');
    assert.ok(capturedUrl.includes('/api/data'), 'Should capture URL');
    assert.strictEqual(capturedMethod, 'GET');
  });

  test('route handler can abort request', () => {
    const vibe = bro.newPage();
    vibe.go(`${baseURL}/fetch`);
    vibe.route('**/api/data', (route) => {
      route.abort();
    });
    // fetch will fail — check that it doesn't hang
    vibe.evaluate('doFetch().catch(e => { document.getElementById("result").textContent = "ABORTED"; })');
    const text = vibe.find('#result').text();
    assert.strictEqual(text, 'ABORTED');
  });

  test('route handler defaults to continue if no action called', () => {
    const vibe = bro.newPage();
    vibe.go(`${baseURL}/fetch`);
    vibe.route('**/api/data', () => {
      // No action — should default to continue
    });
    vibe.evaluate('doFetch()');
    const text = vibe.find('#result').text();
    const data = JSON.parse(text);
    assert.strictEqual(data.message, 'real data');
  });

  test('static route actions still work alongside handler routes', () => {
    const vibe = bro.newPage();
    vibe.go(`${baseURL}/fetch`);
    // Static action (unchanged API)
    vibe.route('**/api/data', { status: 200, body: '{"static":true}' });
    vibe.evaluate('doFetch()');
    const text = vibe.find('#result').text();
    const data = JSON.parse(text);
    assert.strictEqual(data.static, true);
  });

  test('unroute removes handler route', () => {
    const vibe = bro.newPage();
    vibe.go(`${baseURL}/fetch`);
    vibe.route('**/api/data', (route) => {
      route.fulfill({ status: 200, body: '{"handler":"active"}' });
    });
    vibe.evaluate('doFetch()');
    let text = vibe.find('#result').text();
    assert.strictEqual(JSON.parse(text).handler, 'active');

    // Remove the route — clear result first
    vibe.evaluate('document.getElementById("result").textContent = ""');
    vibe.unroute('**/api/data');
    vibe.evaluate('doFetch()');
    text = vibe.find('#result').text();
    const data = JSON.parse(text);
    assert.strictEqual(data.message, 'real data', 'Should get real data after unroute');
  });
});

describe.requires('dialogs')('Sync API: Dialog handler callback', () => {

  test('dialog handler can accept alerts', () => {
    const vibe = bro.newPage();
    vibe.go(`${baseURL}/prompt`);
    let capturedMessage = '';
    let capturedType = '';
    vibe.onDialog((dialog) => {
      capturedType = dialog.type();
      capturedMessage = dialog.message();
      dialog.accept();
    });
    vibe.find('#alert-btn').click();
    assert.strictEqual(capturedType, 'alert');
    assert.strictEqual(capturedMessage, 'hello');
  });

  test('dialog handler can dismiss confirms', () => {
    const vibe = bro.newPage();
    vibe.go(`${baseURL}/prompt`);
    vibe.onDialog((dialog) => {
      dialog.dismiss();
    });
    vibe.find('#confirm-btn').click();
    const text = vibe.find('#result').text();
    assert.strictEqual(text, 'false');
  });

  test('dialog handler can accept confirms', () => {
    const vibe = bro.newPage();
    vibe.go(`${baseURL}/prompt`);
    vibe.onDialog((dialog) => {
      dialog.accept();
    });
    vibe.find('#confirm-btn').click();
    const text = vibe.find('#result').text();
    assert.strictEqual(text, 'true');
  });

  test('dialog handler can provide prompt text', () => {
    const vibe = bro.newPage();
    vibe.go(`${baseURL}/prompt`);
    vibe.onDialog((dialog) => {
      if (dialog.type() === 'prompt') {
        dialog.accept('Claude');
      } else {
        dialog.dismiss();
      }
    });
    vibe.find('#prompt-btn').click();
    const text = vibe.find('#result').text();
    assert.strictEqual(text, 'Claude');
  });

  test('dialog handler defaults to dismiss if no action called', () => {
    const vibe = bro.newPage();
    vibe.go(`${baseURL}/prompt`);
    vibe.onDialog(() => {
      // No action — default dismiss
    });
    vibe.find('#confirm-btn').click();
    const text = vibe.find('#result').text();
    assert.strictEqual(text, 'false');
  });

  test('dialog handler can read defaultValue for prompts', () => {
    const vibe = bro.newPage();
    vibe.go(`${baseURL}/prompt`);
    let defaultVal = '';
    vibe.onDialog((dialog) => {
      defaultVal = dialog.defaultValue();
      dialog.accept();
    });
    // prompt('Enter name:') has no default → empty string
    vibe.find('#prompt-btn').click();
    assert.strictEqual(defaultVal, '');
  });

  test('static onDialog still works', () => {
    const vibe = bro.newPage();
    vibe.go(`${baseURL}/prompt`);
    vibe.onDialog('accept');
    vibe.find('#alert-btn').click();
    assert.ok(true, 'Static accept still works');
  });
});

describe.requires('popups')('Sync API: onPage/onPopup', () => {
  test('onPage fires for new tabs', () => {
    const bro = browser.start({ headless: true });
    try {
      const pages = [];
      bro.onPage((p) => pages.push(p));
      bro.newPage();
      assert.strictEqual(pages.length, 1);
      assert.ok(pages[0], 'Should receive a PageSync');
    } finally {
      bro.stop();
    }
  });

  test('onPopup fires for window.open', () => {
    const bro = browser.start({ headless: true });
    try {
      const popups = [];
      bro.onPopup((p) => popups.push(p));
      const page = bro.page();
      page.evaluate("window.open('about:blank')");
      assert.strictEqual(popups.length, 1);
      assert.ok(popups[0], 'Should receive a PageSync');
    } finally {
      bro.stop();
    }
  });

  test('removeAllListeners stops onPage callbacks', () => {
    const bro = browser.start({ headless: true });
    try {
      const pages = [];
      bro.onPage((p) => pages.push(p));
      bro.newPage();
      assert.strictEqual(pages.length, 1);

      bro.removeAllListeners('page');
      bro.newPage();
      assert.strictEqual(pages.length, 1, 'Should still be 1 after removing listener');
    } finally {
      bro.stop();
    }
  });
});

describe.requires('navigation-capture')('Sync API: Capture navigation', () => {

  test('capture.navigation() returns URL on link click', () => {
    const vibe = bro.newPage();
    vibe.go(`${baseURL}/nav-test`);
    const result = vibe.capture.navigation(() => {
      vibe.find('#link').click();
    });
    assert.ok(result.url.includes('/page2'), `Should include /page2, got: ${result.url}`);
  });
});

describe.requires('downloads')('Sync API: Capture download', () => {

  test('capture.download() returns download info', () => {
    const vibe = bro.newPage();
    vibe.go(`${baseURL}/download`);
    const result = vibe.capture.download(() => {
      vibe.find('#download-link').click();
    });
    assert.ok(result.url.includes('/download-file'), `Should include /download-file, got: ${result.url}`);
    assert.strictEqual(result.suggestedFilename, 'test.txt');
  });
});

describe.requires('dialogs')('Sync API: Capture dialog', () => {

  test('capture.dialog() returns dialog info', () => {
    const vibe = bro.newPage();
    vibe.go(`${baseURL}/prompt`);
    // Trigger alert asynchronously — alert() blocks the page, so we use
    // setTimeout to let expectDialogStart register before the dialog opens.
    vibe.evaluate('setTimeout(() => document.getElementById("alert-btn").click(), 50)');
    const result = vibe.capture.dialog();
    assert.strictEqual(result.type, 'alert');
    assert.strictEqual(result.message, 'hello');
  });
});

describe.requires('network')('Sync API: Capture event', () => {

  test('capture.event("navigation") returns URL', () => {
    const vibe = bro.newPage();
    vibe.go(`${baseURL}/nav-test`);
    const result = vibe.capture.event('navigation', () => {
      vibe.find('#link').click();
    });
    assert.ok(result.data, 'Should have event data');
  });
});

describe('Sync API: Full checkpoint', () => {
  test('Phase 8 checkpoint', () => {
    const vibe = bro.newPage();
    vibe.go(baseURL);
    assert.strictEqual(vibe.title(), 'Test App');
    assert.ok(vibe.url().includes('127.0.0.1'));

    const link = vibe.find('a[href="/subpage"]');
    link.click();
    vibe.find('h3'); // wait for subpage to load
    assert.strictEqual(vibe.title(), 'Subpage');

    const png = vibe.screenshot();
    assert.ok(png.length > 100, 'Screenshot should have data');

    const year = vibe.evaluate('new Date().getFullYear()');
    assert.strictEqual(typeof year, 'number');
  });
});
