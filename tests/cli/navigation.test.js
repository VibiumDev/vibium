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
    const filename = `vibium-test-${Date.now()}.png`;
    let savedPath;
    try {
      const result = execSync(`${VIBIUM} screenshot https://example.com -o ${filename}`, {
        encoding: 'utf-8',
        timeout: 30000,
      });

      // Daemon saves to its screenshot directory — extract path from output
      const match = result.match(/saved to (.+\.png)/i);
      assert.ok(match, 'Should print saved path');
      savedPath = match[1].trim();

      assert.ok(fs.existsSync(savedPath), 'Screenshot file should exist');

      const stats = fs.statSync(savedPath);
      assert.ok(stats.size > 1000, 'Screenshot should be a reasonable size');

      // Check PNG magic bytes
      const buffer = fs.readFileSync(savedPath);
      assert.strictEqual(buffer[0], 0x89, 'Should be valid PNG (byte 0)');
      assert.strictEqual(buffer[1], 0x50, 'Should be valid PNG (byte 1)');
      assert.strictEqual(buffer[2], 0x4E, 'Should be valid PNG (byte 2)');
      assert.strictEqual(buffer[3], 0x47, 'Should be valid PNG (byte 3)');
    } finally {
      if (savedPath && fs.existsSync(savedPath)) {
        fs.unlinkSync(savedPath);
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
  test('eval returns valid JSON for objects', () => {
    const result = execSync(`${VIBIUM} eval https://example.com "({title: document.title, url: location.href})"`, {
      encoding: 'utf-8',
      timeout: 30000,
    });
    const parsed = JSON.parse(result.trim());
    assert.ok(parsed.title, 'Should have title key');
    assert.ok(parsed.url, 'Should have url key');
  });
  test('eval returns valid JSON for arrays', () => {
    const result = execSync(`${VIBIUM} eval https://example.com "[1, 2, 3]"`, {
      encoding: 'utf-8',
      timeout: 30000,
    });
    const parsed = JSON.parse(result.trim());
    assert.deepStrictEqual(parsed, [1, 2, 3], 'Should return array as JSON');
  });
});
