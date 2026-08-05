/**
 * CLI Tests: Storage export/restore
 * Verifies `vibium storage` / `vibium storage restore` round-trip
 * localStorage and sessionStorage correctly (#217).
 */

const { test, describe, before, after } = require('node:test');
const assert = require('node:assert');
const { execSync, spawn } = require('node:child_process');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const { VIBIUM } = require('../helpers');

let serverProcess, baseURL, tmpDir, statePath, legacyPath;

before(async () => {
  serverProcess = spawn('node', [path.join(__dirname, '../helpers/test-server.js')], {
    stdio: ['pipe', 'pipe', 'pipe'],
  });
  baseURL = await new Promise((resolve) => {
    serverProcess.stdout.once('data', (data) => {
      resolve(data.toString().trim());
    });
  });
  execSync(`${VIBIUM} go ${baseURL}/`, { encoding: 'utf-8', timeout: 30000 });

  tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), 'vibium-storage-'));
  statePath = path.join(tmpDir, 'state.json');
  legacyPath = path.join(tmpDir, 'legacy.json');
});

after(() => {
  if (tmpDir) fs.rmSync(tmpDir, { recursive: true, force: true });
  if (serverProcess) serverProcess.kill();
});

describe('CLI: storage export/restore', () => {
  test('export writes storage as a nested object, not a double-encoded string', () => {
    execSync(`${VIBIUM} eval "localStorage.setItem('user', 'user_vibium')"`, {
      encoding: 'utf-8',
      timeout: 30000,
    });
    execSync(`${VIBIUM} eval "sessionStorage.setItem('session_id', 'session_vibium')"`, {
      encoding: 'utf-8',
      timeout: 30000,
    });

    execSync(`${VIBIUM} storage -o ${statePath}`, { encoding: 'utf-8', timeout: 30000 });
    const state = JSON.parse(fs.readFileSync(statePath, 'utf-8'));

    // The bug: Evaluate returns the JSON.stringify'd page result as a Go string,
    // so storing it directly wrote a quoted blob that restore could not read.
    assert.strictEqual(
      typeof state.storage,
      'object',
      '"storage" must be a nested object, not a double-encoded string'
    );
    assert.strictEqual(state.storage.localStorage.user, 'user_vibium');
    assert.strictEqual(state.storage.sessionStorage.session_id, 'session_vibium');
  });

  test('restore repopulates localStorage and sessionStorage', () => {
    execSync(`${VIBIUM} eval "localStorage.clear(); sessionStorage.clear();"`, {
      encoding: 'utf-8',
      timeout: 30000,
    });

    // Confirm storage really is empty, so a passing restore cannot be a false positive.
    const before = execSync(`${VIBIUM} eval "localStorage.getItem('user')"`, {
      encoding: 'utf-8',
      timeout: 30000,
    }).trim();
    assert.match(before, /^(null|)$/, 'localStorage should be empty before restore');

    const result = execSync(`${VIBIUM} storage restore ${statePath}`, {
      encoding: 'utf-8',
      timeout: 30000,
    });
    assert.match(result, /restored/i, 'should confirm storage was restored');

    const user = execSync(`${VIBIUM} eval "localStorage.getItem('user')"`, {
      encoding: 'utf-8',
      timeout: 30000,
    }).trim();
    const sessionId = execSync(`${VIBIUM} eval "sessionStorage.getItem('session_id')"`, {
      encoding: 'utf-8',
      timeout: 30000,
    }).trim();

    assert.strictEqual(user, 'user_vibium', 'localStorage should be restored');
    assert.strictEqual(sessionId, 'session_vibium', 'sessionStorage should be restored');
  });

  test('restore still reads state files written in the old double-encoded shape', () => {
    // Files saved before the fix hold storage as a JSON string; restore unwraps
    // one level so they keep working.
    fs.writeFileSync(
      legacyPath,
      JSON.stringify({
        cookies: [],
        storage: JSON.stringify({ localStorage: { legacy: 'legacy_value' }, sessionStorage: {} }),
      })
    );

    execSync(`${VIBIUM} eval "localStorage.clear()"`, { encoding: 'utf-8', timeout: 30000 });
    execSync(`${VIBIUM} storage restore ${legacyPath}`, { encoding: 'utf-8', timeout: 30000 });

    const legacy = execSync(`${VIBIUM} eval "localStorage.getItem('legacy')"`, {
      encoding: 'utf-8',
      timeout: 30000,
    }).trim();
    assert.strictEqual(legacy, 'legacy_value', 'legacy state file should still restore');
  });
});
