#!/usr/bin/env node
// Run a command with a timeout. Kills the command if it exceeds the limit.
// Usage: node scripts/timeout.mjs SECONDS command [args...]
//
// On POSIX the child runs in its own process group. On timeout: dump the
// process table (shows what was stuck), SIGTERM the group, SIGKILL after
// 10s, exit 124.
import { spawn, spawnSync } from 'node:child_process';

const [seconds, ...cmdArgs] = process.argv.slice(2);
const limit = Number(seconds) * 1000;

const isWin = process.platform === 'win32';
const child = isWin
  ? spawn(cmdArgs.join(' '), { stdio: 'inherit', shell: true })
  : spawn(cmdArgs[0], cmdArgs.slice(1), { stdio: 'inherit', detached: true });

function killGroup(signal) {
  try {
    if (isWin) child.kill();
    else process.kill(-child.pid, signal);
  } catch {}
}

// The child is in its own group, so forward terminal signals explicitly.
process.on('SIGINT', () => killGroup('SIGINT'));
process.on('SIGTERM', () => killGroup('SIGTERM'));

let timedOut = false;
const timer = setTimeout(() => {
  timedOut = true;
  process.stderr.write(`\n--- TIMEOUT after ${seconds}s; process table before kill: ---\n`);
  if (!isWin) {
    const ps = spawnSync('ps', ['axo', 'pid,ppid,pgid,stat,etime,args'], { encoding: 'utf-8' });
    if (ps.stdout) process.stderr.write(ps.stdout);
  }
  process.stderr.write(`--- killing process group ---\n`);
  killGroup('SIGTERM');
  setTimeout(() => killGroup('SIGKILL'), 10_000).unref();
  // In case the child is unkillable.
  setTimeout(() => process.exit(124), 15_000).unref();
}, limit);

child.on('exit', (code) => {
  clearTimeout(timer);
  process.exit(timedOut ? 124 : (code ?? 1));
});
