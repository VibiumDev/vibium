/**
 * CLI Tests: Actionability Checks
 * Tests auto-wait and actionability behavior
 */

const { test, describe } = require('node:test');
const assert = require('node:assert');
const { execSync } = require('node:child_process');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const { VIBIUM } = require('../helpers');

describe('CLI: Actionability', () => {
  test('is actionable reports visibility status', () => {
    const result = execSync(`${VIBIUM} is actionable https://example.com "a"`, {
      encoding: 'utf-8',
      timeout: 30000,
    });
    assert.match(result, /Visible.*true/i, 'Link should be visible');
    assert.match(result, /Stable.*true/i, 'Link should be stable');
    assert.match(result, /ReceivesEvents.*true/i, 'Link should receive events');
    assert.match(result, /Enabled.*true/i, 'Link should be enabled');
  });

  test('click with short timeout fails on non-existent element', () => {
    assert.throws(
      () => {
        execSync(`${VIBIUM} click https://example.com "#does-not-exist" --timeout 1s`, {
          encoding: 'utf-8',
          timeout: 10000,
          stdio: 'pipe', // capture the expected failure instead of leaking it to the console
        });
      },
      /timeout|not found/i,
      'Should timeout or report not found'
    );
  });
});

describe('CLI: --timeout flag formats', () => {
  // Write a page where #late only appears ~1s after load, so a click must
  // auto-wait for it. Uses a temp file to avoid shell-quoting a data: URL.
  const tmpFile = path.join(os.tmpdir(), `vibium-timeout-${process.pid}.html`);
  const html =
    '<body><button id="late" style="display:none">Go</button>' +
    '<script>setTimeout(function(){document.getElementById("late").style.display="block"},1000)</script></body>';
  const fileURL = 'file://' + tmpFile;

  test('setup: write delayed-element fixture', () => {
    fs.writeFileSync(tmpFile, html);
  });

  test('accepts duration form (5s) and auto-waits for a late element', () => {
    execSync(`${VIBIUM} go "${fileURL}"`, { encoding: 'utf-8', timeout: 30000 });
    const result = execSync(`${VIBIUM} click "#late" --timeout 5s`, {
      encoding: 'utf-8',
      timeout: 30000,
    });
    assert.match(result, /Clicked/i, '5s timeout should auto-wait then click');
  });

  test('accepts bare-millisecond form (5000) and auto-waits for a late element', () => {
    execSync(`${VIBIUM} go "${fileURL}"`, { encoding: 'utf-8', timeout: 30000 });
    const result = execSync(`${VIBIUM} click "#late" --timeout 5000`, {
      encoding: 'utf-8',
      timeout: 30000,
    });
    assert.match(result, /Clicked/i, '5000ms timeout should auto-wait then click');
  });

  test('bare-millisecond timeout bounds the wait (reported in the error)', () => {
    assert.throws(
      () => {
        execSync(`${VIBIUM} click "#does-not-exist" --timeout 800`, {
          encoding: 'utf-8',
          timeout: 10000,
          stdio: 'pipe', // capture the expected failure instead of leaking it to the console
        });
      },
      /800ms|not found/i,
      'Should fail reporting the 800ms bound'
    );
  });

  test('rejects an invalid timeout value', () => {
    assert.throws(
      () => {
        execSync(`${VIBIUM} click "#x" --timeout 5q`, { encoding: 'utf-8', timeout: 10000, stdio: 'pipe' });
      },
      /invalid timeout/i,
      'Should reject "5q" with a clear message'
    );
  });

  test('newly-flagged action (hover) accepts --timeout', () => {
    execSync(`${VIBIUM} go "${fileURL}"`, { encoding: 'utf-8', timeout: 30000 });
    const result = execSync(`${VIBIUM} hover "#late" --timeout 5s`, {
      encoding: 'utf-8',
      timeout: 30000,
    });
    assert.match(result, /Hovered/i, 'hover should honor --timeout and wait for #late');
  });

  test('wait command accepts duration form (5s)', () => {
    execSync(`${VIBIUM} go "${fileURL}"`, { encoding: 'utf-8', timeout: 30000 });
    const result = execSync(`${VIBIUM} wait "#late" --state visible --timeout 5s`, {
      encoding: 'utf-8',
      timeout: 30000,
    });
    assert.match(result, /visible/i, 'wait should accept 5s and resolve when #late shows');
  });

  test('teardown: remove fixture', () => {
    fs.rmSync(tmpFile, { force: true });
  });
});

describe('CLI: fillable input types', () => {
  test('fill sets an input[type=range] value (regression: #188)', () => {
    execSync(`${VIBIUM} content '<input type="range" id="s" min="0" max="10" value="5">'`, {
      encoding: 'utf-8',
      timeout: 30000,
    });
    const result = execSync(`${VIBIUM} fill "#s" "3"`, {
      encoding: 'utf-8',
      timeout: 30000,
    });
    assert.match(result, /Filled/, 'range should be fillable, not rejected as "not editable"');
    const value = execSync(`${VIBIUM} eval 'document.getElementById("s").value'`, {
      encoding: 'utf-8',
      timeout: 30000,
    });
    assert.strictEqual(value.trim(), '3', 'range value should be set to 3');
  });

  test('fill still rejects a non-fillable input type (checkbox)', () => {
    execSync(`${VIBIUM} content '<input type="checkbox" id="cb">'`, {
      encoding: 'utf-8',
      timeout: 30000,
    });
    assert.throws(
      () => {
        // --timeout before the positionals (fill uses SetInterspersed(false));
        // 1s so the expected failure returns quickly instead of auto-waiting.
        execSync(`${VIBIUM} fill --timeout 1s "#cb" "x"`, {
          encoding: 'utf-8',
          timeout: 10000,
          stdio: 'pipe',
        });
      },
      /not editable/i,
      'checkbox should not be fillable'
    );
  });
});
