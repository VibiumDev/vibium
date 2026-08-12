import { createInterface, Interface as ReadlineInterface } from 'readline';
import { Writable, Readable } from 'stream';
import { BiDiCommand, BiDiResponse, BiDiEvent, BiDiMessage, isResponse, isEvent } from './types';

export type EventHandler = (event: BiDiEvent) => void;

const DEFAULT_COMMAND_TIMEOUT = 60_000;

// Waiting for event setup would deadlock this one: an open dialog stops Chrome
// answering anything else for that context, including the script.callFunction
// that installs a WebSocket monitor, and this is the only command that closes
// the dialog. Mirrors unblocksAnotherCommand in the Go router.
const NEVER_WAITS = 'browsingContext.handleUserPrompt';

export class BiDiClient {
  private stdin: Writable;
  private rl: ReadlineInterface;
  private nextId: number = 1;
  private pendingCommands: Map<number, {
    resolve: (result: unknown) => void;
    reject: (error: Error) => void;
    timer: ReturnType<typeof setTimeout>;
  }> = new Map();
  private eventHandlers: EventHandler[] = [];
  private _closed: boolean = false;

  // Event registration (page.onWebSocket, network.addDataCollector) is
  // triggered from synchronous callback APIs with nothing for the caller to
  // await, so its command used to be fired and forgotten. The next command — an
  // eval that opens a socket — could reach the engine first, and the one-shot
  // event was lost (#351).
  //
  // sendSetup() parks a gate here that send() waits on. Settle-only: the gate
  // sequences commands and nothing more. A failed setup is surfaced by whoever
  // owns that registration, not injected into an unrelated command.
  private setupGate: Promise<void> | null = null;

  private constructor(stdin: Writable, stdout: Readable) {
    this.stdin = stdin;

    // Read responses/events line by line from stdout
    this.rl = createInterface({ input: stdout, crlfDelay: Infinity });

    this.rl.on('line', (line: string) => {
      const trimmed = line.trim();
      if (!trimmed) return;
      try {
        const msg = JSON.parse(trimmed) as BiDiMessage;
        if (isResponse(msg)) {
          this.handleResponse(msg);
        } else if (isEvent(msg)) {
          this.handleEvent(msg);
        }
      } catch (err) {
        // Ignore unparseable lines
      }
    });

    this.rl.on('close', () => {
      this._closed = true;
      for (const [id, pending] of this.pendingCommands) {
        clearTimeout(pending.timer);
        pending.reject(new Error('Connection closed unexpectedly'));
        this.pendingCommands.delete(id);
      }
    });
  }

  /**
   * Create a BiDiClient from stdin/stdout streams.
   * Optionally replay pre-ready lines (events received before the ready signal).
   */
  static fromStreams(stdin: Writable, stdout: Readable, preReadyLines: string[] = []): BiDiClient {
    const client = new BiDiClient(stdin, stdout);
    // Replay buffered events
    for (const line of preReadyLines) {
      try {
        const msg = JSON.parse(line) as BiDiMessage;
        if (isEvent(msg)) {
          client.handleEvent(msg);
        }
      } catch {
        // Ignore
      }
    }
    return client;
  }

  private handleResponse(response: BiDiResponse): void {
    const pending = this.pendingCommands.get(response.id);
    if (!pending) {
      return;
    }

    clearTimeout(pending.timer);
    this.pendingCommands.delete(response.id);

    if (response.type === 'error' && response.error) {
      pending.reject(new Error(`${response.error}: ${response.message}`));
    } else {
      pending.resolve(response.result);
    }
  }

  private handleEvent(event: BiDiEvent): void {
    for (const handler of this.eventHandlers) {
      handler(event);
    }
  }

  onEvent(handler: EventHandler): void {
    this.eventHandlers.push(handler);
  }

  offEvent(handler: EventHandler): void {
    this.eventHandlers = this.eventHandlers.filter(h => h !== handler);
  }

  send<T = unknown>(method: string, params: Record<string, unknown> = {}, timeout: number = DEFAULT_COMMAND_TIMEOUT): Promise<T> {
    const gate = this.setupGate;
    if (gate === null || method === NEVER_WAITS) {
      return this.dispatch<T>(method, params, timeout);
    }
    // Chaining directly on the gate keeps issue order: these reactions run in
    // registration order, and the reaction that clears setupGate was registered
    // first (in sendSetup), so a command issued after the gate opens dispatches
    // strictly later.
    return gate.then(() => this.dispatch<T>(method, params, timeout));
  }

  /**
   * Send a command whose acknowledgement the next command depends on.
   *
   * Goes out immediately; commands sent before it is answered wait. For event
   * registration issued from a synchronous callback API, where the caller has no
   * promise to await (#351). The returned promise is the registration's — the
   * caller decides how a failure is surfaced.
   */
  sendSetup<T = unknown>(method: string, params: Record<string, unknown> = {}, timeout: number = DEFAULT_COMMAND_TIMEOUT): Promise<T> {
    const result = this.dispatch<T>(method, params, timeout);
    const settled = result.then(() => undefined, () => undefined);
    const gate = this.setupGate === null
      ? settled
      : Promise.all([this.setupGate, settled]).then(() => undefined);
    this.setupGate = gate;
    void gate.then(() => {
      // A registration made in the meantime replaced this gate and keeps
      // commands waiting; only the newest one clears.
      if (this.setupGate === gate) this.setupGate = null;
    });
    return result;
  }

  /** Write a command and wait for its response, bypassing the setup gate. */
  private dispatch<T = unknown>(method: string, params: Record<string, unknown>, timeout: number): Promise<T> {
    return new Promise((resolve, reject) => {
      const id = this.nextId++;
      const command: BiDiCommand = { id, method, params };

      const timer = setTimeout(() => {
        this.pendingCommands.delete(id);
        reject(new Error(`Command '${method}' timed out after ${timeout}ms`));
      }, timeout);

      this.pendingCommands.set(id, {
        resolve: resolve as (result: unknown) => void,
        reject,
        timer,
      });

      try {
        if (this._closed) {
          throw new Error('Connection closed');
        }
        this.stdin.write(JSON.stringify(command) + '\n');
      } catch (err) {
        clearTimeout(timer);
        this.pendingCommands.delete(id);
        reject(err);
      }
    });
  }

  async close(): Promise<void> {
    if (this._closed) {
      return;
    }
    this._closed = true;

    // Reject all pending commands
    for (const [id, pending] of this.pendingCommands) {
      clearTimeout(pending.timer);
      pending.reject(new Error('Connection closed'));
      this.pendingCommands.delete(id);
    }

    // Rejecting the in-flight setup settles the gate; drop it so a command
    // racing close() fails on the closed connection instead of waiting.
    this.setupGate = null;

    this.rl.close();
    try { this.stdin.end(); } catch {}
  }
}
