import { spawn, execFileSync, ChildProcess } from 'child_process';
import { getVibiumBinPath } from './binary';
import { TimeoutError, BrowserCrashedError } from '../utils/errors';

export interface VibiumProcessOptions {
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

          const timeout = setTimeout(() => {
            if (!resolved) {
              resolved = true;
              reject(new TimeoutError('vibium', 60000, 'waiting for vibium ready signal'));
            }
          }, 60000);

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
              reject(new BrowserCrashedError(code ?? 1, buffer));
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
