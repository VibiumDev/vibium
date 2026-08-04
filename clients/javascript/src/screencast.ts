import { BiDiClient } from './bidi';

export interface ScreencastStartOptions {
  /** Video MIME type (browser default if omitted, typically video/webm). */
  mimeType?: string;
  /** Requested video width in pixels. */
  width?: number;
  /** Requested video height in pixels. */
  height?: number;
  /** Requested frame rate. */
  frameRate?: number;
  /** Record page audio as well (default false). Firefox 154 does not support this yet. */
  audio?: boolean;
}

export interface ScreencastStopOptions {
  /** Save the video to this path. Omit to get the bytes back instead. */
  path?: string;
}

/**
 * Native browser video recording (WebDriver BiDi screencast).
 *
 * Supported on Firefox 154+. Chrome has not implemented the BiDi screencast
 * commands yet; start() fails there with an explanatory error.
 */
export class Screencast {
  private client: BiDiClient;
  private contextId: string;

  constructor(client: BiDiClient, contextId: string) {
    this.client = client;
    this.contextId = contextId;
  }

  /** Start recording this page. */
  async start(options: ScreencastStartOptions = {}): Promise<void> {
    await this.client.send('vibium:screencast.start', {
      context: this.contextId,
      ...options,
    });
  }

  /** Stop recording and return the video as a Buffer. */
  async stop(options: ScreencastStopOptions = {}): Promise<Buffer> {
    const result = await this.client.send<{ path?: string; data?: string }>('vibium:screencast.stop', {
      ...options,
    });

    if (options.path) {
      // File was written by the engine; read it back
      const fs = await import('fs');
      return fs.readFileSync(result.path!);
    }

    return Buffer.from(result.data!, 'base64');
  }
}
