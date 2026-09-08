#!/usr/bin/env node
/**
 * A stand-in for the vibium binary, for tests that assert command ordering
 * rather than browser behavior.
 *
 * It answers vibium:page.onWebSocket after a delay and reports a socket only
 * once that install has been answered, like the real router, which replies
 * after subscribing and injecting the monitor. A client that fires the install
 * and moves on loses the event its next command produces (#351).
 *
 * Point a client at it with browser.start({ executablePath }).
 * FAKE_ENGINE_SETUP_DELAY_MS adjusts the install delay (default 300).
 * FAKE_ENGINE_FAIL_SETUP=1 makes every install fail; =once fails only the
 * first, so tests can pin that a failed install is retried.
 */

const readline = require('readline');

const SETUP_DELAY_MS = Number(process.env.FAKE_ENGINE_SETUP_DELAY_MS || 300);
const FAIL_SETUP = process.env.FAKE_ENGINE_FAIL_SETUP || '';

// The clients run `vibium is-installed` before launching.
if (process.argv[2] === 'is-installed') process.exit(0);

let monitorInstalled = false;
let installCount = 0;
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

    case 'vibium:page.onWebSocket': {
      // Counted before deciding failure, so __installCount below includes
      // failed attempts and a test can pin that a retry re-sent the install.
      installCount++;
      const fail = FAIL_SETUP === '1' || (FAIL_SETUP === 'once' && installCount === 1);
      return setTimeout(() => {
        if (fail) {
          return write({ id, type: 'error', error: 'unsupported operation', message: 'no preload scripts here' });
        }
        monitorInstalled = true;
        ok(id);
      }, SETUP_DELAY_MS);
    }

    case 'vibium:page.eval':
      // Test back-channel, not protocol behavior: reports how many installs
      // this engine was sent. Race-free because the eval goes through the
      // gate, so any pending install is answered before this is.
      if (String(params.expression || '') === '__installCount') {
        return ok(id, { value: installCount });
      }
      // Stands in for `new WebSocket(...)`: only a live monitor sees it.
      if (monitorInstalled && String(params.expression || '').includes('openSocket')) {
        write({ method: 'vibium:ws.created', params: { context: 'ctx-1', id: 1, url: 'ws://127.0.0.1:1/live' } });
      }
      return ok(id, { value: null });

    default:
      return ok(id);
  }
});
