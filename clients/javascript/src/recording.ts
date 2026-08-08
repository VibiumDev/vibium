import { BiDiClient } from './bidi';

export interface RecordingVideoOptions {
  /** Video width in pixels (defaults to the viewport). */
  width?: number;
  /** Video height in pixels (defaults to the viewport). */
  height?: number;
  /** Video frame rate (engine default if omitted). */
  frameRate?: number;
}

export interface RecordingStartOptions {
  name?: string;
  screenshots?: boolean;
  snapshots?: boolean;
  sources?: boolean;
  title?: string;
  bidi?: boolean;
  /** Screenshot format: 'jpeg' (default, faster/smaller) or 'png' (lossless). */
  format?: 'jpeg' | 'png';
  /** JPEG quality 0.0-1.0 (default 0.5). Ignored for PNG. */
  quality?: number;
  /**
   * Video track (Firefox 154+, local browsers). Omitted: record video if the
   * engine supports it; the stop result reports videoUnavailable otherwise.
   * `true` (or explicit dimensions): start fails if the engine can't deliver.
   * `false`: off.
   */
  video?: boolean | RecordingVideoOptions;
  /**
   * Where the recording zip lands at stop (default: record.zip in the
   * working directory). `null` selects bytes-only capture: no file is
   * written and the recording is lost if the session closes before stop().
   */
  path?: string | null;
}

export interface RecordingStopOptions {
  /** Overrides the path declared at start. */
  path?: string;
}

export class Recording {
  private client: BiDiClient;
  private userContextId: string;

  constructor(client: BiDiClient, userContextId: string) {
    this.client = client;
    this.userContextId = userContextId;
  }

  /** Start recording. */
  async start(options: RecordingStartOptions = {}): Promise<void> {
    const { path: declaredPath, ...rest } = options;
    const params: Record<string, unknown> = {
      userContext: this.userContextId,
      ...rest,
    };
    // The binary's working directory is not necessarily ours, so relative
    // paths resolve here before going over the wire. null = bytes-only.
    if (declaredPath !== null) {
      const path = await import('path');
      params.path = path.resolve(declaredPath ?? 'record.zip');
    }
    await this.client.send('vibium:recording.start', params);
  }

  /** Stop recording and return the recording zip as a Buffer. */
  async stop(options: RecordingStopOptions = {}): Promise<Buffer> {
    const params: Record<string, unknown> = {
      userContext: this.userContextId,
    };
    if (options.path) {
      const path = await import('path');
      params.path = path.resolve(options.path);
    }
    const result = await this.client.send<{ path?: string; data?: string }>('vibium:recording.stop', params);

    if (result.path) {
      // File was written by the engine; read it back
      const fs = await import('fs');
      return fs.readFileSync(result.path);
    }

    // Base64-encoded zip returned inline (path: null recordings)
    return Buffer.from(result.data!, 'base64');
  }

  /** Start a new recording chunk (resets event buffer, keeps resources). */
  async startChunk(options: { name?: string; title?: string } = {}): Promise<void> {
    await this.client.send('vibium:recording.startChunk', {
      userContext: this.userContextId,
      ...options,
    });
  }

  /** Stop the current chunk and return the recording zip as a Buffer. */
  async stopChunk(options: RecordingStopOptions = {}): Promise<Buffer> {
    const result = await this.client.send<{ path?: string; data?: string }>('vibium:recording.stopChunk', {
      userContext: this.userContextId,
      ...options,
    });

    if (options.path) {
      const fs = await import('fs');
      return fs.readFileSync(result.path!);
    }

    return Buffer.from(result.data!, 'base64');
  }

  /** Start a named group of actions in the recording. */
  async startGroup(name: string, options: { location?: { file: string; line?: number; column?: number } } = {}): Promise<void> {
    await this.client.send('vibium:recording.startGroup', {
      userContext: this.userContextId,
      name,
      ...options,
    });
  }

  /** End the current group. */
  async stopGroup(): Promise<void> {
    await this.client.send('vibium:recording.stopGroup', {
      userContext: this.userContextId,
    });
  }
}
