import fs from 'node:fs';

const file = process.argv[2];
if (!file) {
  console.error('usage: report-capability-summary.mjs <ndjson-file>');
  process.exit(1);
}

let text;
try {
  text = fs.readFileSync(file, 'utf8');
} catch {
  // No adapter-using test file ran, so there is nothing to report.
  process.exit(0);
}

const totals = { collected: 0, selected: 0, skipped: 0 };
const capabilities = new Map();
const engines = new Set();
for (const line of text.split('\n')) {
  if (!line.trim()) continue;
  const entry = JSON.parse(line);
  engines.add(entry.engine);
  totals.collected += entry.collected;
  totals.selected += entry.selected;
  totals.skipped += entry.skipped;
  for (const [name, count] of Object.entries(entry.capabilities)) {
    capabilities.set(name, (capabilities.get(name) || 0) + count);
  }
}

console.log(
  `capabilities: engine=${[...engines].join(',')} collected=${totals.collected} ` +
  `selected=${totals.selected} skipped=${totals.skipped}`
);
for (const [name, count] of [...capabilities].sort()) {
  console.log(`capabilities: skip:${name}=${count}`);
}
fs.rmSync(file, { force: true });
