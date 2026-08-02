/**
 * CLI Tests: Input Tools
 * Tests hover command in oneshot mode
 * Note: scroll, keys, select require daemon mode and are tested via MCP
 */

const { test, describe, before, after } = require('node:test');
const assert = require('node:assert');
const { execSync, spawn } = require('node:child_process');
const path = require('path');
const { VIBIUM } = require('../helpers');

let serverProcess, baseURL;

before(async () => {
  serverProcess = spawn('node', [path.join(__dirname, '../helpers/test-server.js')], {
    stdio: ['pipe', 'pipe', 'pipe'],
  });
  baseURL = await new Promise((resolve) => {
    serverProcess.stdout.once('data', (data) => {
      resolve(data.toString().trim());
    });
  });
});

after(() => {
  if (serverProcess) serverProcess.kill();
});

describe('CLI: Input Tools', () => {
  test('hover command hovers over element', () => {
    const result = execSync(`${VIBIUM} hover ${baseURL}/example "a"`, {
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
    assert.match(result, /vibium page new/, 'Should list new page');
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

describe('CLI: Negative value flag parsing', () => {
  test('sleep rejects negative value with meaningful error, not flag parse error', () => {
    try {
      execSync(`${VIBIUM} sleep -1`, { encoding: 'utf-8', timeout: 5000, stdio: 'pipe' });
      assert.fail('Should have thrown');
    } catch (err) {
      const output = err.stderr + err.stdout;
      assert.doesNotMatch(output, /unknown shorthand flag/, 'Should not treat -1 as a flag');
      assert.match(output, /positive|invalid/, 'Should give a meaningful error');
    }
  });

  test('sleep rejects a value over the cap instead of silently clamping', () => {
    try {
      execSync(`${VIBIUM} sleep 999999`, { encoding: 'utf-8', timeout: 60000, stdio: 'pipe' });
      assert.fail('Should have thrown');
    } catch (err) {
      const output = err.stderr + err.stdout;
      assert.match(output, /30000 or less/, 'Should say what the limit is');
      assert.doesNotMatch(output, /Slept for/, 'Should not report a shorter sleep as success');
    }
  });

  test('geolocation accepts negative coordinates', () => {
    const result = execSync(`${VIBIUM} geolocation 37.7749 -122.4194`, {
      encoding: 'utf-8',
      timeout: 30000,
    });
    assert.match(result, /Geolocation set/, 'Should set geolocation with negative longitude');
  });

  test('fill accepts negative numeric value', () => {
    // Hermetic fixture — avoids a third-party site so the test only exercises
    // flag parsing, not network reachability.
    execSync(`${VIBIUM} content '<input id="username" value="">'`, {
      encoding: 'utf-8',
      timeout: 30000,
    });
    const result = execSync(`${VIBIUM} fill "#username" "-2"`, {
      encoding: 'utf-8',
      timeout: 30000,
    });
    assert.doesNotMatch(result, /unknown shorthand flag/, 'Should not treat -2 as a flag');
    assert.match(result, /Filled/, 'fill should succeed');
    const value = execSync(`${VIBIUM} eval 'document.getElementById("username").value'`, {
      encoding: 'utf-8',
      timeout: 30000,
    });
    assert.match(value, /-2/, 'field value should actually be set to -2');
  });

  test('type accepts negative numeric value', () => {
    execSync(`${VIBIUM} content '<input id="username" value="">'`, {
      encoding: 'utf-8',
      timeout: 30000,
    });
    const result = execSync(`${VIBIUM} type "#username" "-2"`, {
      encoding: 'utf-8',
      timeout: 30000,
    });
    assert.doesNotMatch(result, /unknown shorthand flag/, 'Should not treat -2 as a flag');
    assert.match(result, /Typed/, 'type should succeed');
    const value = execSync(`${VIBIUM} eval 'document.getElementById("username").value'`, {
      encoding: 'utf-8',
      timeout: 30000,
    });
    assert.match(value, /-2/, 'field value should actually be set to -2');
  });
});

describe('CLI: fill edge cases', () => {
  test('fill "" clears an existing value (regression: #187)', () => {
    execSync(`${VIBIUM} content '<input id="u" value="hello">'`, {
      encoding: 'utf-8',
      timeout: 30000,
    });
    const result = execSync(`${VIBIUM} fill "#u" ""`, {
      encoding: 'utf-8',
      timeout: 30000,
    });
    assert.match(result, /Filled/, 'fill "" should succeed, not error with "value is required"');
    const value = execSync(`${VIBIUM} eval 'document.getElementById("u").value'`, {
      encoding: 'utf-8',
      timeout: 30000,
    });
    assert.strictEqual(value.trim(), '', 'field should be cleared');
  });
});
