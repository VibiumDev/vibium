import { BiDiClient } from './bidi';

export interface RecordingVideoOptions {
  /** Video width in pixels (defaults to the viewport). */
  width?: number;
  /** Video height in pixels (defaults to the viewport). */
  height?: number;
  /** Video frame rate (engine default if omitted). */
  frameRate?: number;
  /**
   * On a remote browser connection, 'keep' records anyway and leaves the
   * video on the remote host: the stop result and zip manifest carry its
   * remotePath, and retrieval and cleanup are yours. Ignored on local
   * connections.
   */
  remote?: 'keep';
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
   * Where the recording zip lands at stop. Defaults to a timestamped
   * record-YYYYMMDD-HHMMSS.zip in the working directory, so a rerun never
   * clobbers the previous one. `null` selects bytes-only capture: no file
   * is written and the recording is lost if the session closes before
   * stop().
   */
  path?: string | null;
}

export interface RecordingStopOptions {
  /** Overrides the path declared at start. */
  path?: string;
}

export interface RecordingVideoSummary {
  context: string;
  durationMs: number;
  width: number;
  height: number;
  /** Where a remote-keep video lives on the remote host. */
  remotePath?: string;
  /** Set when the video pipeline died; the zip delivered without or with a partial video. */
  error?: string;
}

export interface RecordingResult {
  /** Where the zip landed. Absent for bytes-only recordings. */
  path?: string;
  /** The zip itself. Present only for bytes-only recordings (start path: null). */
  bytes?: Buffer;
  steps?: number;
  durationMs?: number;
  videos?: RecordingVideoSummary[];
  /** Why no video was recorded (video omitted on an engine without support). */
  videoUnavailable?: string;
}

/**
 * Timestamped default destination so a rerun never clobbers the previous
 * artifact. The recording's name, sanitized, seeds the stem: name 'login'
 * yields login-20260808-094123.zip.
 */
function defaultRecordName(fs: typeof import('fs'), name?: string): string {
  const stem = (name ?? '').replace(/[^a-zA-Z0-9._-]/g, '-').replace(/^[-.]+|[-.]+$/g, '') || 'record';
  const d = new Date();
  const pad = (n: number) => String(n).padStart(2, '0');
  const stamp = `${d.getFullYear()}${pad(d.getMonth() + 1)}${pad(d.getDate())}-${pad(d.getHours())}${pad(d.getMinutes())}${pad(d.getSeconds())}`;
  let candidate = `${stem}-${stamp}.zip`;
  for (let n = 2; fs.existsSync(candidate); n++) {
    candidate = `${stem}-${stamp}-${n}.zip`;
  }
  return candidate;
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
      if (declaredPath !== undefined) {
        params.path = path.resolve(declaredPath);
      } else {
        const fs = await import('fs');
        params.path = path.resolve(defaultRecordName(fs, rest.name));
      }
    }
    await this.client.send('vibium:recording.start', params);
  }

  /**
   * Stop recording and deliver the zip to the declared path.
   * The result carries where it landed and what it holds; the zip bytes are
   * included only for bytes-only recordings (start path: null).
   */
  async stop(options: RecordingStopOptions = {}): Promise<RecordingResult> {
    const params: Record<string, unknown> = {
      userContext: this.userContextId,
    };
    if (options.path) {
      const path = await import('path');
      params.path = path.resolve(options.path);
    }
    const { data, ...rest } = await this.client.send<{ data?: string } & RecordingResult>(
      'vibium:recording.stop', params);

    const result: RecordingResult = { ...rest };
    if (data) {
      result.bytes = Buffer.from(data, 'base64');
    }
    return result;
  }

  /** Start a new recording chunk (resets event buffer, keeps resources). */
  async startChunk(options: { name?: string; title?: string } = {}): Promise<void> {
    await this.client.send('vibium:recording.startChunk', {
      userContext: this.userContextId,
      ...options,
    });
  }

  /**
   * Stop the current chunk. The result carries the path it was written to,
   * or the chunk's bytes when no path was given.
   */
  async stopChunk(options: RecordingStopOptions = {}): Promise<RecordingResult> {
    const params: Record<string, unknown> = {
      userContext: this.userContextId,
    };
    if (options.path) {
      const path = await import('path');
      params.path = path.resolve(options.path);
    }
    const { data, ...rest } = await this.client.send<{ data?: string } & RecordingResult>(
      'vibium:recording.stopChunk', params);

    const result: RecordingResult = { ...rest };
    if (data) {
      result.bytes = Buffer.from(data, 'base64');
    }
    return result;
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
