import { BiDiConnection } from './connection';
import { BiDiCommand, BiDiResponse, BiDiEvent, isResponse, isEvent } from './types';

export type EventHandler = (event: BiDiEvent) => void;

export class BiDiClient {
  private connection: BiDiConnection;
  private nextId: number = 1;
  private pendingCommands: Map<number, {
    resolve: (result: unknown) => void;
    reject: (error: Error) => void;
  }> = new Map();
  private eventHandler: EventHandler | null = null;

  private constructor(connection: BiDiConnection) {
    this.connection = connection;

    connection.onMessage((msg) => {
      if (isResponse(msg)) {
        this.handleResponse(msg);
      } else if (isEvent(msg)) {
        this.handleEvent(msg);
      }
    });
  }

  static async connect(url: string): Promise<BiDiClient> {
    const connection = await BiDiConnection.connect(url);
    return new BiDiClient(connection);
  }

  private handleResponse(response: BiDiResponse): void {
    const pending = this.pendingCommands.get(response.id);
    if (!pending) {
      console.warn('Received response for unknown command:', response.id);
      return;
    }

    this.pendingCommands.delete(response.id);

    if (response.type === 'error' && response.error) {
      const errorStr = typeof response.error === 'string'
        ? response.error
        : JSON.stringify(response.error);
      const messageStr = typeof response.message === 'string'
        ? response.message
        : (response.message !== undefined ? JSON.stringify(response.message) : 'unknown error');
      pending.reject(new Error(`${errorStr}: ${messageStr}`));
    } else {
      pending.resolve(response.result);
    }
  }

  private handleEvent(event: BiDiEvent): void {
    if (this.eventHandler) {
      this.eventHandler(event);
    }
  }

  onEvent(handler: EventHandler): void {
    this.eventHandler = handler;
  }

  send<T = unknown>(method: string, params: Record<string, unknown> = {}): Promise<T> {
    return new Promise((resolve, reject) => {
      const id = this.nextId++;
      const command: BiDiCommand = { id, method, params };

      this.pendingCommands.set(id, {
        resolve: resolve as (result: unknown) => void,
        reject,
      });

      try {
        this.connection.send(JSON.stringify(command));
      } catch (err) {
        this.pendingCommands.delete(id);
        reject(err);
      }
    });
  }

  async close(): Promise<void> {
    // Reject all pending commands
    for (const [id, pending] of this.pendingCommands) {
      pending.reject(new Error('Connection closed'));
      this.pendingCommands.delete(id);
    }

    await this.connection.close();
  }
}
