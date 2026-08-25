/**
 * Packaging Tests: the published npm tarball must contain the built dist/.
 * dist/ is generated and git-ignored; publishing without it shipped an empty
 * package and broke require('vibium')/'vibium/sync' and the TypeScript types
 * (#103, #127, #100). `npm pack` runs the prepack hook which rebuilds dist.
 */

const { test, describe } = require('node:test');
const assert = require('node:assert');
const { execFileSync } = require('node:child_process');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');

const PKG_DIR = path.join(__dirname, '../../packages/vibium');
// On Windows `npm` is `npm.cmd`; execFileSync has no shell to resolve the bare name.
const NPM = process.platform === 'win32' ? 'npm.cmd' : 'npm';

describe('Packaging: npm tarball contents', () => {
  test('npm pack includes the built dist files (#103/#127/#100)', () => {
    const dest = fs.mkdtempSync(path.join(os.tmpdir(), 'vibium-pack-'));
    // `npm pack --json` writes machine-readable output to stdout; the prepack
    // build logs go to stderr (see scripts/prepack.mjs).
    // On Windows we must spawn via the shell (Node blocks .cmd otherwise), but
    // the shell doesn't auto-quote args — so quote the destination ourselves in
    // case the temp path contains a space (e.g. a username with a space). On
    // POSIX there's no shell, so the path is passed literally.
    const useShell = process.platform === 'win32';
    const destArg = useShell ? `"${dest}"` : dest;
    const out = execFileSync(NPM, ['pack', '--json', '--pack-destination', destArg], {
      cwd: PKG_DIR,
      encoding: 'utf-8',
      stdio: ['ignore', 'pipe', 'inherit'],
      shell: useShell,
    });

    try {
      const files = JSON.parse(out)[0].files.map((f) => f.path);
      const required = [
        'dist/index.js', 'dist/index.mjs', 'dist/index.d.ts',
        'dist/sync.js', 'dist/sync.mjs', 'dist/sync.d.ts',
        'dist/worker.js',
      ];
      for (const f of required) {
        assert.ok(files.includes(f), `published tarball must include ${f}; got: ${files.join(', ')}`);
      }
    } finally {
      fs.rmSync(dest, { recursive: true, force: true });
    }
  });
});

describe('Packaging: postinstall shim replacement (#356)', () => {
  const { linkBinaryOverShim } = require('../../packages/vibium/scripts/link-binary');
  const { spawnSync } = require('node:child_process');

  function makeFixture() {
    const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'vibium-link-'));
    const shim = path.join(dir, 'cli.js');
    const binary = path.join(dir, 'vibium');
    fs.writeFileSync(shim, '#!/usr/bin/env node\nconsole.log("shim");\n');
    // Stands in for the platform binary: prints a marker and exits 0
    fs.writeFileSync(binary, '#!/bin/sh\necho REAL_BINARY\n');
    fs.chmodSync(binary, 0o755);
    return { dir, shim, binary };
  }

  test('replaces the shim with the binary and the bin path stays runnable', (t) => {
    if (process.platform === 'win32') {
      t.skip('replacement is POSIX-only');
      return;
    }
    const { dir, shim, binary } = makeFixture();
    try {
      assert.strictEqual(linkBinaryOverShim(shim, binary), true);
      assert.strictEqual(fs.readFileSync(shim, 'utf-8'), fs.readFileSync(binary, 'utf-8'),
        'shim path must now hold the binary bytes');
      const run = spawnSync(shim, [], { encoding: 'utf-8' });
      assert.strictEqual(run.status, 0);
      assert.match(run.stdout, /REAL_BINARY/, 'running the bin path must run the binary');
      assert.ok(!fs.existsSync(shim + '.link-tmp'), 'no temp file left behind');
    } finally {
      fs.rmSync(dir, { recursive: true, force: true });
    }
  });

  test('missing binary leaves the shim untouched', () => {
    const { dir, shim } = makeFixture();
    try {
      const before = fs.readFileSync(shim, 'utf-8');
      assert.strictEqual(linkBinaryOverShim(shim, path.join(dir, 'nope')), false);
      assert.strictEqual(linkBinaryOverShim(shim, null), false);
      assert.strictEqual(fs.readFileSync(shim, 'utf-8'), before);
    } finally {
      fs.rmSync(dir, { recursive: true, force: true });
    }
  });

  test('prefers a hard link so no extra copy is stored', (t) => {
    if (process.platform === 'win32') {
      t.skip('replacement is POSIX-only');
      return;
    }
    const { dir, shim, binary } = makeFixture();
    try {
      linkBinaryOverShim(shim, binary);
      assert.strictEqual(fs.statSync(shim).ino, fs.statSync(binary).ino,
        'same filesystem should get a hard link, not a copy');
    } finally {
      fs.rmSync(dir, { recursive: true, force: true });
    }
  });
});

describe('Packaging: shim replacement guard (#356)', () => {
  const { shouldReplaceShim } = require('../../packages/vibium/scripts/link-binary');

  test('replaces only installed copies on POSIX without PnP', () => {
    assert.strictEqual(shouldReplaceShim('darwin', undefined, '/app/node_modules/vibium'), true);
    assert.strictEqual(shouldReplaceShim('linux', undefined, '/x/node_modules/vibium'), true);
  });

  test('never replaces on Windows or under Yarn PnP', () => {
    assert.strictEqual(shouldReplaceShim('win32', undefined, '/app/node_modules/vibium'), false);
    assert.strictEqual(shouldReplaceShim('linux', '3.0.0', '/app/node_modules/vibium'), false);
  });

  test('never replaces a source checkout, whose shim is tracked', () => {
    // The repo links packages/vibium as a file dependency; npm runs the
    // symlinked dep's scripts in its real directory, outside node_modules.
    assert.strictEqual(shouldReplaceShim('darwin', undefined, '/Users/dev/vibium/packages/vibium'), false);
  });
});
