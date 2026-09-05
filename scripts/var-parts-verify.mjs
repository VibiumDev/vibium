#!/usr/bin/env node
// Reusable CLI-only tutorial smoke test. Load verifier configuration in the
// invoking shell; credentials are inherited, never read from or written to code.
import { execFile } from 'node:child_process';
import { promisify } from 'node:util';
import { randomUUID } from 'node:crypto';
import { resolve } from 'node:path';

const exec = promisify(execFile);
const binary = process.env.VIBIUM_BINARY || 'vibium';
const session = `first-verify-${randomUUID().slice(0, 8)}`;
const recording = resolve(process.env.VIBIUM_VERIFY_RECORD || `${session}.zip`);
const env = { ...process.env, VIBIUM_SESSION: session, VIBIUM_ENGINE: 'chrome', VIBIUM_CONNECT_URL: '', VIBIUM_ENGINE_PATH: '', VIBIUM_ENGINE_CHANNEL: '' };
const claim = 'The current cart contains exactly one Vibium Battery Pack with quantity 1, and the subtotal matches its unit price. Inspect the current cart without changing its contents or proceeding to checkout.';
let recordingStarted = false;

async function cli(...args) {
  const flags = ['--json'];
  if (process.env.VIBIUM_HEADLESS === '1') flags.push('--headless');
  let stdout;
  try {
    ({ stdout } = await exec(binary, [...flags, ...args], { env, timeout: 230000, maxBuffer: 4 * 1024 * 1024 }));
  } catch (error) {
    // Do not dump subprocess options/environment or arbitrary provider output.
    let failure;
    try { failure = JSON.parse(error.stdout); } catch { /* not a CLI envelope */ }
    throw new Error(failure?.error || `Vibium ${args[0]} could not complete`);
  }
  const response = JSON.parse(stdout);
  if (!response.ok) throw new Error(response.error || `Vibium ${args[0]} failed`);
  return response.result;
}

async function findAndClick(...query) {
  const found = await cli('find', ...query);
  const ref = String(found).match(/@e\d+\b/)?.[0];
  if (!ref) throw new Error(`No element reference found for ${query.join(' ')}`);
  await cli('click', ref);
}

try {
  await exec(binary, ['verify', '--help'], { env, timeout: 5000 });
  await cli('go', 'https://var.parts');
  await cli('record', 'start', '-o', recording);
  recordingStarted = true;
  await cli('map');
  await findAndClick('text', 'Vibium Battery Pack');
  await cli('map');
  await findAndClick('role', 'button', '--name', 'Add to Cart');
  await cli('map');
  // The demo's cart icon has no accessible name. Resolve its live link first.
  await findAndClick('a[href="/cart"]');
  await cli('map');
  const result = await cli('verify', claim);
  console.log(JSON.stringify({ ...result, recording }, null, 2));
  // Vibium's exit zero means completed verification. This smoke test instead
  // gates success on the verdict; execution/cleanup errors use exit code 2.
  if (result.status !== 'passed') process.exitCode = 1;
} catch (error) {
  console.error(`Execution error: ${error.message}`);
  process.exitCode = 2;
} finally {
  if (recordingStarted) {
    try { await cli('record', 'stop'); }
    catch (error) { console.error(`Recording error: ${error.message}`); process.exitCode = 2; }
  }
  try { await exec(binary, ['daemon', 'stop'], { env, timeout: 15000 }); }
  catch (error) { console.error(`Session cleanup error: ${error.message}`); process.exitCode = 2; }
}
