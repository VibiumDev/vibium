/**
 * MCP Server Tests
 * Tests the vibium mcp command via stdin/stdout JSON-RPC
 */

const { test, describe, before, after } = require('node:test');
const assert = require('node:assert');
const { spawn, execFileSync, execSync } = require('node:child_process');
const fs = require('node:fs');
const path = require('node:path');
const os = require('node:os');
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

  receive(timeout = 120000) {
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

  test('tools/list returns all 85 browser tools', async () => {
    const response = await client.call('tools/list', {});

    assert.ok(response.result, 'Should have result');
    assert.ok(response.result.tools, 'Should have tools array');
    assert.strictEqual(response.result.tools.length, 85, 'Should have 85 tools');

    const toolNames = response.result.tools.map(t => t.name);
    const expectedTools = [
      'browser_start', 'browser_navigate', 'browser_click', 'browser_type',
      'browser_screenshot', 'browser_find', 'browser_evaluate', 'browser_stop',
      'browser_get_text', 'browser_get_url', 'browser_get_title',
      'browser_get_html', 'browser_find_all', 'browser_wait',
      'browser_hover', 'browser_select', 'browser_scroll', 'browser_keys',
      'browser_new_page', 'browser_list_pages', 'browser_switch_page', 'browser_close_page',
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
      'browser_record_start', 'browser_record_stop',
      'browser_record_start_group', 'browser_record_stop_group',
      'browser_record_start_chunk', 'browser_record_stop_chunk',
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

  test('browser_start when already running returns success', async () => {
    const response = await client.call('tools/call', {
      name: 'browser_start',
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

  test('browser_stop closes session', async () => {
    const response = await client.call('tools/call', {
      name: 'browser_stop',
      arguments: {},
    });

    assert.ok(response.result, 'Should have result');
    assert.ok(!response.result.isError, 'Should not be an error');
    assert.ok(
      response.result.content[0].text.includes('closed'),
      'Should confirm close'
    );
  });

  test('browser_stop when no session returns gracefully', async () => {
    const response = await client.call('tools/call', {
      name: 'browser_stop',
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

  after(async () => {
    await client.call('tools/call', { name: 'browser_stop', arguments: {} });
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

  test('browser_list_pages returns page list', async () => {
    const response = await client.call('tools/call', {
      name: 'browser_list_pages',
      arguments: {},
    });

    assert.ok(response.result, 'Should have result');
    assert.ok(!response.result.isError, 'Should not be an error');
    assert.ok(
      response.result.content[0].text.includes('[0]'),
      'Should list at least one page'
    );
  });

  test('browser_new_page creates a page and list shows 2', async () => {
    const newPageResponse = await client.call('tools/call', {
      name: 'browser_new_page',
      arguments: {},
    });

    assert.ok(newPageResponse.result, 'Should have result');
    assert.ok(!newPageResponse.result.isError, 'Should not be an error');

    const listResponse = await client.call('tools/call', {
      name: 'browser_list_pages',
      arguments: {},
    });

    assert.ok(listResponse.result.content[0].text.includes('[1]'), 'Should have 2 pages');
  });

  test('browser_switch_page switches to page', async () => {
    const response = await client.call('tools/call', {
      name: 'browser_switch_page',
      arguments: { index: 0 },
    });

    assert.ok(response.result, 'Should have result');
    assert.ok(!response.result.isError, 'Should not be an error');
    assert.ok(
      response.result.content[0].text.includes('Switched'),
      'Should confirm switch'
    );
  });

  test('browser_close_page closes page and list shows 1', async () => {
    const closeResponse = await client.call('tools/call', {
      name: 'browser_close_page',
      arguments: { index: 1 },
    });

    assert.ok(closeResponse.result, 'Should have result');
    assert.ok(!closeResponse.result.isError, 'Should not be an error');

    const listResponse = await client.call('tools/call', {
      name: 'browser_list_pages',
      arguments: {},
    });

    assert.ok(!listResponse.result.content[0].text.includes('[1]'), 'Should have 1 page');
  });
});

describe('MCP Server: Viewport & Window', () => {
  /** @type {MCPClient} */
  let client;

  before(async () => {
    client = new MCPClient();
    await client.start();
    await client.call('initialize', {
      protocolVersion: '2024-11-05',
      capabilities: {},
      clientInfo: { name: 'test-viewport', version: '1.0.0' },
    });
  });

  after(async () => {
    await client.call('tools/call', { name: 'browser_stop', arguments: {} });
    client.stop();
  });

  test('browser_get_viewport returns width and height', async () => {
    const response = await client.call('tools/call', {
      name: 'browser_get_viewport',
      arguments: {},
    });
    assert.ok(response.result, 'Should have result');
    assert.ok(!response.result.isError, 'Should not be an error');
    const text = response.result.content[0].text;
    assert.ok(text.includes('width'), 'Should contain width');
    assert.ok(text.includes('height'), 'Should contain height');
  });

  test('browser_set_viewport then browser_get_viewport', async () => {
    await client.call('tools/call', {
      name: 'browser_set_viewport',
      arguments: { width: 800, height: 600 },
    });

    const response = await client.call('tools/call', {
      name: 'browser_get_viewport',
      arguments: {},
    });
    assert.ok(!response.result.isError, 'Should not be an error');
    const text = response.result.content[0].text;
    assert.ok(text.includes('800'), 'Should reflect width 800');
    assert.ok(text.includes('600'), 'Should reflect height 600');
  });

  test('browser_get_window returns state and dimensions', async () => {
    const response = await client.call('tools/call', {
      name: 'browser_get_window',
      arguments: {},
    });
    assert.ok(response.result, 'Should have result');
    assert.ok(!response.result.isError, 'Should not be an error');
    const text = response.result.content[0].text;
    assert.ok(text.includes('width'), 'Should contain width');
    assert.ok(text.includes('height'), 'Should contain height');
  });

  test('browser_set_window then browser_get_window', async () => {
    await client.call('tools/call', {
      name: 'browser_set_window',
      arguments: { width: 900, height: 700 },
    });

    const response = await client.call('tools/call', {
      name: 'browser_get_window',
      arguments: {},
    });
    assert.ok(!response.result.isError, 'Should not be an error');
    const text = response.result.content[0].text;
    assert.ok(text.includes('900'), 'Should reflect width 900');
    assert.ok(text.includes('700'), 'Should reflect height 700');
  });
});

// --- Recording helpers ---

function unzipRecording(zipPath) {
  const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), 'vibium-mcp-rec-'));
  execSync(`unzip -o "${zipPath}" -d "${tmpDir}/extracted"`, { stdio: 'pipe' });
  return { tmpDir, extractedDir: path.join(tmpDir, 'extracted') };
}

function cleanupDir(dir) {
  fs.rmSync(dir, { recursive: true, force: true });
}

function readRecordingEvents(extractedDir) {
  const files = fs.readdirSync(extractedDir).filter(f => f.endsWith('.trace'));
  const events = [];
  for (const file of files) {
    const content = fs.readFileSync(path.join(extractedDir, file), 'utf-8');
    for (const line of content.split('\n')) {
      if (line.trim()) {
        events.push(JSON.parse(line));
      }
    }
  }
  return events;
}

function readNetworkEvents(extractedDir) {
  const files = fs.readdirSync(extractedDir).filter(f => f.endsWith('.network'));
  const events = [];
  for (const file of files) {
    const content = fs.readFileSync(path.join(extractedDir, file), 'utf-8');
    for (const line of content.split('\n')) {
      if (line.trim()) {
        events.push(JSON.parse(line));
      }
    }
  }
  return events;
}

describe('MCP Server: Recording', { timeout: 120000 }, () => {
  let client;
  const tmpFiles = [];

  function tmpPath(name) {
    const p = path.join(os.tmpdir(), `vibium-mcp-rec-${Date.now()}-${name}`);
    tmpFiles.push(p);
    return p;
  }

  before(async () => {
    client = new MCPClient();
    await client.start();
    await client.call('initialize', { capabilities: {} });

    // Navigate to example.com so browser is launched
    await client.call('tools/call', {
      name: 'browser_navigate',
      arguments: { url: 'https://example.com' },
    });
  });

  after(async () => {
    await client.call('tools/call', { name: 'browser_stop', arguments: {} });
    client.stop();
    for (const f of tmpFiles) {
      try { fs.unlinkSync(f); } catch {}
    }
  });

  test('browser_record_start begins recording', async () => {
    const response = await client.call('tools/call', {
      name: 'browser_record_start',
      arguments: {},
    });

    assert.ok(response.result, 'Should have result');
    assert.ok(!response.result.isError, 'Should not be an error');
    assert.ok(
      response.result.content[0].text.includes('started'),
      'Should confirm recording started'
    );

    // Clean up
    const stopPath = tmpPath('start-test.zip');
    await client.call('tools/call', {
      name: 'browser_record_stop',
      arguments: { path: stopPath },
    });
  });

  test('browser_record_start rejects when already recording', async () => {
    // Start recording
    await client.call('tools/call', {
      name: 'browser_record_start',
      arguments: {},
    });

    // Try to start again
    const response = await client.call('tools/call', {
      name: 'browser_record_start',
      arguments: {},
    });

    assert.ok(response.result, 'Should have result');
    assert.strictEqual(response.result.isError, true, 'Should be an error');
    assert.ok(
      response.result.content[0].text.includes('already recording'),
      'Should say already recording'
    );

    // Clean up
    const stopPath = tmpPath('double-start.zip');
    await client.call('tools/call', {
      name: 'browser_record_stop',
      arguments: { path: stopPath },
    });
  });

  test('browser_record_stop saves valid Playwright-compatible zip', async () => {
    // Start recording
    await client.call('tools/call', {
      name: 'browser_record_start',
      arguments: {},
    });

    // Navigate to generate events
    await client.call('tools/call', {
      name: 'browser_navigate',
      arguments: { url: 'https://example.com' },
    });

    // Stop and save
    const zipPath = tmpPath('valid-zip.zip');
    const response = await client.call('tools/call', {
      name: 'browser_record_stop',
      arguments: { path: zipPath },
    });

    assert.ok(!response.result.isError, 'Should not be an error');
    assert.ok(
      response.result.content[0].text.includes('Recording saved'),
      'Should confirm save'
    );

    // Verify file exists and has content
    assert.ok(fs.existsSync(zipPath), 'Zip file should exist');
    assert.ok(fs.statSync(zipPath).size > 0, 'Zip file should not be empty');

    // Unzip and verify structure
    const { tmpDir, extractedDir } = unzipRecording(zipPath);
    try {
      const files = fs.readdirSync(extractedDir);
      assert.ok(files.some(f => f.endsWith('.trace')), 'Should have .trace file');
      assert.ok(files.some(f => f.endsWith('.network')), 'Should have .network file');

      // Verify trace events
      const events = readRecordingEvents(extractedDir);
      assert.ok(events.length > 0, 'Should have recording events');
      assert.strictEqual(events[0].type, 'context-options');
      assert.strictEqual(events[0].browserName, 'chromium');

      // Verify before/after action events (unified with API path)
      const beforeEvents = events.filter(e => e.type === 'before' && e.method !== 'group');
      assert.ok(beforeEvents.length > 0, 'Should have before events for actions');
      assert.ok(beforeEvents[0].callId, 'before event should have callId');
      assert.ok(beforeEvents[0].title, 'before event should have title');

      const afterEvents = events.filter(e => e.type === 'after');
      assert.ok(afterEvents.length > 0, 'Should have after events for actions');
      assert.ok(afterEvents[0].callId, 'after event should have callId');
      assert.ok(afterEvents[0].endTime, 'after event should have endTime');

      // Verify before/after callIds match
      const beforeCallId = beforeEvents[0].callId;
      const matchingAfter = afterEvents.find(e => e.callId === beforeCallId);
      assert.ok(matchingAfter, 'Should have matching after event for first before event');

      // Verify network events
      const networkEvents = readNetworkEvents(extractedDir);
      assert.ok(networkEvents.length > 0, 'Should have network events (resource-snapshot)');
      assert.strictEqual(networkEvents[0].type, 'resource-snapshot', 'Network event should be resource-snapshot');
    } finally {
      cleanupDir(tmpDir);
    }
  });

  test('browser_record_stop rejects when not recording', async () => {
    const response = await client.call('tools/call', {
      name: 'browser_record_stop',
      arguments: {},
    });

    assert.ok(response.result, 'Should have result');
    assert.strictEqual(response.result.isError, true, 'Should be an error');
    assert.ok(
      response.result.content[0].text.includes('no recording'),
      'Should say no recording in progress'
    );
  });

  test('browser_record_start with screenshots captures per-action screenshots', async () => {
    await client.call('tools/call', {
      name: 'browser_record_start',
      arguments: { screenshots: true },
    });

    // Perform actions to trigger screenshots
    await client.call('tools/call', {
      name: 'browser_navigate',
      arguments: { url: 'https://example.com' },
    });
    await client.call('tools/call', {
      name: 'browser_click',
      arguments: { selector: 'a' },
    });

    const zipPath = tmpPath('screenshots.zip');
    await client.call('tools/call', {
      name: 'browser_record_stop',
      arguments: { path: zipPath },
    });

    const { tmpDir, extractedDir } = unzipRecording(zipPath);
    try {
      // Check for resources directory with screenshot files
      const resourcesDir = path.join(extractedDir, 'resources');
      assert.ok(fs.existsSync(resourcesDir), 'resources/ directory should exist');
      const resources = fs.readdirSync(resourcesDir);
      assert.ok(resources.length > 0, 'Should have screenshot resources');

      // Check for screencast-frame events in trace
      const events = readRecordingEvents(extractedDir);
      const frames = events.filter(e => e.type === 'screencast-frame');
      assert.ok(frames.length > 0, 'Should have screencast-frame events');
      assert.ok(frames[0].sha1, 'screencast-frame should have sha1');
      assert.ok(frames[0].width > 0, 'screencast-frame should have width');
      assert.ok(frames[0].height > 0, 'screencast-frame should have height');

      // Verify before/after action events are present alongside screenshots
      const beforeEvents = events.filter(e => e.type === 'before' && e.method !== 'group');
      assert.ok(beforeEvents.length >= 2, `Should have >= 2 before events (navigate + click), got ${beforeEvents.length}`);

      // Check action titles map correctly
      const navBefore = beforeEvents.find(e => e.title === 'Page.navigate');
      assert.ok(navBefore, 'Should have before event with title Page.navigate');
      const clickBefore = beforeEvents.find(e => e.title === 'Element.click');
      assert.ok(clickBefore, 'Should have before event with title Element.click');

      // Check input event with point and box for click action
      const inputEvents = events.filter(e => e.type === 'input');
      assert.ok(inputEvents.length > 0, 'Should have input events for click actions');
      assert.ok(inputEvents[0].point, 'input event should have point');
      assert.ok(typeof inputEvents[0].point.x === 'number', 'point.x should be a number');
      assert.ok(typeof inputEvents[0].point.y === 'number', 'point.y should be a number');
      assert.ok(inputEvents[0].box, 'input event should have box');
      assert.ok(inputEvents[0].box.width > 0, 'box.width should be > 0');
      assert.ok(inputEvents[0].box.height > 0, 'box.height should be > 0');
    } finally {
      cleanupDir(tmpDir);
    }
  });

  test('browser_record_start_group / stop_group adds group markers', async () => {
    await client.call('tools/call', {
      name: 'browser_record_start',
      arguments: {},
    });

    // Start group
    const groupResp = await client.call('tools/call', {
      name: 'browser_record_start_group',
      arguments: { name: 'test-group' },
    });
    assert.ok(!groupResp.result.isError, 'start_group should not error');
    assert.ok(groupResp.result.content[0].text.includes('test-group'), 'Should confirm group name');

    // Perform action inside group
    await client.call('tools/call', {
      name: 'browser_navigate',
      arguments: { url: 'https://example.com' },
    });

    // Stop group
    const stopGroupResp = await client.call('tools/call', {
      name: 'browser_record_stop_group',
      arguments: {},
    });
    assert.ok(!stopGroupResp.result.isError, 'stop_group should not error');

    // Stop recording
    const zipPath = tmpPath('groups.zip');
    await client.call('tools/call', {
      name: 'browser_record_stop',
      arguments: { path: zipPath },
    });

    const { tmpDir, extractedDir } = unzipRecording(zipPath);
    try {
      const events = readRecordingEvents(extractedDir);

      // Look for group before event
      const beforeEvents = events.filter(e => e.type === 'before' && e.title === 'test-group' && e.method === 'tracingGroup');
      assert.ok(beforeEvents.length > 0, 'Should have before event with title "test-group" and method "tracingGroup"');

      // Look for matching after event
      const callId = beforeEvents[0].callId;
      const afterEvents = events.filter(e => e.type === 'after' && e.callId === callId);
      assert.ok(afterEvents.length > 0, 'Should have matching after event with same callId');
    } finally {
      cleanupDir(tmpDir);
    }
  });

  test('browser_record_start_chunk / stop_chunk saves chunk zip', async () => {
    await client.call('tools/call', {
      name: 'browser_record_start',
      arguments: {},
    });

    // First chunk: navigate
    await client.call('tools/call', {
      name: 'browser_navigate',
      arguments: { url: 'https://example.com' },
    });

    // Stop first chunk
    const chunkPath1 = tmpPath('chunk1.zip');
    const chunk1Resp = await client.call('tools/call', {
      name: 'browser_record_stop_chunk',
      arguments: { path: chunkPath1 },
    });
    assert.ok(!chunk1Resp.result.isError, 'stop_chunk should not error');
    assert.ok(chunk1Resp.result.content[0].text.includes('Chunk saved'), 'Should confirm chunk saved');

    // Start second chunk
    await client.call('tools/call', {
      name: 'browser_record_start_chunk',
      arguments: {},
    });

    // Navigate again
    await client.call('tools/call', {
      name: 'browser_navigate',
      arguments: { url: 'https://example.com' },
    });

    // Stop second chunk
    const chunkPath2 = tmpPath('chunk2.zip');
    await client.call('tools/call', {
      name: 'browser_record_stop_chunk',
      arguments: { path: chunkPath2 },
    });

    // Verify both files exist and have events
    assert.ok(fs.existsSync(chunkPath1), 'Chunk 1 should exist');
    assert.ok(fs.existsSync(chunkPath2), 'Chunk 2 should exist');

    const { tmpDir: td1, extractedDir: ed1 } = unzipRecording(chunkPath1);
    const { tmpDir: td2, extractedDir: ed2 } = unzipRecording(chunkPath2);
    try {
      const events1 = readRecordingEvents(ed1);
      assert.ok(events1.length > 0, 'Chunk 1 should have events');
      const events2 = readRecordingEvents(ed2);
      assert.ok(events2.length > 0, 'Chunk 2 should have events');
    } finally {
      cleanupDir(td1);
      cleanupDir(td2);
    }

    // Stop recording to clean up
    const stopPath = tmpPath('chunks-final.zip');
    await client.call('tools/call', {
      name: 'browser_record_stop',
      arguments: { path: stopPath },
    });
  });

  test('browser_record_start with title sets trace viewer title', async () => {
    await client.call('tools/call', {
      name: 'browser_record_start',
      arguments: { title: 'My Test Title' },
    });

    await client.call('tools/call', {
      name: 'browser_navigate',
      arguments: { url: 'https://example.com' },
    });

    const zipPath = tmpPath('title.zip');
    await client.call('tools/call', {
      name: 'browser_record_stop',
      arguments: { path: zipPath },
    });

    const { tmpDir, extractedDir } = unzipRecording(zipPath);
    try {
      const events = readRecordingEvents(extractedDir);
      assert.strictEqual(events[0].title, 'My Test Title', 'First event should have the custom title');
    } finally {
      cleanupDir(tmpDir);
    }
  });
});

describe('MCP Server: regression fixes', () => {
  let client;

  before(async () => {
    client = new MCPClient();
    await client.start();
    await client.call('initialize', { capabilities: {} });
    await client.call('tools/call', { name: 'browser_navigate', arguments: { url: 'https://example.com' } });
  });

  after(() => {
    client.stop();
  });

  const tool = (name, args = {}) => client.call('tools/call', { name, arguments: args });

  test('browser_sleep accepts a stringified number (#85)', async () => {
    const r = await tool('browser_sleep', { ms: '100' });
    assert.ok(!r.result.isError, `ms:"100" should be accepted: ${JSON.stringify(r.result)}`);
  });

  test('browser_count returns an integer (#149)', async () => {
    const r = await tool('browser_count', { selector: 'a' });
    assert.ok(!r.result.isError, 'count should not error');
    assert.ok(/^\d+$/.test(r.result.content[0].text.trim()), `count should be an integer, got: ${r.result.content[0].text}`);
  });

  test('browser_evaluate returning "" gives {type:text,text:""} not invalid_union (#154)', async () => {
    const r = await tool('browser_evaluate', { expression: "[].join(',')" });
    assert.ok(!r.result.isError, `empty result should not error: ${JSON.stringify(r.result)}`);
    const c = r.result.content[0];
    assert.strictEqual(c.type, 'text');
    assert.strictEqual(c.text, '', 'empty string result must carry an explicit empty text field');
  });

  test('browser_get_attribute on an absent attribute does not error (#153)', async () => {
    const r = await tool('browser_get_attribute', { selector: 'h1', attribute: 'data-nonexistent' });
    assert.ok(!r.result.isError, `absent attribute should not error: ${JSON.stringify(r.result)}`);
    assert.strictEqual(r.result.content[0].type, 'text');
  });

  test('browser_get_text on an empty element returns "" (#157)', async () => {
    await tool('browser_navigate', { url: 'data:text/html,<div id="e"></div>' });
    const r = await tool('browser_get_text', { selector: '#e' });
    assert.ok(!r.result.isError, `empty text should not error: ${JSON.stringify(r.result)}`);
    assert.strictEqual(r.result.content[0].text, '');
    await tool('browser_navigate', { url: 'https://example.com' });
  });

  test('browser_set_cookie derives domain from the page when omitted (#152)', async () => {
    const r = await tool('browser_set_cookie', { name: 'tc', value: 'abc' });
    assert.ok(!r.result.isError, `set_cookie without domain should succeed: ${JSON.stringify(r.result)}`);
  });

  test('browser_storage_state parses cookies without crashing (#150)', async () => {
    const r = await tool('browser_storage_state', {});
    assert.ok(!r.result.isError, `storage_state should not crash: ${JSON.stringify(r.result)}`);
  });

  test('browser_screenshot annotate:true returns an image (#156)', async () => {
    const r = await tool('browser_screenshot', { annotate: true });
    assert.ok(!r.result.isError, `annotate should not crash: ${JSON.stringify(r.result)}`);
    const img = r.result.content.find((c) => c.type === 'image');
    assert.ok(img && img.data && img.data.length > 100, 'should return image data');
  });
});
