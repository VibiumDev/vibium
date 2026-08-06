#!/usr/bin/env node
// Cloud browser benchmark harness for vibium.
//
// Times the full lifecycle against each configured provider:
//   mint (vendor REST call, if any) → connect (vibium start, includes the
//   classic POST /session for grids) → first navigation → title read →
//   screenshot → close.
//
// Providers activate when their env vars are set (see providers.js and
// .env.example). Run everything configured:
//
//   node scripts/cloud-bench/bench.js
//
// Options:
//   --provider <name>   run one provider (repeatable)
//   --runs <n>          iterations per provider (default 3)
//   --mint-only         only time vendor session mint/delete (works for
//                       CDP-only providers that vibium can't drive yet)
//   --url <url>         page to navigate to (default https://example.com).
//                       Must be reachable FROM THE BROWSER'S location —
//                       cloud vendors' browsers can't see your localhost,
//                       so keep it public (a neutral constant also keeps
//                       the nav column comparable across vendors). Vendor
//                       tunnel products (BrowserStack Local, Sauce Connect)
//                       solve private-app reachability for real testing,
//                       but don't bench through them: you'd be measuring
//                       the tunnel, not the vendor.
//
// Results print as a table and append to scripts/cloud-bench/results/*.jsonl

const path = require('path');
const fs = require('fs');

const repoRoot = path.resolve(__dirname, '..', '..');
const { browser } = require(path.join(repoRoot, 'clients', 'javascript', 'dist'));
const { PROVIDERS } = require('./providers');

const args = process.argv.slice(2);
function flag(name) { return args.includes(`--${name}`); }
function opt(name, dflt) {
  const i = args.indexOf(`--${name}`);
  return i >= 0 && args[i + 1] ? args[i + 1] : dflt;
}
function optAll(name) {
  const out = [];
  args.forEach((a, i) => { if (a === `--${name}` && args[i + 1]) out.push(args[i + 1]); });
  return out;
}

const RUNS = parseInt(opt('runs', '3'), 10);
const NAV_URL = opt('url', 'https://example.com');
const MINT_ONLY = flag('mint-only');
const ONLY = optAll('provider');

const now = () => performance.now();
const ms = (v) => (v == null ? '—' : `${Math.round(v)}ms`);

function percentile(sorted, p) {
  if (!sorted.length) return null;
  return sorted[Math.min(sorted.length - 1, Math.floor((p / 100) * sorted.length))];
}

async function benchOnce(provider) {
  const t = {};
  let minted = null;

  const t0 = now();
  if (provider.mint) {
    minted = await provider.mint();
    t.mint = now() - t0;
  }

  const url = minted?.url ?? provider.url;
  const headers = minted?.headers ?? provider.headers;
  const caps = provider.caps;

  if (MINT_ONLY || provider.mintOnly) {
    if (minted?.cleanup) await minted.cleanup();
    t.note = provider.mintOnly ? 'mint-only (CDP-only provider — vibium needs the BiDi shim to drive it)' : 'mint-only';
    return t;
  }

  let bro;
  const t1 = now();
  try {
    bro = await browser.start(url, { headers, caps, headless: true });
    t.connect = now() - t1;

    const page = await bro.page();
    const t2 = now();
    await page.go(NAV_URL);
    t.nav = now() - t2;

    const t3 = now();
    await page.title();
    t.title = now() - t3;

    const t4 = now();
    await page.screenshot();
    t.screenshot = now() - t4;
  } finally {
    const t5 = now();
    if (bro) await bro.stop().catch(() => {});
    if (minted?.cleanup) await minted.cleanup().catch(() => {});
    t.close = now() - t5;
  }
  t.total = now() - t0;
  return t;
}

async function benchProvider(provider) {
  const label = provider.resultName || provider.name;
  const runs = [];
  for (let i = 0; i < RUNS; i++) {
    process.stderr.write(`  ${label} run ${i + 1}/${RUNS}...`);
    try {
      const t = await benchOnce(provider);
      runs.push(t);
      process.stderr.write(` ok (${ms(t.total ?? t.mint)})\n`);
    } catch (err) {
      runs.push({ error: String(err && err.message || err) });
      process.stderr.write(` FAILED: ${err && err.message || err}\n`);
    }
  }
  return runs;
}

function summarize(name, runs) {
  const ok = runs.filter((r) => !r.error);
  const row = { provider: name, ok: `${ok.length}/${runs.length}` };
  for (const key of ['mint', 'connect', 'nav', 'title', 'screenshot', 'close', 'total']) {
    const vals = ok.map((r) => r[key]).filter((v) => v != null).sort((a, b) => a - b);
    row[key] = vals.length ? ms(percentile(vals, 50)) : '—';
  }
  return row;
}

async function main() {
  const active = PROVIDERS.filter((p) =>
    ONLY.length ? ONLY.includes(p.name) : p.available());
  const skipped = PROVIDERS.filter((p) => !active.includes(p));

  if (!active.length) {
    console.error('No providers configured. Set env vars (see .env.example) or use --provider local');
    process.exit(1);
  }
  for (const p of skipped) {
    console.error(`skip ${p.name}: ${p.missing()}`);
  }

  const results = {};
  for (const p of active) {
    const label = p.resultName || p.name;
    console.error(`\n=== ${label} ===`);
    results[label] = await benchProvider(p);
  }

  // Persist raw runs
  const resultsDir = path.join(__dirname, 'results');
  fs.mkdirSync(resultsDir, { recursive: true });
  const stamp = new Date().toISOString().replace(/[:.]/g, '-');
  const outFile = path.join(resultsDir, `bench-${stamp}.jsonl`);
  for (const [name, runs] of Object.entries(results)) {
    for (const r of runs) {
      fs.appendFileSync(outFile, JSON.stringify({ provider: name, url: NAV_URL, ts: stamp, ...r }) + '\n');
    }
  }

  console.log(`\nMedian times over ${RUNS} run(s), navigating to ${NAV_URL}:`);
  console.table(Object.entries(results).map(([name, runs]) => summarize(name, runs)));
  console.log(`Raw results: ${path.relative(process.cwd(), outFile)}`);
}

main().catch((err) => { console.error(err); process.exit(1); });
