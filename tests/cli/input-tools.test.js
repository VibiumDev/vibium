/**
 * CLI Tests: Input Tools
 * Tests hover command in oneshot mode
 * Note: scroll, keys, select require daemon mode and are tested via MCP
 */

const { test, describe } = require('node:test');
const assert = require('node:assert');
const { execSync } = require('node:child_process');
const { VIBIUM } = require('../helpers');

describe('CLI: Input Tools', () => {
  test('hover command hovers over element', () => {
    const result = execSync(`${VIBIUM} hover https://example.com "a"`, {
      encoding: 'utf-8',
      timeout: 30000,
    });
    assert.match(result, /Hovered/, 'Should confirm hover');
  });

  test('skill --stdout outputs markdown', () => {
    const result = execSync(`${VIBIUM} add-skill --stdout`, {
      encoding: 'utf-8',
      timeout: 5000,
    });
    assert.match(result, /# Vibium Browser Automation/, 'Should have title');
    assert.match(result, /vibium go/, 'Should list go');
    assert.match(result, /vibium click/, 'Should list click');
    assert.match(result, /vibium screenshot/, 'Should list screenshot');
    assert.match(result, /vibium tab-new/, 'Should list new tab');
    assert.match(result, /vibium scroll/, 'Should list scroll');
    assert.match(result, /vibium keys/, 'Should list keys');
  });

  test('skill command installs to ~/.claude/skills/', () => {
    const result = execSync(`${VIBIUM} add-skill`, {
      encoding: 'utf-8',
      timeout: 5000,
    });
    assert.match(result, /Installed Vibium skill/, 'Should confirm install');
    assert.match(result, /SKILL\.md/, 'Should mention SKILL.md');
  });
});
