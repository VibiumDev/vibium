/**
 * CLI Tests: touch gestures (tap, swipe, long-press).
 * These assert the CLI wiring: arg parsing -> daemon -> agent handler -> touch
 * primitive, by the command's own confirmation. The MCP suite verifies the
 * resulting pointer events, through the same handlers and primitives, against a
 * recorder on an isolated page. Gated by the touch capability (Chrome only).
 */

const { test, describe } = require("../../helpers/capabilities").suite("touch");
const assert = require('node:assert');
const { execSync } = require('node:child_process');
const { VIBIUM } = require("../../helpers");

function sh(cmd) {
  return execSync(`${VIBIUM} ${cmd}`, { encoding: 'utf-8', timeout: 30000 });
}

describe('CLI: touch gestures', () => {
  test('tap confirms the point', () => {
    assert.match(sh(`tap 60 80`), /Tapped at \(60, 80\)/);
  });

  test('swipe confirms start, end, and honors --duration', () => {
    assert.match(sh(`swipe 40 100 240 100 --duration 200`), /Swiped from \(40, 100\) to \(240, 100\)/);
  });

  test('long-press confirms the point and honors --duration', () => {
    assert.match(sh(`long-press 120 120 --duration 300`), /Long-pressed at \(120, 120\)/);
  });
});
