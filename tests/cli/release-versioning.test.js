const { test, describe } = require('node:test');
const assert = require('node:assert');
const { execFileSync } = require('node:child_process');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');

const ROOT = path.join(__dirname, '../..');

describe('nightly release metadata', () => {
  test('maps one commit to every registry coordinate', () => {
    const output = execFileSync(process.execPath, [
      'scripts/nightly-metadata.mjs',
      '2863722a580c5637041e1a4a417e27b7e23bd00e',
      '2026-08-20T19:15:42Z',
    ], { cwd: ROOT, encoding: 'utf8' });
    const metadata = JSON.parse(output);
    assert.strictEqual(metadata.npmVersion, '2026.8.20-dev.20260820191542');
    assert.strictEqual(metadata.pythonVersion, '2026.8.20.dev20260820191542');
    assert.strictEqual(metadata.mavenVersion, '2026.8-SNAPSHOT');
    assert.strictEqual(metadata.tag, 'nightly-2026.8.20-dev.20260820191542-2863722');
  });

  test('writes an audit manifest when requested', () => {
    const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'vibium-nightly-'));
    const manifest = path.join(dir, 'manifest.json');
    try {
      execFileSync(process.execPath, [
        'scripts/nightly-metadata.mjs',
        '2863722a580c5637041e1a4a417e27b7e23bd00e',
        '2026-08-20T19:15:42Z',
        manifest,
      ], { cwd: ROOT });
      fs.writeFileSync(path.join(dir, 'artifact.bin'), 'nightly');
      execFileSync(process.execPath, [
        'scripts/finalize-nightly-manifest.mjs', dir, '2026.8-20260820.191542-1',
      ], { cwd: ROOT });
      const result = JSON.parse(fs.readFileSync(manifest));
      assert.strictEqual(result.sha.length, 40);
      assert.strictEqual(result.mavenResolved, '2026.8-20260820.191542-1');
      assert.deepStrictEqual(result.artifacts.map((artifact) => artifact.path), ['artifact.bin']);
      assert.strictEqual(result.artifacts[0].bytes, 7);
    } finally {
      fs.rmSync(dir, { recursive: true, force: true });
    }
  });
});

describe('cross-ecosystem version writer', () => {
  test('writes npm and canonical PyPI versions with exact nightly dependencies', () => {
    const root = fs.mkdtempSync(path.join(os.tmpdir(), 'vibium-version-'));
    const jsonFiles = [
      'package.json', 'clients/javascript/package.json', 'packages/vibium/package.json',
      'packages/linux-x64/package.json', 'packages/linux-arm64/package.json',
      'packages/darwin-x64/package.json', 'packages/darwin-arm64/package.json',
      'packages/win32-x64/package.json',
    ];
    const pythonProjects = [
      'clients/python/pyproject.toml',
      ...['darwin_arm64', 'darwin_x64', 'linux_arm64', 'linux_x64', 'win32_x64']
        .map((name) => `packages/python/vibium_${name}/pyproject.toml`),
    ];
    const initFiles = [
      'clients/python/src/vibium/__init__.py',
      ...['darwin_arm64', 'darwin_x64', 'linux_arm64', 'linux_x64', 'win32_x64']
        .map((name) => `packages/python/vibium_${name}/src/vibium_${name}/__init__.py`),
    ];
    try {
      for (const file of jsonFiles) {
        fs.mkdirSync(path.dirname(path.join(root, file)), { recursive: true });
        const optionalDependencies = file === 'packages/vibium/package.json'
          ? { '@vibium/linux-x64': '1.2.3' }
          : undefined;
        fs.writeFileSync(path.join(root, file), `${JSON.stringify({ version: '1.2.3', optionalDependencies }, null, 2)}\n`);
      }
      for (const file of pythonProjects) {
        fs.mkdirSync(path.dirname(path.join(root, file)), { recursive: true });
        const dependency = file === 'clients/python/pyproject.toml'
          ? '\n"vibium-linux-x64>=1.2.3; sys_platform == \'linux\'"\n'
          : '\n';
        fs.writeFileSync(path.join(root, file), `version = "1.2.3"${dependency}`);
      }
      for (const file of initFiles) {
        fs.mkdirSync(path.dirname(path.join(root, file)), { recursive: true });
        fs.writeFileSync(path.join(root, file), '__version__ = "1.2.3"\n');
      }
      execFileSync(process.execPath, [
        path.join(ROOT, 'scripts/set-version.mjs'), '--root', root,
        '--version', '2026.8.20-dev.20260820191542',
        '--python-version', '2026.8.20.dev20260820191542', '--python-exact',
      ]);
      assert.strictEqual(JSON.parse(fs.readFileSync(path.join(root, 'packages/vibium/package.json'))).version,
        '2026.8.20-dev.20260820191542');
      assert.match(fs.readFileSync(path.join(root, 'clients/python/pyproject.toml'), 'utf8'),
        /vibium-linux-x64==2026\.8\.20\.dev20260820191542;/);
      assert.strictEqual(fs.readFileSync(path.join(root, 'VERSION'), 'utf8'), '2026.8.20-dev.20260820191542\n');
    } finally {
      fs.rmSync(root, { recursive: true, force: true });
    }
  });
});
