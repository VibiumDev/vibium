/**
 * CLI Tests: npm wrapper (packages/vibium/bin/cli.js)
 * The wrapper runs the platform binary and must surface ordinary command
 * failures cleanly — the binary's own message and its exit code — without
 * dumping a Node child_process stack trace on top (#161, #111).
 */

const { test, describe } = require('node:test');
const assert = require('node:assert');
const { spawnSync, execSync } = require('node:child_process');
const path = require('node:path');
const { VIBIUM } = require('../helpers');

const CLI_JS = path.join(__dirname, '../../packages/vibium/bin/cli.js');

describe('CLI: npm wrapper error handling', () => {
  test('a failing command exits non-zero with no Node stack trace', () => {
    const result = spawnSync(process.execPath, [CLI_JS, 'definitely-not-a-real-command'], {
      encoding: 'utf-8',
    });

    assert.notStrictEqual(result.status, 0, 'should exit non-zero on a failed command');

    const out = (result.stdout || '') + (result.stderr || '');
    assert.doesNotMatch(
      out,
      /node:child_process|at ChildProcess|at Object\.execFileSync|throw err;/,
      `should not print a Node child_process stack trace, got:\n${out}`,
    );
    // The binary's own error message should still be shown.
    assert.match(out, /unknown command|Error|usage/i, `should show the binary's message, got:\n${out}`);
  });
});

describe('CLI: ws-test scheme hint', () => {
  test('names the ws:// form when given http:// (#196)', () => {
    try {
      execSync(`${VIBIUM} ws-test http://localhost:59999`, {
        encoding: 'utf-8',
        timeout: 30000,
        stdio: 'pipe',
      });
      assert.fail('should reject an http:// URL');
    } catch (err) {
      const out = err.stdout + err.stderr;
      assert.match(out, /ws:\/\/localhost:59999/, `should suggest the ws:// form, got: ${out.slice(0, 200)}`);
    }
  });
});

describe('CLI: shell completion', () => {
  test('zsh script sources without compinit already loaded (#201)', () => {
    const script = execSync(`${VIBIUM} completion zsh`, { encoding: 'utf-8', timeout: 30000 });

    const lines = script.split('\n');
    assert.match(lines[0], /^#compdef/, 'fpath installs need #compdef on line 1');
    assert.match(lines[1], /\$\+functions\[compdef\]/, 'guard must come right after it');

    // Executing it needs zsh, which CI runners do not all have. The structural
    // checks above are the real guard; this confirms the behavior where we can.
    let haveZsh = true;
    try {
      execSync('command -v zsh', { encoding: 'utf-8', stdio: 'pipe' });
    } catch {
      haveZsh = false;
    }
    if (!haveZsh) return;

    // zsh -f skips rc files, so compdef is undefined unless the guard loads it.
    const out = execSync(
      `zsh -f -c 'source <(${VIBIUM} completion zsh) && echo SOURCED'`,
      { encoding: 'utf-8', timeout: 30000 }
    );
    assert.match(out, /SOURCED/);
    assert.doesNotMatch(out, /command not found/);
  });

  test('other shells still generate', () => {
    assert.match(execSync(`${VIBIUM} completion bash`, { encoding: 'utf-8', timeout: 30000 }), /bash completion/);
    assert.ok(execSync(`${VIBIUM} completion fish`, { encoding: 'utf-8', timeout: 30000 }).length > 0);
  });
});
