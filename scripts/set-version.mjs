#!/usr/bin/env node

import { readFileSync, writeFileSync } from "node:fs";
import { resolve } from "node:path";

const args = process.argv.slice(2);
const argument = (flag) => {
  const index = args.indexOf(flag);
  if (index === -1 || !args[index + 1]) throw new Error(`missing ${flag}`);
  return args[index + 1];
};
const root = args.includes("--root")
  ? resolve(argument("--root"))
  : resolve(import.meta.dirname, "..");

const npmVersion = argument("--version");
const pythonVersion = args.includes("--python-version")
  ? argument("--python-version")
  : npmVersion;
const exactPythonDependencies = args.includes("--python-exact");

if (!/^\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?$/.test(npmVersion)) {
  throw new Error(`invalid package version: ${npmVersion}`);
}
if (!/^\d+\.\d+\.\d+(?:\.dev\d+)?$/.test(pythonVersion)) {
  throw new Error(`invalid Python version: ${pythonVersion}`);
}

function update(relative, transform) {
  const file = resolve(root, relative);
  const before = readFileSync(file, "utf8");
  const after = transform(before);
  writeFileSync(file, after);
}

for (const relative of [
  "package.json",
  "clients/javascript/package.json",
  "packages/vibium/package.json",
  "packages/linux-x64/package.json",
  "packages/linux-arm64/package.json",
  "packages/darwin-x64/package.json",
  "packages/darwin-arm64/package.json",
  "packages/win32-x64/package.json",
]) {
  update(relative, (source) => {
    let result = source.replace(/("version"\s*:\s*")[^"]+/, `$1${npmVersion}`);
    if (relative === "packages/vibium/package.json") {
      result = result.replace(
        /("@vibium\/[^"]+"\s*:\s*")[^"]+/g,
        `$1${npmVersion}`,
      );
    }
    return result;
  });
}

for (const relative of [
  "clients/python/pyproject.toml",
  "packages/python/vibium_darwin_arm64/pyproject.toml",
  "packages/python/vibium_darwin_x64/pyproject.toml",
  "packages/python/vibium_linux_arm64/pyproject.toml",
  "packages/python/vibium_linux_x64/pyproject.toml",
  "packages/python/vibium_win32_x64/pyproject.toml",
]) {
  update(relative, (source) => {
    let result = source.replace(/^version = "[^"]+"/m, `version = "${pythonVersion}"`);
    if (relative === "clients/python/pyproject.toml") {
      const operator = exactPythonDependencies ? "==" : ">=";
      result = result.replace(
        /(vibium-(?:darwin|linux|win32)-(?:arm64|x64))(?:>=|==)[^;"']+/g,
        `$1${operator}${pythonVersion}`,
      );
    }
    return result;
  });
}

for (const relative of [
  "clients/python/src/vibium/__init__.py",
  "packages/python/vibium_darwin_arm64/src/vibium_darwin_arm64/__init__.py",
  "packages/python/vibium_darwin_x64/src/vibium_darwin_x64/__init__.py",
  "packages/python/vibium_linux_arm64/src/vibium_linux_arm64/__init__.py",
  "packages/python/vibium_linux_x64/src/vibium_linux_x64/__init__.py",
  "packages/python/vibium_win32_x64/src/vibium_win32_x64/__init__.py",
]) {
  update(relative, (source) =>
    source.replace(/^__version__ = "[^"]+"/m, `__version__ = "${pythonVersion}"`),
  );
}

writeFileSync(resolve(root, "VERSION"), `${npmVersion}\n`);
console.log(JSON.stringify({ npmVersion, pythonVersion, exactPythonDependencies }));
