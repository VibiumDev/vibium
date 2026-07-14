/**
 * CLI Tests: npm wrapper (packages/vibium/bin/cli.js)
 * The wrapper runs the platform binary and must surface ordinary command
 * failures cleanly — the binary's own message and its exit code — without
 * dumping a Node child_process stack trace on top (#161, #111).
 */

const { test, describe } = require('node:test');
const assert = require('node:assert');
const { spawnSync } = require('node:child_process');
const path = require('node:path');

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
