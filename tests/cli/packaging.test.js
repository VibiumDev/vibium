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

describe('Packaging: setup links the binary when no install script ran (#678)', () => {
  const { spawnSync } = require('node:child_process');
  const FAKE_BINARY = '#!/bin/sh\necho "REAL_BINARY $*"\n';

  // An installed copy of the bin scripts beside a stand-in platform package.
  // packageDir is where the vibium package sits under root.
  function makeInstall(packageDir) {
    const root = fs.realpathSync(fs.mkdtempSync(path.join(os.tmpdir(), 'vibium-setup-link-')));
    const pkg = path.join(root, packageDir);
    for (const file of ['package.json', 'bin/cli.js', 'bin/node-version-check.js', 'scripts/link-binary.js']) {
      fs.mkdirSync(path.dirname(path.join(pkg, file)), { recursive: true });
      fs.copyFileSync(path.join(PKG_DIR, file), path.join(pkg, file));
    }
    const arch = os.arch() === 'x64' ? 'x64' : 'arm64';
    const platformPkg = path.join(root, 'node_modules', '@vibium', `${os.platform()}-${arch}`);
    fs.mkdirSync(path.join(platformPkg, 'bin'), { recursive: true });
    fs.writeFileSync(path.join(platformPkg, 'package.json'), '{"name":"stand-in"}');
    fs.writeFileSync(path.join(platformPkg, 'bin', 'vibium'), FAKE_BINARY);
    fs.chmodSync(path.join(platformPkg, 'bin', 'vibium'), 0o755);
    return { root, shim: path.join(pkg, 'bin', 'cli.js') };
  }

  const run = (shim, ...args) => spawnSync(process.execPath, [shim, ...args], { encoding: 'utf-8' });

  test('setup replaces the shim, and the bin path then runs the binary', (t) => {
    if (process.platform === 'win32') {
      t.skip('replacement is POSIX-only');
      return;
    }
    const { root, shim } = makeInstall('node_modules/vibium');
    try {
      const setup = run(shim, 'setup', '--non-interactive');
      assert.strictEqual(setup.status, 0, setup.stderr);
      assert.match(setup.stdout, /REAL_BINARY setup --non-interactive/, 'setup itself must still run');
      assert.strictEqual(fs.readFileSync(shim, 'utf-8'), FAKE_BINARY, 'shim path must now hold the binary');
      const direct = spawnSync(shim, ['--version'], { encoding: 'utf-8' });
      assert.match(direct.stdout, /REAL_BINARY --version/);
    } finally {
      fs.rmSync(root, { recursive: true, force: true });
    }
  });

  test('other commands leave the shim alone', (t) => {
    if (process.platform === 'win32') {
      t.skip('replacement is POSIX-only');
      return;
    }
    const { root, shim } = makeInstall('node_modules/vibium');
    try {
      const before = fs.readFileSync(shim, 'utf-8');
      for (const args of [['--version'], ['go', 'setup'], ['--json', 'ready']]) {
        const result = run(shim, ...args);
        assert.match(result.stdout, new RegExp(`REAL_BINARY ${args.join(' ')}`));
        assert.strictEqual(fs.readFileSync(shim, 'utf-8'), before, `vibium ${args.join(' ')} must not replace the shim`);
      }
    } finally {
      fs.rmSync(root, { recursive: true, force: true });
    }
  });

  test('setup in a source checkout keeps the tracked shim', (t) => {
    if (process.platform === 'win32') {
      t.skip('replacement is POSIX-only');
      return;
    }
    const { root, shim } = makeInstall('packages/vibium');
    try {
      const before = fs.readFileSync(shim, 'utf-8');
      assert.match(run(shim, 'setup').stdout, /REAL_BINARY setup/);
      assert.strictEqual(fs.readFileSync(shim, 'utf-8'), before);
    } finally {
      fs.rmSync(root, { recursive: true, force: true });
    }
  });

  test('a read-only install keeps the shim and setup still runs', (t) => {
    if (process.platform === 'win32' || process.getuid() === 0) {
      t.skip('needs POSIX file permissions and a user they apply to');
      return;
    }
    const { root, shim } = makeInstall('node_modules/vibium');
    const binDir = path.dirname(shim);
    try {
      const before = fs.readFileSync(shim, 'utf-8');
      fs.chmodSync(binDir, 0o555);
      const setup = run(shim, 'setup');
      assert.strictEqual(setup.status, 0, setup.stderr);
      assert.match(setup.stdout, /REAL_BINARY setup/);
      assert.strictEqual(fs.readFileSync(shim, 'utf-8'), before);
    } finally {
      fs.chmodSync(binDir, 0o755);
      fs.rmSync(root, { recursive: true, force: true });
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

describe('Packaging: postinstall and bin scripts are self-contained', () => {
  // The nightly local-smoke on 2026-08-26 failed on every platform because
  // postinstall.js required scripts/link-binary.js, which the files
  // allowlist did not ship. Every relative require reachable from the
  // packaged entry scripts must itself be in the tarball.
  test('every relative require of packaged scripts ships in the tarball', () => {
    const dest = fs.mkdtempSync(path.join(os.tmpdir(), 'vibium-pack-req-'));
    const useShell = process.platform === 'win32';
    const destArg = useShell ? `"${dest}"` : dest;
    const out = execFileSync(NPM, ['pack', '--json', '--pack-destination', destArg], {
      cwd: PKG_DIR,
      encoding: 'utf-8',
      stdio: ['ignore', 'pipe', 'inherit'],
      shell: useShell,
    });

    try {
      const files = new Set(JSON.parse(out)[0].files.map((f) => f.path));
      // Walk relative requires transitively from the executable entries.
      const queue = ['postinstall.js', 'bin/cli.js'];
      const seen = new Set();

      while (queue.length > 0) {
        const entry = queue.shift();
        if (seen.has(entry)) continue;
        seen.add(entry);

        assert.ok(files.has(entry), `tarball must include ${entry}`);
        if (!entry.endsWith('.js')) continue;
        const source = fs.readFileSync(path.join(PKG_DIR, entry), 'utf-8');
        for (const match of source.matchAll(/require\(['"](\.[^'"]+)['"]\)/g)) {
          let dep = path.posix.join(path.posix.dirname(entry), match[1]);
          if (!dep.endsWith('.js') && !dep.endsWith('.json')) dep += '.js';
          assert.ok(
            files.has(dep),
            `${entry} requires ${match[1]}, but ${dep} is not in the tarball; add it to the files allowlist`
          );
          queue.push(dep);
        }
      }
    } finally {
      fs.rmSync(dest, { recursive: true, force: true });
    }
  });
});
