#!/usr/bin/env node
// Runs automatically before `npm pack` / `npm publish` (npm "prepack" lifecycle).
//
// The published `vibium` package ships the built JS client in `dist/`, but
// `dist/` is git-ignored and generated. Publishing from a fresh checkout (or
// before `make package-js` had run) used to ship a package WITHOUT `dist/` and
// npm gave no warning — see issues #103, #127, #100 ("Cannot find module
// 'vibium/sync'", missing TypeScript declarations).
//
// This script makes publishing self-contained:
//   1. If the monorepo JS client source is present, (re)build it and copy the
//      fresh output into ./dist — so you can never publish a stale/empty dist.
//   2. Verify every file referenced by package.json "exports" actually exists.
//      If anything is missing, fail loudly and abort the pack/publish.

import { execFileSync } from "node:child_process";
import { cpSync, existsSync, mkdirSync, readdirSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

// On Windows `npm` is `npm.cmd`; execFileSync doesn't go through a shell, so it
// can't resolve the bare name and fails with ENOENT.
const NPM = process.platform === "win32" ? "npm.cmd" : "npm";

const pkgDir = join(dirname(fileURLToPath(import.meta.url)), "..");
const distDir = join(pkgDir, "dist");
// packages/vibium -> packages -> <repo root> -> clients/javascript
const clientDir = join(pkgDir, "..", "..", "clients", "javascript");
const clientDist = join(clientDir, "dist");

// Files the package.json "exports"/"main"/"types" map points at, plus the
// worker bundle that sync.js loads at runtime. Publishing without any of these
// produces a broken package.
const REQUIRED = [
  "index.js",
  "index.mjs",
  "index.d.ts",
  "sync.js",
  "sync.mjs",
  "sync.d.ts",
  "worker.js",
];

// Log progress to stderr: npm reserves stdout for machine-readable output such
// as `npm pack --json`, so anything we print to stdout would corrupt it.
function buildClient() {
  console.error("[prepack] Building JS client in clients/javascript ...");
  // Send the child's stdout to our stderr (fd 2) so build logs don't pollute
  // stdout either.
  execFileSync(NPM, ["run", "build"], {
    cwd: clientDir,
    stdio: ["ignore", 2, 2],
    shell: process.platform === "win32",
  });
}

function copyDist() {
  mkdirSync(distDir, { recursive: true });
  for (const entry of readdirSync(clientDist)) {
    cpSync(join(clientDist, entry), join(distDir, entry), { recursive: true });
  }
  console.error("[prepack] Copied clients/javascript/dist -> packages/vibium/dist");
}

if (existsSync(clientDir)) {
  // Building inside the monorepo: always regenerate so the tarball is fresh.
  buildClient();
  copyDist();
} else {
  console.error("[prepack] Monorepo client source not found; using existing dist/.");
}

const missing = REQUIRED.filter((f) => !existsSync(join(distDir, f)));
if (missing.length > 0) {
  console.error(
    `\n[prepack] ERROR: refusing to pack/publish — missing dist files:\n` +
      missing.map((f) => `  - dist/${f}`).join("\n") +
      `\n\nRun \`make package-js\` from the repo root to build them first.\n`,
  );
  process.exit(1);
}

console.error(`[prepack] Verified ${REQUIRED.length} required dist files. OK.`);
