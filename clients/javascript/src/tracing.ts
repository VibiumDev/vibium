import { BiDiClient } from './bidi';

export interface TracingStartOptions {
  name?: string;
  screenshots?: boolean;
  snapshots?: boolean;
  sources?: boolean;
  title?: string;
  bidi?: boolean;
  /** Screenshot format: 'jpeg' (default, faster/smaller) or 'png' (lossless). */
  format?: 'jpeg' | 'png';
  /** JPEG quality 0.0-1.0 (default 0.8). Ignored for PNG. */
  quality?: number;
}

export interface TracingStopOptions {
  path?: string;
}

export class Tracing {
  private client: BiDiClient;
  private userContextId: string;

  constructor(client: BiDiClient, userContextId: string) {
    this.client = client;
    this.userContextId = userContextId;
  }

  /** Start trace recording. */
  async start(options: TracingStartOptions = {}): Promise<void> {
    await this.client.send('vibium:tracing.start', {
      userContext: this.userContextId,
      ...options,
    });
  }

  /** Stop trace recording and return the trace zip as a Buffer. */
  async stop(options: TracingStopOptions = {}): Promise<Buffer> {
    const result = await this.client.send<{ path?: string; data?: string }>('vibium:tracing.stop', {
      userContext: this.userContextId,
      ...options,
    });

    if (options.path) {
      // File was written by the engine; read it back
      const fs = await import('fs');
      return fs.readFileSync(result.path!);
    }

    // Base64-encoded zip returned inline
    return Buffer.from(result.data!, 'base64');
  }

  /** Start a new trace chunk (resets event buffer, keeps resources). */
  async startChunk(options: { name?: string; title?: string } = {}): Promise<void> {
    await this.client.send('vibium:tracing.startChunk', {
      userContext: this.userContextId,
      ...options,
    });
  }

  /** Stop the current chunk and return the trace zip as a Buffer. */
  async stopChunk(options: TracingStopOptions = {}): Promise<Buffer> {
    const result = await this.client.send<{ path?: string; data?: string }>('vibium:tracing.stopChunk', {
      userContext: this.userContextId,
      ...options,
    });

    if (options.path) {
      const fs = await import('fs');
      return fs.readFileSync(result.path!);
    }

    return Buffer.from(result.data!, 'base64');
  }

  /** Start a named group of actions in the trace. */
  async startGroup(name: string, options: { location?: { file: string; line?: number; column?: number } } = {}): Promise<void> {
    await this.client.send('vibium:tracing.startGroup', {
      userContext: this.userContextId,
      name,
      ...options,
    });
  }

  /** End the current group. */
  async stopGroup(): Promise<void> {
    await this.client.send('vibium:tracing.stopGroup', {
      userContext: this.userContextId,
    });
  }
}
