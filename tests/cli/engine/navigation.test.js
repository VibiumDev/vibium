/**
 * CLI Tests: Navigation and Screenshots
 * Tests the vibium binary directly
 */

const { test, describe, before, after } = require("../../helpers/capabilities").suite("core");
const assert = require('node:assert');
const { execSync, spawn } = require('node:child_process');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
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

describe('CLI: Navigation', () => {
  test('navigate command loads page and prints title', () => {
    const result = execSync(`${VIBIUM} go ${baseURL}/example`, {
      encoding: 'utf-8',
      timeout: 30000,
    });
    assert.match(result, /example/i, 'Should show example page content');
  });

  test('screenshot command creates valid PNG', () => {
    const filename = `vibium-test-${Date.now()}.png`;
    let savedPath;
    try {
      const result = execSync(`${VIBIUM} screenshot ${baseURL}/example -o ${filename}`, {
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

  test('screenshot honors -o paths instead of redirecting to the screenshot dir (#119)', () => {
    // The daemon used to reduce -o to its basename and join it with
    // ~/Pictures/Vibium, so every path the user typed was discarded.
    const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'vibium-shot-'));
    const nested = path.join(dir, 'sub', 'deep.png');
    try {
      const result = execSync(`${VIBIUM} screenshot ${baseURL}/example -o ${nested}`, {
        encoding: 'utf-8',
        timeout: 30000,
      });
      assert.match(result, new RegExp(nested.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')),
        `should report the path it was given, got: ${result}`);
      assert.ok(fs.existsSync(nested), 'screenshot should be at the requested path');
      assert.ok(fs.statSync(nested).size > 1000, 'should be a real PNG');
    } finally {
      fs.rmSync(dir, { recursive: true, force: true });
    }
  });

  test('pdf and record honor -o paths like screenshot does (#119)', () => {
    // pdf and record stop hand the path to the daemon, whose working directory
    // is not the caller's, so a relative path used to land wherever the daemon
    // was started. screenshot was fixed first; these are its siblings.
    const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'vibium-out-'));
    try {
      const pdfPath = path.join(dir, 'nested', 'page.pdf');
      const pdfOut = execSync(`${VIBIUM} pdf ${baseURL}/example -o ${pdfPath}`, {
        encoding: 'utf-8',
        timeout: 30000,
      });
      assert.match(pdfOut, /saved/i, `pdf should report success, got: ${pdfOut}`);
      assert.ok(fs.existsSync(pdfPath), 'pdf should be at the requested path');
      assert.strictEqual(
        fs.readFileSync(pdfPath).subarray(0, 4).toString(),
        '%PDF',
        'should be a real PDF'
      );

      const zipPath = path.join(dir, 'nested', 'trace.zip');
      execSync(`${VIBIUM} record start --name outpath`, { encoding: 'utf-8', timeout: 30000 });
      const recOut = execSync(`${VIBIUM} record stop -o ${zipPath}`, {
        encoding: 'utf-8',
        timeout: 30000,
      });
      assert.match(recOut, /saved/i, `record stop should report success, got: ${recOut}`);
      assert.ok(fs.existsSync(zipPath), 'recording should be at the requested path');
    } finally {
      fs.rmSync(dir, { recursive: true, force: true });
    }
  });

  test('eval command executes JavaScript', () => {
    const result = execSync(`${VIBIUM} eval ${baseURL}/example "document.title"`, {
      encoding: 'utf-8',
      timeout: 30000,
    });
    assert.match(result, /Example Domain/i, 'Should return page title');
  });
  test('eval returns valid JSON for objects', () => {
    const result = execSync(`${VIBIUM} eval ${baseURL}/example "({title: document.title, url: location.href})"`, {
      encoding: 'utf-8',
      timeout: 30000,
    });
    const parsed = JSON.parse(result.trim());
    assert.ok(parsed.title, 'Should have title key');
    assert.ok(parsed.url, 'Should have url key');
  });
  test('eval returns valid JSON for arrays', () => {
    const result = execSync(`${VIBIUM} eval ${baseURL}/example "[1, 2, 3]"`, {
      encoding: 'utf-8',
      timeout: 30000,
    });
    const parsed = JSON.parse(result.trim());
    assert.deepStrictEqual(parsed, [1, 2, 3], 'Should return array as JSON');
  });
});
