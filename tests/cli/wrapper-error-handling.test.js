/**
 * CLI Wrapper Tests: error handling
 *
 * The JS wrapper at packages/vibium/bin/cli.js shells out to the platform
 * binary via execFileSync. When the binary exits non-zero, the wrapper must
 * forward the exit status without dumping a Node stack trace on the user.
 */

const { test, describe, before, after } = require('node:test');
const assert = require('node:assert');
const { execSync, execFileSync } = require('node:child_process');
const { VIBIUM } = require('../helpers');

function stopDaemon() {
  try { execSync(`${VIBIUM} daemon stop`, { encoding: 'utf-8', timeout: 10000 }); } catch (e) {}
}

describe('CLI wrapper: forwards binary exit status without Node stack trace', () => {
  before(() => {
    stopDaemon();
    execSync(`${VIBIUM} daemon start --headless`, { encoding: 'utf-8' });
    execSync(`${VIBIUM} go https://example.com`, { encoding: 'utf-8' });
  });

  after(() => {
    stopDaemon();
  });

  test('failing eval exits non-zero without Node trace', () => {
    let err = null;
    let stderr = '';
    try {
      execFileSync(VIBIUM, ['eval', 'throw new Error("boom")'], {
        encoding: 'utf-8',
        stdio: ['pipe', 'pipe', 'pipe'],
      });
    } catch (e) {
      err = e;
      stderr = (e.stderr || '').toString();
    }

    assert.ok(err, 'eval that throws should cause non-zero exit');
    assert.ok(typeof err.status === 'number' && err.status !== 0, `expected non-zero status, got ${err.status}`);
    assert.ok(!stderr.includes('node:child_process'), `stderr should not contain a Node trace, got: ${stderr.slice(0, 400)}`);
    assert.ok(!/Node\.js v\d+/.test(stderr), `stderr should not contain a Node version footer, got: ${stderr.slice(0, 400)}`);
  });

  test('failing eval --stdin exits non-zero without Node trace', () => {
    let err = null;
    let stderr = '';
    try {
      execFileSync(VIBIUM, ['eval', '--stdin'], {
        input: 'throw new Error("boom")',
        encoding: 'utf-8',
        stdio: ['pipe', 'pipe', 'pipe'],
      });
    } catch (e) {
      err = e;
      stderr = (e.stderr || '').toString();
    }

    assert.ok(err, 'failing stdin eval should cause non-zero exit');
    assert.ok(typeof err.status === 'number' && err.status !== 0, `expected non-zero status, got ${err.status}`);
    assert.ok(!stderr.includes('node:child_process'), `stderr should not contain a Node trace, got: ${stderr.slice(0, 400)}`);
    assert.ok(!/Node\.js v\d+/.test(stderr), `stderr should not contain a Node version footer, got: ${stderr.slice(0, 400)}`);
  });

  test('successful eval exits 0', () => {
    const out = execFileSync(VIBIUM, ['eval', 'document.title'], {
      encoding: 'utf-8',
      stdio: ['pipe', 'pipe', 'pipe'],
    });
    assert.ok(out.includes('Example Domain'), `expected Example Domain in output, got: ${out}`);
  });
});
