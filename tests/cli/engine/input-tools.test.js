/**
 * CLI Tests: Input Tools
 * Tests hover command in oneshot mode
 * Note: scroll, keys, select require daemon mode and are tested via MCP
 */

const { test, describe, before, after } = require("../../helpers/capabilities").suite("core");
const assert = require('node:assert');
const { execSync, spawn } = require('node:child_process');
const fs = require('node:fs');
const os = require('node:os');
const path = require('path');
const { VIBIUM } = require("../../helpers");

let serverProcess, baseURL;

before(async () => {
  serverProcess = spawn('node', [path.join(__dirname, '../../helpers/test-server.js')], {
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

  test('skill command installs for detected agents', () => {
    // GROK_HOME marks Grok as present; the host's real HOME decides Claude,
    // so only the Grok expectation is deterministic here.
    const grokHome = fs.mkdtempSync(path.join(os.tmpdir(), 'grok-home-'));
    try {
      const result = execSync(`${VIBIUM} add-skill`, {
        encoding: 'utf-8',
        timeout: 5000,
        env: { ...process.env, GROK_HOME: grokHome },
      });
      assert.match(result, /Installed Vibium skill/, 'Should confirm install');
      assert.match(result, /SKILL\.md/, 'Should mention SKILL.md');
      assert.ok(result.includes(grokHome), 'Should install for Grok under GROK_HOME');
    } finally {
      fs.rmSync(grokHome, { recursive: true, force: true });
    }
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

  test('trailing flags work alongside negative positionals (#241)', () => {
    // sleep used DisableFlagParsing and fill/type/geolocation used
    // SetInterspersed(false); both kept negatives working by swallowing any
    // flag that came after a positional.
    const json = execSync(`${VIBIUM} sleep 1 --json`, { encoding: 'utf-8', timeout: 30000 });
    assert.strictEqual(JSON.parse(json.trim()).ok, true, 'sleep should honour a trailing --json');

    execSync(`${VIBIUM} content '<input id="x"><input id="y">'`, { encoding: 'utf-8', timeout: 30000 });

    assert.match(
      execSync(`${VIBIUM} fill "#x" val --timeout 5s`, { encoding: 'utf-8', timeout: 30000 }),
      /Filled/,
      'fill should accept a trailing --timeout'
    );
    assert.match(
      execSync(`${VIBIUM} type "#y" hi --timeout 5s`, { encoding: 'utf-8', timeout: 30000 }),
      /Typed/,
      'type should accept a trailing --timeout'
    );

    const geo = execSync(`${VIBIUM} geolocation 37.8 -122.4 --json`, { encoding: 'utf-8', timeout: 30000 });
    assert.strictEqual(JSON.parse(geo.trim()).ok, true, 'a negative arg and a trailing flag together');
  });

  test('an unknown flag errors instead of being swallowed (#241)', () => {
    assert.throws(
      () => execSync(`${VIBIUM} sleep 1 --bogus`, { encoding: 'utf-8', timeout: 30000, stdio: 'pipe' }),
      /unknown flag/,
      'should report the flag, not treat it as a positional or panic'
    );
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

  test('fill errors when the input type rejects the value (#530)', () => {
    execSync(`${VIBIUM} content '<input id="n" type="number"><input id="d" type="date"><input id="r" type="range" min="0" max="100"><input id="c" type="color">'`, {
      encoding: 'utf-8',
      timeout: 30000,
    });
    const cases = [
      ['#n', 'abc', 'number'],
      ['#d', 'not-a-date', 'date'],
      ['#r', 'abc', 'range'],
      ['#c', 'notacolor', 'color'],
    ];
    for (const [sel, val, type] of cases) {
      try {
        execSync(`${VIBIUM} fill "${sel}" "${val}"`, { encoding: 'utf-8', timeout: 30000, stdio: 'pipe' });
        assert.fail(`fill "${sel}" "${val}" should have errored, the ${type} input discards it`);
      } catch (err) {
        const output = String(err.stderr) + String(err.stdout);
        assert.match(output, new RegExp(`input\\[type=${type}\\] did not accept`), `should name the rejecting type for ${sel}`);
        assert.doesNotMatch(output, /Filled/, `should not report success for ${sel}`);
      }
    }
  });

  test('fill succeeds when the input merely normalizes the value (#530)', () => {
    execSync(`${VIBIUM} content '<input id="r" type="range" min="0" max="100"><input id="c" type="color">'`, {
      encoding: 'utf-8',
      timeout: 30000,
    });
    // Accepted-but-transformed values must not trip the rejection check:
    // range clamps to its bounds, color resolves CSS colors to #rrggbb.
    const cases = [
      ['#r', '-5', '0'],
      ['#c', 'red', '#ff0000'],
      ['#c', '#AABBCC', '#aabbcc'],
    ];
    for (const [sel, val, stored] of cases) {
      const result = execSync(`${VIBIUM} fill "${sel}" "${val}"`, { encoding: 'utf-8', timeout: 30000 });
      assert.match(result, /Filled/, `fill "${sel}" "${val}" should succeed`);
      const value = execSync(`${VIBIUM} eval 'document.querySelector("${sel}").value'`, {
        encoding: 'utf-8',
        timeout: 30000,
      });
      assert.strictEqual(value.trim(), stored, `${sel} should hold the normalized value`);
    }
  });

  test('fill "" still clears a type that cannot hold "" (#530, #187)', () => {
    // clear writes "" through the same script; a color input can never hold
    // "", so the rejection check must exempt empty writes.
    execSync(`${VIBIUM} content '<input id="c" type="color" value="#aabbcc">'`, {
      encoding: 'utf-8',
      timeout: 30000,
    });
    const result = execSync(`${VIBIUM} fill "#c" ""`, { encoding: 'utf-8', timeout: 30000 });
    assert.match(result, /Filled/, 'fill "" should stay exempt from the rejection check');
  });
});
