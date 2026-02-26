/**
 * CLI Tests: Navigation and Screenshots
 * Tests the vibium binary directly
 */

const { test, describe } = require('node:test');
const assert = require('node:assert');
const { execSync, spawn } = require('node:child_process');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const { VIBIUM } = require('../helpers');

describe('CLI: Navigation', () => {
  test('navigate command loads page and prints title', () => {
    const result = execSync(`${VIBIUM} go https://example.com`, {
      encoding: 'utf-8',
      timeout: 30000,
    });
    assert.match(result, /example/i, 'Should show example.com content');
  });

  test('screenshot command creates valid PNG', () => {
    const outFile = path.join(os.tmpdir(), `vibium-test-${Date.now()}.png`);
    try {
      execSync(`${VIBIUM} screenshot https://example.com -o ${outFile}`, {
        encoding: 'utf-8',
        timeout: 30000,
      });

      assert.ok(fs.existsSync(outFile), 'Screenshot file should exist');

      const stats = fs.statSync(outFile);
      assert.ok(stats.size > 1000, 'Screenshot should be a reasonable size');

      // Check PNG magic bytes
      const buffer = fs.readFileSync(outFile);
      assert.strictEqual(buffer[0], 0x89, 'Should be valid PNG (byte 0)');
      assert.strictEqual(buffer[1], 0x50, 'Should be valid PNG (byte 1)');
      assert.strictEqual(buffer[2], 0x4E, 'Should be valid PNG (byte 2)');
      assert.strictEqual(buffer[3], 0x47, 'Should be valid PNG (byte 3)');
    } finally {
      if (fs.existsSync(outFile)) {
        fs.unlinkSync(outFile);
      }
    }
  });

  test('eval command executes JavaScript', () => {
    const result = execSync(`${VIBIUM} eval https://example.com "document.title"`, {
      encoding: 'utf-8',
      timeout: 30000,
    });
    assert.match(result, /Example Domain/i, 'Should return page title');
  });
});
