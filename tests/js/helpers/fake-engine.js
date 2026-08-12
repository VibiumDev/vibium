#!/usr/bin/env node
/**
 * A stand-in for the vibium binary, for tests that need to observe ordering
 * rather than a real browser.
 *
 * It installs the WebSocket monitor slowly and reports a socket only once that
 * install has been answered — exactly like the router, which replies to
 * vibium:page.onWebSocket after subscribing, adding the preload script and
 * injecting the monitor. A client that fires the install and moves on therefore
 * loses the event its next command produces, which is the bug in #351.
 *
 * Point a client at it with VIBIUM_BIN_PATH.
 */

const readline = require('readline');

const SETUP_DELAY_MS = Number(process.env.FAKE_ENGINE_SETUP_DELAY_MS || 300);
const FAIL_SETUP = process.env.FAKE_ENGINE_FAIL_SETUP === '1';

// The clients run `vibium is-installed` before launching.
if (process.argv[2] === 'is-installed') process.exit(0);

let monitorInstalled = false;
const write = (msg) => process.stdout.write(JSON.stringify(msg) + '\n');
const ok = (id, result = {}) => write({ id, type: 'success', result });

write({ method: 'vibium:lifecycle.ready', params: {} });

readline.createInterface({ input: process.stdin, crlfDelay: Infinity }).on('line', (line) => {
  if (!line.trim()) return;
  const { id, method, params = {} } = JSON.parse(line);

  switch (method) {
    case 'vibium:browser.page':
    case 'vibium:browser.newPage':
      return ok(id, { context: 'ctx-1', userContext: 'default' });

    case 'vibium:page.onWebSocket':
      return setTimeout(() => {
        if (FAIL_SETUP) {
          return write({ id, type: 'error', error: 'unsupported operation', message: 'no preload scripts here' });
        }
        monitorInstalled = true;
        ok(id);
      }, SETUP_DELAY_MS);

    case 'vibium:page.eval':
      // Stands in for `new WebSocket(...)`: only a live monitor sees it.
      if (monitorInstalled && String(params.expression || '').includes('openSocket')) {
        write({ method: 'vibium:ws.created', params: { context: 'ctx-1', id: 1, url: 'ws://127.0.0.1:1/live' } });
      }
      return ok(id, { value: null });

    default:
      return ok(id);
  }
});
