/**
 * MCP Tests: pinning tool calls to a page
 *
 * Concurrent callers multiplexed over one MCP connection used to share a
 * single ambient "current page", so one caller's navigate moved the page
 * under another caller's click. Page-scoped tools now take an optional page
 * argument that pins the call to that browsing context, and a stale id
 * fails loudly instead of acting on whatever page is current (#383).
 */

const { test, describe, before, after } = require('node:test');
const assert = require('node:assert');
const { spawn, execFileSync } = require('node:child_process');
const { VIBIUM } = require('../helpers');

class MCPClient {
  constructor() {
    this.proc = null;
    this.buffer = '';
    this.responses = [];
    this.resolvers = [];
  }

  start() {
    return new Promise((resolve, reject) => {
      this.proc = spawn(VIBIUM, ['mcp', '--headless'], { stdio: ['pipe', 'pipe', 'pipe'] });
      this.proc.stdout.on('data', (data) => {
        this.buffer += data.toString();
        const lines = this.buffer.split('\n');
        this.buffer = lines.pop();
        for (const line of lines) {
          if (!line.trim()) continue;
          try {
            const response = JSON.parse(line);
            if (this.resolvers.length > 0) this.resolvers.shift()(response);
            else this.responses.push(response);
          } catch { /* non-JSON line */ }
        }
      });
      this.proc.on('error', reject);
      setTimeout(resolve, 100);
    });
  }

  async call(method, params = {}) {
    const id = Date.now() + Math.random();
    this.proc.stdin.write(JSON.stringify({ jsonrpc: '2.0', id, method, params }) + '\n');
    const response = await new Promise((resolve, reject) => {
      if (this.responses.length > 0) return resolve(this.responses.shift());
      const timer = setTimeout(() => reject(new Error('Timeout waiting for response')), 120000);
      this.resolvers.push((r) => { clearTimeout(timer); resolve(r); });
    });
    assert.strictEqual(response.id, id);
    return response;
  }

  async tool(name, args = {}) {
    const response = await this.call('tools/call', { name, arguments: args });
    return response.result;
  }

  stop() {
    if (!this.proc) return;
    if (process.platform === 'win32') {
      try { execFileSync('taskkill', ['/T', '/F', '/PID', this.proc.pid.toString()], { stdio: 'ignore' }); } catch {}
    } else {
      this.proc.kill();
    }
    this.proc = null;
  }
}

function text(result) {
  return (result.content || []).map((c) => c.text || '').join('');
}

function pageIdFrom(result) {
  const match = text(result).match(/\(page: ([^)]+)\)/);
  assert.ok(match, `expected a page id in: ${text(result)}`);
  return match[1];
}

const URL_A = 'data:text/html,<title>page-a</title><h1 id="h">A</h1>';
const URL_B = 'data:text/html,<title>page-b</title><h1 id="h">B</h1>';

let client;

describe('MCP: page pinning', () => {
  before(async () => {
    client = new MCPClient();
    await client.start();
    await client.call('initialize', { capabilities: {} });
  });

  after(() => {
    client.stop();
  });

  test('a pinned call targets its page, not the globally current one', async () => {
    const pageA = pageIdFrom(await client.tool('browser_new_page', { url: URL_A }));
    const pageB = pageIdFrom(await client.tool('browser_new_page', { url: URL_B }));
    assert.notStrictEqual(pageA, pageB);

    // Page B is globally current (last created). Pinned reads must not care.
    const urlA = text(await client.tool('browser_get_url', { page: pageA }));
    const urlB = text(await client.tool('browser_get_url', { page: pageB }));
    assert.match(urlA, /page-a/, 'pinned to A must read A');
    assert.match(urlB, /page-b/, 'pinned to B must read B');

    // Unpinned calls keep today's ambient behavior.
    const ambient = text(await client.tool('browser_get_url', {}));
    assert.match(ambient, /page-b/, 'no page argument means the current page');
  });

  test('a pinned navigate moves its page and leaves the current one alone', async () => {
    const pageA = pageIdFrom(await client.tool('browser_new_page', { url: URL_A }));
    const pageB = pageIdFrom(await client.tool('browser_new_page', { url: URL_B }));

    await client.tool('browser_navigate', {
      page: pageA,
      url: 'data:text/html,<title>page-a2</title>',
    });

    assert.match(text(await client.tool('browser_get_title', { page: pageA })), /page-a2/);
    assert.match(text(await client.tool('browser_get_title', { page: pageB })), /page-b/, 'the other page must be untouched');
    assert.match(text(await client.tool('browser_get_title', {})), /page-b/, 'the ambient page must not move on a pinned navigate');
  });

  test('a pinned evaluate and find run against the pinned page', async () => {
    const pageA = pageIdFrom(await client.tool('browser_new_page', { url: URL_A }));
    await client.tool('browser_new_page', { url: URL_B });

    assert.match(text(await client.tool('browser_evaluate', { page: pageA, expression: 'document.title' })), /page-a/);
    assert.match(text(await client.tool('browser_get_text', { page: pageA, selector: '#h' })), /A/);
  });

  test('a stale page id fails loudly instead of acting on the current page', async () => {
    const result = await client.tool('browser_get_url', { page: 'gone-1234' });
    assert.ok(result.isError, 'a bad page id must be an error');
    assert.match(text(result), /not found/, 'the error should say the page is gone');
  });

  test('browser_list_pages includes the id to pin with', async () => {
    const listing = text(await client.tool('browser_list_pages', {}));
    assert.match(listing, /\(page: /, 'every row should carry the page id');
  });

  // Element refs (@e1, ...) used to live in one table shared by every page,
  // so one caller's map replaced another's selectors and a pinned click
  // could faithfully target its own page with a selector minted on a
  // different one. Refs are now scoped per page.
  test('a map on one page does not replace another page\'s refs', async () => {
    // #two exists on both pages, so the old shared table resolved A's @e1
    // to B's selector and still found an element — silently the wrong one.
    const pageA = pageIdFrom(await client.tool('browser_new_page', {
      url: 'data:text/html,<button id="one">alpha</button><button id="two">beta</button>',
    }));
    const pageB = pageIdFrom(await client.tool('browser_new_page', {
      url: 'data:text/html,<button id="two">gamma</button>',
    }));

    await client.tool('browser_map', { page: pageA });
    await client.tool('browser_map', { page: pageB });

    const seen = text(await client.tool('browser_get_text', { page: pageA, selector: '@e1' }));
    assert.strictEqual(seen, 'alpha', "page A's @e1 must still be its own first element");
  });

  test('browser_diff compares against the same page\'s previous map', async () => {
    const pageA = pageIdFrom(await client.tool('browser_new_page', {
      url: 'data:text/html,<button>stable</button>',
    }));
    const pageB = pageIdFrom(await client.tool('browser_new_page', {
      url: 'data:text/html,<button>other</button>',
    }));

    await client.tool('browser_map', { page: pageA });
    await client.tool('browser_map', { page: pageB });

    const diff = text(await client.tool('browser_diff_map', { page: pageA }));
    assert.strictEqual(diff, 'No changes detected', "an unchanged page must not diff against another page's map");
  });
});
