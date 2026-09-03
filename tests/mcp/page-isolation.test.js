/**
 * MCP Tests: isolated pages
 *
 * Every page in the agent engine used to share one browser profile, so
 * concurrent callers saw each other's cookies and storage even when they
 * pinned their calls to their own page. browser_new_page now takes an
 * isolated flag that opens the page in its own user context, and closing
 * the last page of an isolated context removes the context with it (#383).
 */

const { test, describe, before, after } = require('node:test');
const assert = require('node:assert');
const { spawn, execFileSync } = require('node:child_process');
const http = require('node:http');
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

let client;
let server;
let origin; // cookies need a real http origin; data: URLs are opaque

describe('MCP: isolated pages', () => {
  before(async () => {
    server = http.createServer((req, res) => {
      res.writeHead(200, { 'Content-Type': 'text/html' });
      res.end('<title>isolation</title><h1>ok</h1>');
    });
    await new Promise((resolve) => server.listen(0, '127.0.0.1', resolve));
    origin = `http://127.0.0.1:${server.address().port}/`;

    client = new MCPClient();
    await client.start();
    await client.call('initialize', { capabilities: {} });
  });

  after(() => {
    client.stop();
    server.close();
  });

  test('an isolated page does not share cookies with the default context', async () => {
    const plain = pageIdFrom(await client.tool('browser_new_page', { url: origin }));
    await client.tool('browser_evaluate', { page: plain, expression: 'document.cookie = "who=default"' });

    const result = await client.tool('browser_new_page', { url: origin, isolated: true });
    assert.match(text(result), /isolated page/, 'the reply should say the page is isolated');
    const isolated = pageIdFrom(result);

    const seen = text(await client.tool('browser_evaluate', { page: isolated, expression: 'document.cookie' }));
    assert.ok(!seen.includes('who=default'), `isolated page must not see the default cookie, got: ${seen}`);

    await client.tool('browser_evaluate', { page: isolated, expression: 'document.cookie = "who=isolated"' });
    const plainSees = text(await client.tool('browser_evaluate', { page: plain, expression: 'document.cookie' }));
    assert.ok(!plainSees.includes('who=isolated'), `default page must not see the isolated cookie, got: ${plainSees}`);
    assert.match(plainSees, /who=default/, 'the default page keeps its own cookie');
  });

  test('two isolated pages do not share cookies with each other', async () => {
    const a = pageIdFrom(await client.tool('browser_new_page', { url: origin, isolated: true }));
    const b = pageIdFrom(await client.tool('browser_new_page', { url: origin, isolated: true }));

    await client.tool('browser_evaluate', { page: a, expression: 'document.cookie = "agent=a"' });
    const bSees = text(await client.tool('browser_evaluate', { page: b, expression: 'document.cookie' }));
    assert.ok(!bSees.includes('agent=a'), `page b must not see page a's cookie, got: ${bSees}`);
  });

  test('browser_list_pages marks isolated pages', async () => {
    const isolated = pageIdFrom(await client.tool('browser_new_page', { isolated: true }));
    const listing = text(await client.tool('browser_list_pages', {}));
    for (const line of listing.split('\n')) {
      if (line.includes(`(page: ${isolated})`)) {
        assert.match(line, /\[isolated\]/, `the isolated page's row should be marked: ${line}`);
        return;
      }
    }
    assert.fail(`isolated page ${isolated} missing from listing: ${listing}`);
  });

  test('closing an isolated page by id removes its context too', async () => {
    const isolated = pageIdFrom(await client.tool('browser_new_page', { isolated: true }));

    const closed = await client.tool('browser_close_page', { page: isolated });
    assert.match(text(closed), /and its isolated context/, 'the reply should confirm the context was removed');

    const listing = text(await client.tool('browser_list_pages', {}));
    assert.ok(!listing.includes(isolated), 'the closed page must be gone from the listing');
  });

  test('closing a default-context page by id leaves no context note', async () => {
    const plain = pageIdFrom(await client.tool('browser_new_page', {}));
    const closed = await client.tool('browser_close_page', { page: plain });
    assert.match(text(closed), new RegExp(`Closed page ${plain}$`), 'a plain page close names the page and nothing else');
  });

  test('closing by a stale page id fails loudly', async () => {
    const result = await client.tool('browser_close_page', { page: 'gone-5678' });
    assert.ok(result.isError, 'a bad page id must be an error');
    assert.match(text(result), /not found/);
  });
});
