/**
 * MCP Server Tests
 * Tests the vibium mcp command via stdin/stdout JSON-RPC
 */

const { test, describe, before, after } = require('node:test');
const assert = require('node:assert');
const { spawn, execFileSync } = require('node:child_process');
const { VIBIUM } = require('../helpers');

/**
 * Helper to run MCP server and send/receive JSON-RPC messages
 */
class MCPClient {
  constructor() {
    this.proc = null;
    this.buffer = '';
    this.responses = [];
    this.resolvers = [];
  }

  start() {
    return new Promise((resolve, reject) => {
      this.proc = spawn(VIBIUM, ['mcp'], {
        stdio: ['pipe', 'pipe', 'pipe'],
      });

      this.proc.stdout.on('data', (data) => {
        this.buffer += data.toString();
        // Process complete JSON lines
        const lines = this.buffer.split('\n');
        this.buffer = lines.pop(); // Keep incomplete line in buffer
        for (const line of lines) {
          if (line.trim()) {
            try {
              const response = JSON.parse(line);
              if (this.resolvers.length > 0) {
                const resolver = this.resolvers.shift();
                resolver(response);
              } else {
                this.responses.push(response);
              }
            } catch (e) {
              // Ignore parse errors for non-JSON output
            }
          }
        }
      });

      this.proc.on('error', reject);

      // Give process a moment to start
      setTimeout(resolve, 100);
    });
  }

  send(method, params = {}, id = null) {
    const msg = {
      jsonrpc: '2.0',
      id: id ?? Date.now(),
      method,
      params,
    };
    this.proc.stdin.write(JSON.stringify(msg) + '\n');
    return msg.id;
  }

  receive(timeout = 60000) {
    return new Promise((resolve, reject) => {
      // Check if we already have a response buffered
      if (this.responses.length > 0) {
        resolve(this.responses.shift());
        return;
      }

      const timer = setTimeout(() => {
        reject(new Error(`Timeout waiting for response after ${timeout}ms`));
      }, timeout);

      this.resolvers.push((response) => {
        clearTimeout(timer);
        resolve(response);
      });
    });
  }

  async call(method, params = {}) {
    const id = this.send(method, params);
    const response = await this.receive();
    assert.strictEqual(response.id, id, 'Response ID should match request ID');
    return response;
  }

  stop() {
    if (this.proc) {
      if (process.platform === 'win32') {
        // On Windows, proc.kill() only kills the immediate process.
        // Use taskkill /T to kill the entire process tree (clicker + chromedriver + Chrome).
        try {
          execFileSync('taskkill', ['/T', '/F', '/PID', this.proc.pid.toString()], { stdio: 'ignore' });
        } catch {
          // Process may have already exited
        }
      } else {
        this.proc.kill();
      }
      this.proc = null;
    }
  }
}

describe('MCP Server: Protocol', () => {
  let client;

  before(async () => {
    client = new MCPClient();
    await client.start();
  });

  after(() => {
    client.stop();
  });

  test('initialize returns server info and capabilities', async () => {
    const response = await client.call('initialize', {
      protocolVersion: '2024-11-05',
      capabilities: {},
      clientInfo: { name: 'test', version: '1.0' },
    });

    assert.strictEqual(response.jsonrpc, '2.0');
    assert.ok(response.result, 'Should have result');
    assert.strictEqual(response.result.protocolVersion, '2024-11-05');
    assert.strictEqual(response.result.serverInfo.name, 'vibium');
    assert.ok(response.result.capabilities.tools, 'Should have tools capability');
  });

  test('tools/list returns all 82 browser tools', async () => {
    const response = await client.call('tools/list', {});

    assert.ok(response.result, 'Should have result');
    assert.ok(response.result.tools, 'Should have tools array');
    assert.strictEqual(response.result.tools.length, 81, 'Should have 81 tools');

    const toolNames = response.result.tools.map(t => t.name);
    const expectedTools = [
      'browser_launch', 'browser_navigate', 'browser_click', 'browser_type',
      'browser_screenshot', 'browser_find', 'browser_evaluate', 'browser_quit',
      'browser_get_text', 'browser_get_url', 'browser_get_title',
      'browser_get_html', 'browser_find_all', 'browser_wait',
      'browser_hover', 'browser_select', 'browser_scroll', 'browser_keys',
      'browser_new_tab', 'browser_list_tabs', 'browser_switch_tab', 'browser_close_tab',
      'browser_a11y_tree',
      'page_clock_install', 'page_clock_fast_forward', 'page_clock_run_for',
      'page_clock_pause_at', 'page_clock_resume', 'page_clock_set_fixed_time',
      'page_clock_set_system_time', 'page_clock_set_timezone',
      'browser_fill', 'browser_press',
      'browser_back', 'browser_forward', 'browser_reload',
      'browser_get_value', 'browser_get_attribute', 'browser_is_visible',
      'browser_check', 'browser_uncheck', 'browser_scroll_into_view',
      'browser_wait_for_url', 'browser_wait_for_load', 'browser_sleep',
      'browser_map', 'browser_diff_map', 'browser_pdf', 'browser_highlight',
      'browser_dblclick', 'browser_focus', 'browser_count',
      'browser_is_enabled', 'browser_is_checked',
      'browser_wait_for_text', 'browser_wait_for_fn',
      'browser_dialog_accept', 'browser_dialog_dismiss',
      'browser_get_cookies', 'browser_set_cookie', 'browser_delete_cookies',
      'browser_mouse_move', 'browser_mouse_down', 'browser_mouse_up', 'browser_mouse_click', 'browser_drag',
      'browser_set_viewport', 'browser_get_viewport',
      'browser_get_window', 'browser_set_window',
      'browser_emulate_media',
      'browser_set_geolocation', 'browser_set_content',
      'browser_frames', 'browser_frame',
      'browser_upload',
      'browser_trace_start', 'browser_trace_stop',
      'browser_storage_state', 'browser_restore_storage',
      'browser_download_set_dir',
    ];
    for (const tool of expectedTools) {
      assert.ok(toolNames.includes(tool), `Should have ${tool}`);
    }
  });

  test('unknown method returns error', async () => {
    const response = await client.call('unknown/method', {});

    assert.ok(response.error, 'Should have error');
    assert.strictEqual(response.error.code, -32601, 'Should be method not found error');
  });

  test('invalid JSON returns parse error', async () => {
    client.proc.stdin.write('not valid json\n');
    const response = await client.receive();

    assert.ok(response.error, 'Should have error');
    assert.strictEqual(response.error.code, -32700, 'Should be parse error');
  });
});

describe('MCP Server: Browser Tools', () => {
  let client;

  before(async () => {
    client = new MCPClient();
    await client.start();

    // Initialize first
    await client.call('initialize', { capabilities: {} });
  });

  after(() => {
    client.stop();
  });

  test('browser_navigate auto-launches browser (lazy launch)', async () => {
    const response = await client.call('tools/call', {
      name: 'browser_navigate',
      arguments: { url: 'https://example.com' },
    });

    assert.ok(response.result, 'Should have result');
    assert.ok(!response.result.isError, 'Should not be an error');
    assert.ok(
      response.result.content[0].text.includes('example.com'),
      'Should confirm navigation'
    );
  });

  test('browser_launch when already running returns success', async () => {
    const response = await client.call('tools/call', {
      name: 'browser_launch',
      arguments: { headless: true },
    });

    assert.ok(response.result, 'Should have result');
    assert.ok(!response.result.isError, 'Should not be an error');
    assert.ok(
      response.result.content[0].text.includes('already running') ||
      response.result.content[0].text.includes('Browser launched'),
      'Should confirm browser state'
    );
  });

  test('browser_navigate goes to URL', async () => {
    const response = await client.call('tools/call', {
      name: 'browser_navigate',
      arguments: { url: 'https://example.com' },
    });

    assert.ok(response.result, 'Should have result');
    assert.ok(!response.result.isError, 'Should not be an error');
    assert.ok(
      response.result.content[0].text.includes('example.com'),
      'Should confirm navigation'
    );
  });

  test('browser_find returns element info', async () => {
    const response = await client.call('tools/call', {
      name: 'browser_find',
      arguments: { selector: 'h1' },
    });

    assert.ok(response.result, 'Should have result');
    assert.ok(!response.result.isError, 'Should not be an error');
    assert.ok(
      response.result.content[0].text.includes('@e1'),
      'Should return @e1 ref'
    );
    assert.ok(
      response.result.content[0].text.includes('[h1]'),
      'Should find h1 element'
    );
  });

  test('browser_evaluate executes JavaScript', async () => {
    const response = await client.call('tools/call', {
      name: 'browser_evaluate',
      arguments: { expression: 'document.title' },
    });

    assert.ok(response.result, 'Should have result');
    assert.ok(!response.result.isError, 'Should not be an error');
    assert.ok(
      response.result.content[0].text.includes('Example Domain'),
      'Should return page title'
    );
  });

  test('browser_screenshot returns image', async () => {
    const response = await client.call('tools/call', {
      name: 'browser_screenshot',
      arguments: {},
    });

    assert.ok(response.result, 'Should have result');
    assert.ok(!response.result.isError, 'Should not be an error');

    const content = response.result.content[0];
    assert.strictEqual(content.type, 'image', 'Should be image type');
    assert.strictEqual(content.mimeType, 'image/png', 'Should be PNG');
    assert.ok(content.data.length > 100, 'Should have base64 data');
  });

  test('browser_click clicks element', async () => {
    const response = await client.call('tools/call', {
      name: 'browser_click',
      arguments: { selector: 'a' },
    });

    assert.ok(response.result, 'Should have result');
    assert.ok(!response.result.isError, 'Should not be an error');
    assert.ok(
      response.result.content[0].text.includes('Clicked'),
      'Should confirm click'
    );
  });

  test('browser_quit closes session', async () => {
    const response = await client.call('tools/call', {
      name: 'browser_quit',
      arguments: {},
    });

    assert.ok(response.result, 'Should have result');
    assert.ok(!response.result.isError, 'Should not be an error');
    assert.ok(
      response.result.content[0].text.includes('closed'),
      'Should confirm close'
    );
  });

  test('browser_quit when no session returns gracefully', async () => {
    const response = await client.call('tools/call', {
      name: 'browser_quit',
      arguments: {},
    });

    assert.ok(response.result, 'Should have result');
    assert.ok(!response.result.isError, 'Should not be an error');
  });
});

describe('MCP Server: New Tools', () => {
  let client;

  before(async () => {
    client = new MCPClient();
    await client.start();
    await client.call('initialize', { capabilities: {} });

    // Navigate to example.com for testing
    await client.call('tools/call', {
      name: 'browser_navigate',
      arguments: { url: 'https://example.com' },
    });
  });

  after(() => {
    client.stop();
  });

  test('browser_get_text returns page text', async () => {
    const response = await client.call('tools/call', {
      name: 'browser_get_text',
      arguments: {},
    });

    assert.ok(response.result, 'Should have result');
    assert.ok(!response.result.isError, 'Should not be an error');
    assert.ok(
      response.result.content[0].text.includes('Example Domain'),
      'Should contain page text'
    );
  });

  test('browser_get_text with selector returns element text', async () => {
    const response = await client.call('tools/call', {
      name: 'browser_get_text',
      arguments: { selector: 'h1' },
    });

    assert.ok(response.result, 'Should have result');
    assert.ok(!response.result.isError, 'Should not be an error');
    assert.ok(
      response.result.content[0].text.includes('Example Domain'),
      'Should contain h1 text'
    );
  });

  test('browser_get_url returns current URL', async () => {
    const response = await client.call('tools/call', {
      name: 'browser_get_url',
      arguments: {},
    });

    assert.ok(response.result, 'Should have result');
    assert.ok(!response.result.isError, 'Should not be an error');
    assert.ok(
      response.result.content[0].text.includes('example.com'),
      'Should contain example.com URL'
    );
  });

  test('browser_get_title returns page title', async () => {
    const response = await client.call('tools/call', {
      name: 'browser_get_title',
      arguments: {},
    });

    assert.ok(response.result, 'Should have result');
    assert.ok(!response.result.isError, 'Should not be an error');
    assert.ok(
      response.result.content[0].text.includes('Example Domain'),
      'Should return page title'
    );
  });

  test('browser_get_html returns page HTML', async () => {
    const response = await client.call('tools/call', {
      name: 'browser_get_html',
      arguments: { selector: 'h1' },
    });

    assert.ok(response.result, 'Should have result');
    assert.ok(!response.result.isError, 'Should not be an error');
    assert.ok(
      response.result.content[0].text.includes('Example Domain'),
      'Should contain HTML'
    );
  });

  test('browser_find_all returns array of elements', async () => {
    const response = await client.call('tools/call', {
      name: 'browser_find_all',
      arguments: { selector: 'p' },
    });

    assert.ok(response.result, 'Should have result');
    assert.ok(!response.result.isError, 'Should not be an error');
    assert.ok(
      response.result.content[0].text.includes('@e1'),
      'Should contain @e1 ref'
    );
    assert.ok(
      response.result.content[0].text.includes('[p]'),
      'Should contain [p] tag label'
    );
  });

  test('browser_wait succeeds for existing element', async () => {
    const response = await client.call('tools/call', {
      name: 'browser_wait',
      arguments: { selector: 'h1', state: 'visible' },
    });

    assert.ok(response.result, 'Should have result');
    assert.ok(!response.result.isError, 'Should not be an error');
    assert.ok(
      response.result.content[0].text.includes('reached state'),
      'Should confirm wait succeeded'
    );
  });

  test('browser_hover hovers over element', async () => {
    const response = await client.call('tools/call', {
      name: 'browser_hover',
      arguments: { selector: 'a' },
    });

    assert.ok(response.result, 'Should have result');
    assert.ok(!response.result.isError, 'Should not be an error');
    assert.ok(
      response.result.content[0].text.includes('Hovered'),
      'Should confirm hover'
    );
  });

  test('browser_scroll scrolls without error', async () => {
    const response = await client.call('tools/call', {
      name: 'browser_scroll',
      arguments: { direction: 'down', amount: 1 },
    });

    assert.ok(response.result, 'Should have result');
    assert.ok(!response.result.isError, 'Should not be an error');
    assert.ok(
      response.result.content[0].text.includes('Scrolled'),
      'Should confirm scroll'
    );
  });

  test('browser_keys presses Enter without error', async () => {
    const response = await client.call('tools/call', {
      name: 'browser_keys',
      arguments: { keys: 'Enter' },
    });

    assert.ok(response.result, 'Should have result');
    assert.ok(!response.result.isError, 'Should not be an error');
    assert.ok(
      response.result.content[0].text.includes('Pressed'),
      'Should confirm key press'
    );
  });

  test('browser_list_tabs returns tab list', async () => {
    const response = await client.call('tools/call', {
      name: 'browser_list_tabs',
      arguments: {},
    });

    assert.ok(response.result, 'Should have result');
    assert.ok(!response.result.isError, 'Should not be an error');
    assert.ok(
      response.result.content[0].text.includes('[0]'),
      'Should list at least one tab'
    );
  });

  test('browser_new_tab creates a tab and list shows 2', async () => {
    const newTabResponse = await client.call('tools/call', {
      name: 'browser_new_tab',
      arguments: {},
    });

    assert.ok(newTabResponse.result, 'Should have result');
    assert.ok(!newTabResponse.result.isError, 'Should not be an error');

    const listResponse = await client.call('tools/call', {
      name: 'browser_list_tabs',
      arguments: {},
    });

    assert.ok(listResponse.result.content[0].text.includes('[1]'), 'Should have 2 tabs');
  });

  test('browser_switch_tab switches to tab', async () => {
    const response = await client.call('tools/call', {
      name: 'browser_switch_tab',
      arguments: { index: 0 },
    });

    assert.ok(response.result, 'Should have result');
    assert.ok(!response.result.isError, 'Should not be an error');
    assert.ok(
      response.result.content[0].text.includes('Switched'),
      'Should confirm switch'
    );
  });

  test('browser_close_tab closes tab and list shows 1', async () => {
    const closeResponse = await client.call('tools/call', {
      name: 'browser_close_tab',
      arguments: { index: 1 },
    });

    assert.ok(closeResponse.result, 'Should have result');
    assert.ok(!closeResponse.result.isError, 'Should not be an error');

    const listResponse = await client.call('tools/call', {
      name: 'browser_list_tabs',
      arguments: {},
    });

    assert.ok(!listResponse.result.content[0].text.includes('[1]'), 'Should have 1 tab');
  });
});
