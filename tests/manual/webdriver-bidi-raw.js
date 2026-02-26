#!/usr/bin/env node

/**
 * Tests the raw WebDriver BiDi commands from docs/tutorials/webdriver-bidi-raw.md
 *
 * Launches Firefox, connects via WebSocket, sends each tutorial command,
 * and validates the responses.
 *
 * Prerequisites: firefox
 *
 * Usage: node tests/manual/webdriver-bidi-raw.js
 */

const { execSync, spawn } = require('child_process');
const WebSocket = require('ws');

const PASS = '\x1b[32mPASS\x1b[0m';
const FAIL = '\x1b[31mFAIL\x1b[0m';
const BOLD = '\x1b[1m';
const RESET = '\x1b[0m';

const PORT = 9333;

let firefoxProc = null;
let ws = null;
let failures = 0;

function section(title) {
  console.log(`\n${BOLD}━━━ ${title} ━━━${RESET}`);
}

function check(label, condition, resp) {
  if (condition) {
    console.log(`  ${PASS} ${label}`);
  } else {
    console.log(`  ${FAIL} ${label}`);
    console.log(`    Response: ${JSON.stringify(resp)}`);
    failures++;
  }
}

function sleep(ms) {
  return new Promise(r => setTimeout(r, ms));
}

// Find Firefox binary
function findFirefox() {
  const paths = [
    '/Applications/Firefox.app/Contents/MacOS/firefox',
    '/usr/bin/firefox',
    '/usr/bin/firefox-esr',
    '/snap/bin/firefox',
  ];
  for (const p of paths) {
    try { execSync(`test -x "${p}"`); return p; } catch {}
  }
  // Try PATH
  try { return execSync('which firefox').toString().trim(); } catch {}
  return null;
}

async function main() {
  // ─── Check prerequisites ───

  const firefoxPath = findFirefox();
  if (!firefoxPath) {
    console.error('Firefox not found. Install Firefox to run this test.');
    process.exit(1);
  }

  // ─── Launch Firefox ───

  section(`Launch Firefox with BiDi on port ${PORT}`);

  firefoxProc = spawn(firefoxPath, [
    `--remote-debugging-port=${PORT}`,
  ], { stdio: 'ignore' });

  console.log(`  Firefox PID: ${firefoxProc.pid}`);

  // Wait for the port to be ready
  let ready = false;
  for (let i = 0; i < 30; i++) {
    try {
      const net = require('net');
      await new Promise((resolve, reject) => {
        const sock = net.connect(PORT, '127.0.0.1', () => { sock.destroy(); resolve(); });
        sock.on('error', reject);
      });
      console.log(`  Port ${PORT} ready after ${i + 1}s`);
      ready = true;
      break;
    } catch {
      await sleep(1000);
    }
  }

  if (!ready) {
    console.log(`  ${FAIL} Firefox did not start on port ${PORT}`);
    process.exit(1);
  }

  // ─── Connect via WebSocket ───

  section(`Connect to ws://localhost:${PORT}/session`);

  ws = new WebSocket(`ws://localhost:${PORT}/session`);
  await new Promise((resolve, reject) => {
    ws.on('open', resolve);
    ws.on('error', reject);
  });

  // Message queue: responses keyed by id, events buffered separately
  const pending = new Map();
  ws.on('message', (data) => {
    const msg = JSON.parse(data.toString());
    if (msg.id !== undefined && pending.has(msg.id)) {
      pending.get(msg.id)(msg);
      pending.delete(msg.id);
    }
    // Events (no id) are ignored — we don't need them for this test
  });

  function send(cmd) {
    return new Promise((resolve, reject) => {
      const timeout = setTimeout(() => {
        pending.delete(cmd.id);
        reject(new Error(`Timeout waiting for response to id=${cmd.id} method=${cmd.method}`));
      }, 15000);

      pending.set(cmd.id, (msg) => {
        clearTimeout(timeout);
        resolve(msg);
      });

      ws.send(JSON.stringify(cmd));
    });
  }

  console.log(`  ${PASS} Connected`);

  // ─── Step 3: session.new ───

  section('Step 3: session.new');

  let resp = await send({ id: 1, method: 'session.new', params: { capabilities: {} } });

  check('Returns sessionId', resp.result?.sessionId != null, resp);
  check('Returns capabilities', resp.result?.capabilities != null, resp);

  // ─── Step 4: browsingContext.getTree ───

  section('Step 4: browsingContext.getTree');

  resp = await send({ id: 2, method: 'browsingContext.getTree', params: {} });

  check('Returns contexts array', resp.result?.contexts?.length > 0, resp);

  const context = resp.result?.contexts?.[0]?.context;
  check('Context ID is a non-empty string', typeof context === 'string' && context.length > 0, resp);
  console.log(`  Context: ${context}`);

  // ─── Step 5: browsingContext.navigate ───

  section('Step 5: browsingContext.navigate');

  resp = await send({
    id: 3, method: 'browsingContext.navigate',
    params: { context, url: 'https://example.com', wait: 'complete' },
  });

  check('Navigation succeeds', resp.result?.url === 'https://example.com/', resp);

  // ─── Step 6: script.evaluate (document.title) ───

  section('Step 6: script.evaluate — document.title');

  resp = await send({
    id: 4, method: 'script.evaluate',
    params: { expression: 'document.title', target: { context }, awaitPromise: false },
  });

  check('Title is "Example Domain"', resp.result?.result?.value === 'Example Domain', resp);
  check('Type is string', resp.result?.result?.type === 'string', resp);

  // ─── Step 7: script.evaluate (getBoundingClientRect) ───

  section('Step 7: script.evaluate — getBoundingClientRect');

  // DOMRect properties live on the prototype, so BiDi won't serialize them.
  // Extract into a plain object (same approach as the tutorial).
  resp = await send({
    id: 5, method: 'script.evaluate',
    params: {
      expression: "(() => { const r = document.querySelector('a').getBoundingClientRect(); return { x: r.x, y: r.y, width: r.width, height: r.height }; })()",
      target: { context },
      awaitPromise: false,
    },
  });

  check('Returns an object', resp.result?.result?.type === 'object', resp);

  // BiDi serializes as [["x", {type, value}], ["y", {type, value}], ...]
  const rectValue = resp.result?.result?.value;
  let x, y, width, height;
  if (Array.isArray(rectValue)) {
    const map = Object.fromEntries(rectValue.map(([k, v]) => [k, v?.value]));
    x = map.x; y = map.y; width = map.width; height = map.height;
  }

  check('Has x, y, width, height', x != null && y != null && width != null && height != null, { x, y, width, height });
  console.log(`  Bounding box: x=${x}, y=${y}, width=${width}, height=${height}`);

  // ─── Step 7b: input.performActions (click) ───

  section('Step 7b: input.performActions — click the link');

  const clickX = Math.round(x + width / 2);
  const clickY = Math.round(y + height / 2);
  console.log(`  Click coordinates: (${clickX}, ${clickY})`);

  resp = await send({
    id: 6, method: 'input.performActions',
    params: {
      context,
      actions: [{
        type: 'pointer', id: 'mouse',
        actions: [
          { type: 'pointerMove', x: clickX, y: clickY },
          { type: 'pointerDown', button: 0 },
          { type: 'pointerUp', button: 0 },
        ],
      }],
    },
  });

  check('performActions succeeds (no error)', resp.error == null, resp);

  // Wait for navigation
  await sleep(2000);

  // Verify we navigated away
  resp = await send({
    id: 60, method: 'script.evaluate',
    params: { expression: 'document.location.href', target: { context }, awaitPromise: false },
  });
  const currentUrl = resp.result?.result?.value;
  console.log(`  Current URL: ${currentUrl}`);
  check('Click navigated away from example.com', currentUrl !== 'https://example.com/', resp);

  // ─── Step 8: browser.close ───

  section('Step 8: browser.close');

  // browser.close will kill the connection, so we don't await the response normally
  ws.send(JSON.stringify({ id: 7, method: 'browser.close', params: {} }));
  await sleep(2000);

  // Check that Firefox exited
  let exited = false;
  try { process.kill(firefoxProc.pid, 0); } catch { exited = true; }
  check('Firefox closed', exited, {});
  if (exited) firefoxProc = null;

  // ─── Summary ───

  section('Summary');

  if (failures === 0) {
    console.log(`  ${BOLD}All tutorial commands verified.${RESET}`);
  } else {
    console.log(`  ${BOLD}${failures} check(s) failed.${RESET}`);
    process.exit(1);
  }
}

main().catch(err => {
  console.error(`\n${FAIL} Error: ${err.message}`);
  console.error(err.stack);
  process.exit(1);
}).finally(() => {
  if (ws) try { ws.close(); } catch {}
  if (firefoxProc) try { firefoxProc.kill(); } catch {}
});
