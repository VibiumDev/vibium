/**
 * Remote Browser Connect Tests
 * Tests vibium start [url], stop, VIBIUM_CONNECT_URL env var,
 * and window error for remote browsers.
 */

const { test, describe, before, after } = require('node:test');
const assert = require('node:assert');
const { execSync, spawn } = require('node:child_process');
const http = require('node:http');
const net = require('node:net');
const path = require('node:path');
const { VIBIUM } = require('../helpers');

// Helper to run vibium and return trimmed output
function clicker(args, opts = {}) {
  const result = execSync(`${VIBIUM} ${args}`, {
    encoding: 'utf-8',
    timeout: opts.timeout || 60000,
    env: { ...process.env, ...opts.env },
  });
  return result.trim();
}

function clickerJSON(args, opts = {}) {
  const result = clicker(`--json ${args}`, opts);
  return JSON.parse(result);
}

// Run vibium command but don't throw on non-zero exit (for error responses)
function clickerJSONSafe(args, opts = {}) {
  try {
    return clickerJSON(args, opts);
  } catch (e) {
    // Command exited non-zero — parse the JSON from stdout
    if (e.stdout) {
      return JSON.parse(e.stdout.trim());
    }
    throw e;
  }
}

// Helper to stop daemon (ignore errors if not running)
function stopDaemon() {
  try {
    execSync(`${VIBIUM} daemon stop`, { encoding: 'utf-8', timeout: 10000 });
  } catch (e) {
    // Daemon may not be running
  }
}

// Find an available TCP port
function findAvailablePort() {
  return new Promise((resolve, reject) => {
    const server = net.createServer();
    server.listen(0, '127.0.0.1', () => {
      const port = server.address().port;
      server.close(() => resolve(port));
    });
    server.on('error', reject);
  });
}

// Wait for chromedriver to be ready on a port
function waitForChromedriver(port, timeoutMs = 10000) {
  const deadline = Date.now() + timeoutMs;
  return new Promise((resolve, reject) => {
    function poll() {
      if (Date.now() > deadline) {
        return reject(new Error(`Chromedriver not ready after ${timeoutMs}ms`));
      }
      http.get(`http://127.0.0.1:${port}/status`, (res) => {
        let data = '';
        res.on('data', (chunk) => { data += chunk; });
        res.on('end', () => {
          if (res.statusCode === 200) {
            resolve();
          } else {
            setTimeout(poll, 100);
          }
        });
      }).on('error', () => {
        setTimeout(poll, 100);
      });
    }
    poll();
  });
}

// Create a chromedriver session via HTTP and return the BiDi WebSocket URL.
// This simulates an already-running remote browser session.
function createChromedriverSession(port, chromePath) {
  return new Promise((resolve, reject) => {
    const body = JSON.stringify({
      capabilities: {
        alwaysMatch: {
          browserName: 'chrome',
          webSocketUrl: true,
          'goog:chromeOptions': {
            binary: chromePath,
            args: ['--headless=new', '--no-first-run', '--no-default-browser-check'],
          },
        },
      },
    });

    const req = http.request(
      {
        hostname: '127.0.0.1',
        port,
        path: '/session',
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Content-Length': Buffer.byteLength(body),
        },
      },
      (res) => {
        let data = '';
        res.on('data', (chunk) => { data += chunk; });
        res.on('end', () => {
          if (res.statusCode !== 200) {
            return reject(new Error(`Failed to create session: HTTP ${res.statusCode}: ${data}`));
          }
          const parsed = JSON.parse(data);
          const wsUrl = parsed.value?.capabilities?.webSocketUrl;
          const sessionId = parsed.value?.sessionId;
          if (!wsUrl) {
            return reject(new Error('No webSocketUrl in session response'));
          }
          resolve({ wsUrl, sessionId });
        });
      }
    );
    req.on('error', reject);
    req.write(body);
    req.end();
  });
}

// Get chromedriver and Chrome paths from vibium paths output
function getBrowserPaths() {
  const pathsOutput = clicker('paths');
  const cdMatch = pathsOutput.match(/Chromedriver:\s+(.+)/);
  const chromeMatch = pathsOutput.match(/Chrome:\s+(.+)/);
  assert.ok(cdMatch, 'Should find chromedriver path in vibium paths output');
  assert.ok(chromeMatch, 'Should find Chrome path in vibium paths output');
  return {
    chromedriverPath: cdMatch[1].trim(),
    chromePath: chromeMatch[1].trim(),
  };
}

let serverProcess, baseURL;

before(async () => {
  serverProcess = spawn('node', [path.join(__dirname, '../helpers/test-server.js')], {
    stdio: ['pipe', 'pipe', 'pipe'],
  });
  baseURL = await new Promise((resolve) => {
    serverProcess.stdout.once('data', (data) => {
      resolve(data.toString().trim());
    });
  });
});

after(() => {
  if (serverProcess) serverProcess.kill();
});

describe('Daemon: Remote browser connect', () => {
  let chromedriverProc;
  let chromedriverPort;
  let bidiWsUrl;

  before(async () => {
    stopDaemon();

    const { chromedriverPath, chromePath } = getBrowserPaths();

    // Find an available port and start chromedriver
    chromedriverPort = await findAvailablePort();
    chromedriverProc = spawn(chromedriverPath, [`--port=${chromedriverPort}`], {
      stdio: 'ignore',
    });

    await waitForChromedriver(chromedriverPort);

    // Create a session via HTTP to get a BiDi WebSocket URL with an active browser
    const session = await createChromedriverSession(chromedriverPort, chromePath);
    bidiWsUrl = session.wsUrl;
  });

  after(() => {
    stopDaemon();
    if (chromedriverProc) {
      chromedriverProc.kill('SIGKILL');
      chromedriverProc = null;
    }
  });

  test('vibium start [url] starts daemon in connect mode', () => {
    const result = clicker(`start ${bidiWsUrl} --headless`);
    assert.match(result, /Connected to/, 'Should confirm connection');

    // Verify daemon is running
    const status = clicker('daemon status');
    assert.match(status, /running/i, 'Daemon should be running');
  });

  test('navigation works through remote connection', () => {
    const result = clickerJSON(`go ${baseURL}/example`);
    assert.strictEqual(result.ok, true, 'Navigate should succeed');
    assert.ok(result.result.includes('/example'), 'Should confirm navigation');
  });

  test('window set returns error for remote browsers', () => {
    const result = clickerJSONSafe('window 800 600');
    assert.strictEqual(result.ok, false, 'window set should fail for remote browsers');
    assert.ok(
      result.error.toLowerCase().includes('not supported') ||
      result.error.toLowerCase().includes('remote'),
      'Should mention remote browser limitation'
    );
  });

  test('vibium stop stops daemon', () => {
    const result = clicker('stop');
    // stop calls browser_stop via daemon, which closes the session
    assert.ok(result, 'Should return a result');

    // Verify daemon is not running
    const status = clicker('daemon status');
    assert.match(status, /not running/i, 'Daemon should not be running');
  });
});

describe('Daemon: VIBIUM_CONNECT_URL env var', () => {
  let chromedriverProc;
  let chromedriverPort;
  let bidiWsUrl;

  before(async () => {
    stopDaemon();

    const { chromedriverPath, chromePath } = getBrowserPaths();

    // Start chromedriver on a random port
    chromedriverPort = await findAvailablePort();
    chromedriverProc = spawn(chromedriverPath, [`--port=${chromedriverPort}`], {
      stdio: 'ignore',
    });

    await waitForChromedriver(chromedriverPort);

    // Create a session to get BiDi WebSocket URL
    const session = await createChromedriverSession(chromedriverPort, chromePath);
    bidiWsUrl = session.wsUrl;
  });

  after(() => {
    stopDaemon();
    if (chromedriverProc) {
      chromedriverProc.kill('SIGKILL');
      chromedriverProc = null;
    }
  });

  test('VIBIUM_CONNECT_URL auto-starts daemon in connect mode', () => {
    // With no daemon running, set env var and run a command
    const result = clickerJSON(`go ${baseURL}/example --headless`, {
      env: { VIBIUM_CONNECT_URL: bidiWsUrl },
    });
    assert.strictEqual(result.ok, true, 'Navigate should succeed via env var auto-start');

    // Verify daemon is running
    const status = clicker('daemon status');
    assert.match(status, /running/i, 'Daemon should be running after env var auto-start');
  });
});
