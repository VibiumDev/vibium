import { spawn, execFileSync, ChildProcess } from 'child_process';
import { getVibiumBinPath } from './binary';
import { TimeoutError, BrowserCrashedError } from '../utils/errors';

/** How long to wait for the vibium ready signal on a normal launch. */
const READY_TIMEOUT_MS = 60_000;
/**
 * Ready budget once vibium reports it is downloading the browser (first
 * run). Matches the 5-minute install budget the old client-side installer
 * had.
 */
const INSTALL_READY_TIMEOUT_MS = 300_000;
/**
 * Printed by `vibium pipe` on stderr right before it downloads the browser.
 * Must match the installingMarker constant in the binary's pipe.go.
 */
const INSTALLING_MARKER = '[pipe] installing browser';
/** Bytes of trailing stderr kept for error messages. */
const STDERR_TAIL_LIMIT = 8192;

export interface VibiumProcessOptions {
  engine?: 'chrome' | 'firefox';
  channel?: string;
  headless?: boolean;
  executablePath?: string;
  connectURL?: string;
  connectHeaders?: Record<string, string>;
}

export class VibiumProcess {
  private _process: ChildProcess;
  private _stopped: boolean = false;
  private _preReadyLines: string[] = [];

  private constructor(process: ChildProcess, preReadyLines: string[]) {
    this._process = process;
    this._preReadyLines = preReadyLines;
  }

  /** The child process stdin stream (for sending commands). */
  get stdin() { return this._process.stdin!; }

  /** The child process stdout stream (for receiving responses/events). */
  get stdout() { return this._process.stdout!; }

  /** Lines received before the vibium:lifecycle.ready signal (buffered events). */
  get preReadyLines(): string[] { return this._preReadyLines; }

  static async start(options: VibiumProcessOptions = {}): Promise<VibiumProcess> {
    const binaryPath = options.executablePath || getVibiumBinPath();

    const args = ['pipe'];
    if (options.engine) {
      args.push('--engine', options.engine);
    }
    if (options.channel) {
      args.push('--channel', options.channel);
    }
    if (options.headless === true) {
      args.push('--headless');
    }
    if (options.connectURL) {
      args.push('--connect', options.connectURL);
    }
    if (options.connectHeaders) {
      for (const [key, value] of Object.entries(options.connectHeaders)) {
        args.push('--connect-header', `${key}: ${value}`);
      }
    }

    // Startup is slow (~16s cold) and gets slower when many browsers launch at
    // once (test suites, CI), where a cold launch can blow the ready timeout or
    // crash under resource pressure. A single unlucky launch shouldn't fail
    // hard, so retry a timed-out or crashed launch a couple of times with a
    // short backoff. Failures that won't change on retry (e.g. a missing
    // binary) are surfaced immediately.
    const maxAttempts = 2;
    let lastError: unknown;
    for (let attempt = 1; attempt <= maxAttempts; attempt++) {
      const proc = spawn(binaryPath, args, {
        stdio: ['pipe', 'pipe', 'pipe'],
      });

      // Always drain stderr: an unread pipe blocks vibium once the OS buffer
      // fills. Keep a bounded tail for error messages, and watch for the
      // install marker so the ready deadline can be extended while vibium
      // downloads the browser. Forward diagnostics to our stderr when
      // VIBIUM_STDERR is set.
      let stderrTail = '';
      let onInstallMarker: (() => void) | null = null;
      proc.stderr?.on('data', (chunk: Buffer) => {
        if (process.env.VIBIUM_STDERR) process.stderr.write(chunk);
        stderrTail = (stderrTail + chunk.toString()).slice(-STDERR_TAIL_LIMIT);
        if (onInstallMarker && stderrTail.includes(INSTALLING_MARKER)) {
          const fire = onInstallMarker;
          onInstallMarker = null;
          fire();
        }
      });

      // If the ready-wait throws — timeout, the caller's await being interrupted
      // by a test-runner timeout, an unhandled rejection — we must SIGKILL the
      // spawned child. Otherwise its pipes stay open, keep Node's event loop
      // alive, and the test process hangs with "Promise resolution is still
      // pending but the event loop has already resolved".
      const killSpawnedChild = () => {
        try {
          if (proc.exitCode === null) proc.kill('SIGKILL');
        } catch {}
      };

      // Read lines from stdout until we get the vibium:lifecycle.ready signal.
      // Events (e.g. browsingContext.contextCreated) may arrive first.
      const preReadyLines: string[] = [];
      try {
        await new Promise<void>((resolve, reject) => {
          let buffer = '';
          let resolved = false;

          let timeoutMs = READY_TIMEOUT_MS;
          const onReadyTimeout = () => {
            if (!resolved) {
              resolved = true;
              reject(new TimeoutError('vibium', timeoutMs, 'waiting for vibium ready signal'));
            }
          };
          let timeout = setTimeout(onReadyTimeout, timeoutMs);

          // First run: vibium is downloading the browser, which can
          // legitimately take minutes. Extend the deadline once — this is not
          // a per-read reset, so a hang still fails within the budget.
          onInstallMarker = () => {
            if (resolved) return;
            clearTimeout(timeout);
            timeoutMs = INSTALL_READY_TIMEOUT_MS;
            timeout = setTimeout(onReadyTimeout, timeoutMs);
          };

          const handleData = (data: Buffer) => {
            buffer += data.toString();
            let newlineIdx: number;
            while ((newlineIdx = buffer.indexOf('\n')) !== -1) {
              const line = buffer.slice(0, newlineIdx).trim();
              buffer = buffer.slice(newlineIdx + 1);
              if (!line) continue;

              try {
                const msg = JSON.parse(line);
                if (msg.method === 'vibium:lifecycle.ready') {
                  if (!resolved) {
                    resolved = true;
                    clearTimeout(timeout);
                    // Stop listening for data — the BiDiClient will take over
                    proc.stdout?.removeListener('data', handleData);
                    resolve();
                  }
                  return;
                }
              } catch {
                // Not JSON, ignore
              }
              // Buffer pre-ready lines for replay
              preReadyLines.push(line);
            }
          };

          proc.stdout?.on('data', handleData);

          proc.on('error', (err) => {
            if (!resolved) {
              resolved = true;
              clearTimeout(timeout);
              reject(err);
            }
          });

          proc.on('exit', (code) => {
            if (!resolved) {
              resolved = true;
              clearTimeout(timeout);
              // Include the stderr tail: vibium reports startup and install
              // failures there, and exit-before-ready is exactly that case.
              const detail = [buffer.trim(), stderrTail.trim()].filter(Boolean).join('\n');
              reject(new BrowserCrashedError(code ?? 1, detail));
            }
          });
        });
      } catch (err) {
        killSpawnedChild();
        lastError = err;
        // Only startup timeouts and crashes are worth retrying; a spawn error
        // (e.g. ENOENT) would fail identically, so surface it immediately.
        const retryable = err instanceof TimeoutError || err instanceof BrowserCrashedError;
        if (retryable && attempt < maxAttempts) {
          await new Promise((r) => setTimeout(r, 500));
          continue;
        }
        throw err;
      }

      const vp = new VibiumProcess(proc, preReadyLines);

      // Clean up child process when Node exits unexpectedly
      const cleanup = () => vp.stop();
      process.on('exit', cleanup);
      process.on('SIGINT', cleanup);
      process.on('SIGTERM', cleanup);
      vp._cleanupListeners = cleanup;

      return vp;
    }

    // Unreachable — the loop either returns on success or throws on the final
    // attempt — but the type checker needs a terminal statement here.
    throw lastError instanceof Error ? lastError : new Error('failed to start vibium');
  }

  private _cleanupListeners: (() => void) | null = null;

  async stop(): Promise<void> {
    if (this._stopped) return;
    this._stopped = true;

    // Remove process exit listeners to avoid leaks
    if (this._cleanupListeners) {
      process.removeListener('exit', this._cleanupListeners);
      process.removeListener('SIGINT', this._cleanupListeners);
      process.removeListener('SIGTERM', this._cleanupListeners);
      this._cleanupListeners = null;
    }

    return new Promise((resolve) => {
      let resolved = false;
      const done = () => {
        if (!resolved) { resolved = true; resolve(); }
      };

      this._process.on('exit', done);

      // Close stdin to signal graceful shutdown
      try { this._process.stdin?.end(); } catch {}

      if (process.platform === 'win32') {
        try {
          execFileSync('taskkill', ['/T', '/F', '/PID', this._process.pid!.toString()], { stdio: 'ignore' });
        } catch {}
        done();
      } else {
        // SIGTERM after 1s if graceful shutdown hasn't worked
        setTimeout(() => {
          if (!resolved) {
            try { this._process.kill('SIGTERM'); } catch {}
          }
        }, 1000);

        // SIGKILL after 4s as last resort
        setTimeout(() => {
          if (!resolved) {
            try { this._process.kill('SIGKILL'); } catch {}
          }
        }, 4000);

        // Hard resolve after 5s — process is definitely dead by now
        setTimeout(done, 5000);
      }
    });
  }
}
