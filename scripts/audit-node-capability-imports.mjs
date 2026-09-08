import fs from 'node:fs';
import path from 'node:path';

const roots = [
  'tests/cli/engine',
  'tests/js/async/engine',
  'tests/js/sync/engine',
];
let failed = false;
for (const root of roots) {
  for (const name of fs.readdirSync(root)) {
    if (!name.endsWith('.test.js')) continue;
    const file = path.join(root, name);
    const source = fs.readFileSync(file, 'utf8');
    if (/require\(['"]node:test['"]\)/.test(source)) {
      console.error(`${file}: direct node:test import bypasses capability enforcement`);
      failed = true;
    }
    if (!/helpers\/capabilities['"]\)\.suite\(/.test(source)) {
      console.error(`${file}: missing cross-engine capability suite declaration`);
      failed = true;
    }
  }
}
if (failed) process.exitCode = 1;
