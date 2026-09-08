#!/usr/bin/env node

import { writeFileSync } from "node:fs";

const [sha, committedAt, outputFile] = process.argv.slice(2);
if (!/^[0-9a-f]{40}$/.test(sha ?? "")) throw new Error("expected a full commit SHA");
const committed = new Date(committedAt);
if (Number.isNaN(committed.valueOf())) throw new Error("expected an ISO commit timestamp");

const pad = (number) => String(number).padStart(2, "0");
const year = committed.getUTCFullYear();
const month = committed.getUTCMonth() + 1;
const day = committed.getUTCDate();
const stamp = [year, pad(month), pad(day), pad(committed.getUTCHours()),
  pad(committed.getUTCMinutes()), pad(committed.getUTCSeconds())].join("");
const npmVersion = `${year}.${month}.${day}-dev.${stamp}`;
const pythonVersion = `${year}.${month}.${day}.dev${stamp}`;
const mavenVersion = `${year}.${month}-SNAPSHOT`;
const tag = `nightly-${npmVersion}-${sha.slice(0, 7)}`;
const metadata = { sha, committedAt: committed.toISOString(), npmVersion, pythonVersion, mavenVersion, tag };

if (outputFile) writeFileSync(outputFile, `${JSON.stringify(metadata, null, 2)}\n`);
console.log(JSON.stringify(metadata));
if (process.env.GITHUB_OUTPUT && !outputFile) {
  writeFileSync(process.env.GITHUB_OUTPUT,
    `${Object.entries(metadata).map(([key, val]) => `${key}=${val}`).join("\n")}\n`, { flag: "a" });
}
