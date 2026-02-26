import { spawn, execFileSync, ChildProcess } from 'child_process';
import { getVibiumBinPath } from './binary';
import { TimeoutError, BrowserCrashedError } from '../utils/errors';

export interface VibiumProcessOptions {
  port?: number;
  headless?: boolean;
  executablePath?: string;
}

export class VibiumProcess {
  private process: ChildProcess;
  private _port: number;
  private _stopped: boolean = false;

  private constructor(process: ChildProcess, port: number) {
    this.process = process;
    this._port = port;
  }

  get port(): number {
    return this._port;
  }

  static async start(options: VibiumProcessOptions = {}): Promise<VibiumProcess> {
    const binaryPath = options.executablePath || getVibiumBinPath();
    const port = options.port || 0; // 0 means auto-select

    const args = ['serve', '--port', port.toString()];
    if (options.headless === true) {
      args.push('--headless');
    }

    const proc = spawn(binaryPath, args, {
      stdio: ['ignore', 'pipe', 'pipe'],
    });

    // Wait for the server to start and extract the port
    const actualPort = await new Promise<number>((resolve, reject) => {
      let output = '';
      let resolved = false;

      const timeout = setTimeout(() => {
        if (!resolved) {
          reject(new TimeoutError('vibium', 10000, 'waiting for vibium to start'));
        }
      }, 10000);

      const handleData = (data: Buffer) => {
        const text = data.toString();
        output += text;

        // Look for "Server listening on ws://localhost:PORT"
        const match = output.match(/Server listening on ws:\/\/localhost:(\d+)/);
        if (match && !resolved) {
          resolved = true;
          clearTimeout(timeout);
          resolve(parseInt(match[1], 10));
        }
      };

      proc.stdout?.on('data', handleData);
      proc.stderr?.on('data', handleData);

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
          reject(new BrowserCrashedError(code ?? 1, output));
        }
      });
    });

    const vp = new VibiumProcess(proc, actualPort);

    // Clean up child process when Node exits unexpectedly (Ctrl+C, uncaught exception, etc.)
    const cleanup = () => vp.stop();
    process.on('exit', cleanup);
    process.on('SIGINT', cleanup);
    process.on('SIGTERM', cleanup);
    vp._cleanupListeners = cleanup;

    return vp;
  }

  private _cleanupListeners: (() => void) | null = null;

  async stop(): Promise<void> {
    if (this._stopped) {
      return;
    }
    this._stopped = true;

    // Remove process exit listeners to avoid leaks
    if (this._cleanupListeners) {
      process.removeListener('exit', this._cleanupListeners);
      process.removeListener('SIGINT', this._cleanupListeners);
      process.removeListener('SIGTERM', this._cleanupListeners);
      this._cleanupListeners = null;
    }

    return new Promise((resolve) => {
      this.process.on('exit', () => {
        resolve();
      });

      if (process.platform === 'win32') {
        // On Windows, process.kill('SIGTERM') calls TerminateProcess() which
        // kills only the immediate process without letting cleanup code run.
        // Use taskkill /T to kill the entire process tree (vibium + chromedriver + Chrome).
        try {
          execFileSync('taskkill', ['/T', '/F', '/PID', this.process.pid!.toString()], { stdio: 'ignore' });
        } catch {
          // Process may have already exited
        }
        resolve();
      } else {
        // Try graceful shutdown first
        this.process.kill('SIGTERM');

        // Force kill after timeout
        setTimeout(() => {
          if (!this.process.killed) {
            this.process.kill('SIGKILL');
          }
          resolve();
        }, 3000);
      }
    });
  }
}
