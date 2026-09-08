#!/usr/bin/env node

import { createHash } from "node:crypto";
import { readdirSync, readFileSync, statSync, writeFileSync } from "node:fs";
import { join, relative, resolve } from "node:path";

const [releaseDirectory, mavenResolved] = process.argv.slice(2);
if (!releaseDirectory) throw new Error("usage: finalize-nightly-manifest.mjs RELEASE_DIR [MAVEN_RESOLVED]");
const root = resolve(releaseDirectory);
const manifestFile = join(root, "manifest.json");
const manifest = JSON.parse(readFileSync(manifestFile, "utf8"));

function files(directory) {
  return readdirSync(directory, { withFileTypes: true }).flatMap((entry) => {
    const path = join(directory, entry.name);
    return entry.isDirectory() ? files(path) : [path];
  });
}

manifest.artifacts = files(root)
  .filter((file) => file !== manifestFile)
  .sort()
  .map((file) => ({
    path: relative(root, file).replaceAll("\\", "/"),
    bytes: statSync(file).size,
    sha256: createHash("sha256").update(readFileSync(file)).digest("hex"),
  }));
if (mavenResolved) manifest.mavenResolved = mavenResolved;
writeFileSync(manifestFile, `${JSON.stringify(manifest, null, 2)}\n`);
