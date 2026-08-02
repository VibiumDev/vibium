/**
 * CLI Tests: commands against a prompt-blocked context
 *
 * The CLI runs over the agent path (daemon -> agent.Handlers -> bidi.Client),
 * not the proxy the JS client uses. Prompt tracking has to be wired into both,
 * so this covers the half a JS-client test cannot reach (#151).
 */

const { test, describe, before, after } = require('node:test');
const assert = require('node:assert');
const { execSync, spawn } = require('node:child_process');
const path = require('path');
const { VIBIUM } = require('../helpers');

let serverProcess, baseURL;

function run(args) {
  try {
    return execSync(`${VIBIUM} ${args} 2>&1`, { encoding: 'utf-8', timeout: 90000 });
  } catch (e) {
    // Non-zero exit is expected for the blocked command; keep its output.
    return (e.stdout || '') + (e.stderr || '');
  }
}

before(async () => {
  serverProcess = spawn('node', [path.join(__dirname, '../helpers/test-server.js')], {
    stdio: ['pipe', 'pipe', 'pipe'],
  });
  baseURL = await new Promise((resolve) => {
    serverProcess.stdout.once('data', (data) => resolve(data.toString().trim()));
  });
});

after(() => {
  // Leave no dialog behind for the next test file sharing this daemon.
  run('dialog accept');
  if (serverProcess) serverProcess.kill();
});

describe('CLI: prompt-blocked context', () => {
  test('a command fails fast with an actionable error', () => {
    run(`go ${baseURL}/example`);

    // Open an alert without handling it. The alert fires after eval returns,
    // so eval itself is not the command that blocks.
    run(`eval "setTimeout(() => alert('blocked'), 100); 'ok'"`);
    execSync('sleep 1');

    const started = Date.now();
    const out = run('title');
    const elapsed = Date.now() - started;

    assert.match(
      out,
      /blocked by an open .* dialog/,
      `expected an actionable prompt error, got: ${out.slice(0, 200)}`
    );
    assert.ok(
      elapsed < 10000,
      `expected a fast failure, took ${elapsed}ms (the BiDi command timeout is 60s)`
    );
  });

  test('accepting the dialog unblocks the context', () => {
    const accepted = run('dialog accept');
    assert.doesNotMatch(accepted, /blocked by an open/, 'dialog accept must never be blocked');

    const out = run('title');
    assert.doesNotMatch(
      out,
      /blocked by an open/,
      `context should be usable after the dialog is accepted, got: ${out.slice(0, 200)}`
    );
  });
});
