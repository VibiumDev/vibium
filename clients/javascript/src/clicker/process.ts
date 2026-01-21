import { spawn, ChildProcess } from 'child_process';
import { getClickerPath } from './binary';
import { TimeoutError, BrowserCrashedError } from '../utils/errors';

export interface ClickerProcessOptions {
  port?: number;
  headless?: boolean;
  executablePath?: string;
  width?: number;
  height?: number;
}

export class ClickerProcess {
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

  static async start(options: ClickerProcessOptions = {}): Promise<ClickerProcess> {
    const binaryPath = options.executablePath || getClickerPath();
    const port = options.port || 0; // 0 means auto-select

    const args = ['serve'];
    if (port > 0) {
      args.push('--port', port.toString());
    }
    if (options.headless === true) {
      args.push('--headless');
    }
    if (options.width !== undefined) {
      args.push('--width', options.width.toString());
    }
    if (options.height !== undefined) {
      args.push('--height', options.height.toString());
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
          reject(new TimeoutError('clicker', 10000, 'waiting for clicker to start'));
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

    return new ClickerProcess(proc, actualPort);
  }

  async stop(): Promise<void> {
    if (this._stopped) {
      return;
    }
    this._stopped = true;

    return new Promise((resolve) => {
      this.process.on('exit', () => {
        resolve();
      });

      // Try graceful shutdown first
      this.process.kill('SIGTERM');

      // Force kill after timeout
      setTimeout(() => {
        if (!this.process.killed) {
          this.process.kill('SIGKILL');
        }
        resolve();
      }, 3000);
    });
  }
}
