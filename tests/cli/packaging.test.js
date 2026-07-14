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
