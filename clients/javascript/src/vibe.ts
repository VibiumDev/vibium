import { BiDiClient, BrowsingContextTree, NavigationResult, ScreenshotResult } from './bidi';
import { ClickerProcess } from './clicker';
import { Element, ElementInfo } from './element';
import { debug } from './utils/debug';

export interface FindOptions {
  /** Timeout in milliseconds to wait for element. Default: 30000 */
  timeout?: number;
}

export interface RecordingOptions {
  /** Frames per second. Default: 10 */
  fps?: number;
  /** Output format: 'mp4' or 'webm'. Default: 'mp4' */
  format?: 'mp4' | 'webm';
  /** Output file path. If not provided, uses temp directory */
  outputPath?: string;
}

interface StartRecordingResult {
  started: boolean;
  fps: number;
  format: string;
}

interface StopRecordingResult {
  stopped: boolean;
  outputPath: string;
}

interface VibiumFindResult {
  tag: string;
  text: string;
  box: {
    x: number;
    y: number;
    width: number;
    height: number;
  };
}

export class Vibe {
  private client: BiDiClient;
  private process: ClickerProcess | null;
  private context: string | null = null;

  constructor(client: BiDiClient, process: ClickerProcess | null) {
    this.client = client;
    this.process = process;
  }

  private async getContext(): Promise<string> {
    if (this.context) {
      return this.context;
    }

    const tree = await this.client.send<BrowsingContextTree>('browsingContext.getTree', {});
    if (!tree.contexts || tree.contexts.length === 0) {
      throw new Error('No browsing context available');
    }

    this.context = tree.contexts[0].context;
    return this.context;
  }

  async go(url: string): Promise<void> {
    debug('navigating', { url });
    const context = await this.getContext();
    await this.client.send<NavigationResult>('browsingContext.navigate', {
      context,
      url,
      wait: 'complete',
    });
    debug('navigation complete', { url });
  }

  async screenshot(): Promise<Buffer> {
    const context = await this.getContext();
    const result = await this.client.send<ScreenshotResult>('browsingContext.captureScreenshot', {
      context,
    });
    return Buffer.from(result.data, 'base64');
  }

  /**
   * Execute JavaScript in the page context.
   */
  async evaluate<T = unknown>(script: string): Promise<T> {
    const context = await this.getContext();
    const result = await this.client.send<{
      type: string;
      result: { type: string; value?: T };
    }>('script.callFunction', {
      functionDeclaration: `() => { ${script} }`,
      target: { context },
      arguments: [],
      awaitPromise: true,
      resultOwnership: 'root',
    });

    return result.result.value as T;
  }

  /**
   * Find an element by CSS selector.
   * Waits for element to exist before returning.
   */
  async find(selector: string, options?: FindOptions): Promise<Element> {
    debug('finding element', { selector, timeout: options?.timeout });
    const context = await this.getContext();

    const result = await this.client.send<VibiumFindResult>('vibium:find', {
      context,
      selector,
      timeout: options?.timeout,
    });

    const info: ElementInfo = {
      tag: result.tag,
      text: result.text,
      box: result.box,
    };
    debug('element found', { selector, tag: result.tag });

    return new Element(this.client, context, selector, info);
  }

  /**
   * Start recording the browser session as a video.
   * Requires FFmpeg to be installed on the system.
   *
   * @param options - Recording options (fps, format, outputPath)
   * @returns Promise that resolves when recording starts
   *
   * @example
   * ```typescript
   * await vibe.startRecording({ fps: 10, format: 'mp4' });
   * // ... perform actions ...
   * const videoPath = await vibe.stopRecording();
   * ```
   */
  async startRecording(options?: RecordingOptions): Promise<void> {
    debug('starting recording', { fps: options?.fps, format: options?.format, outputPath: options?.outputPath });
    const context = await this.getContext();

    await this.client.send<StartRecordingResult>('vibium:startRecording', {
      context,
      fps: options?.fps,
      format: options?.format,
      outputPath: options?.outputPath,
    });

    debug('recording started');
  }

  /**
   * Stop recording and save the video file.
   *
   * @returns Promise that resolves with the path to the saved video file
   *
   * @example
   * ```typescript
   * const videoPath = await vibe.stopRecording();
   * console.log('Video saved to:', videoPath);
   * ```
   */
  async stopRecording(): Promise<string> {
    debug('stopping recording');
    const context = await this.getContext();

    const result = await this.client.send<StopRecordingResult>('vibium:stopRecording', {
      context,
    });

    debug('recording stopped', { outputPath: result.outputPath });
    return result.outputPath;
  }

  async quit(): Promise<void> {
    await this.client.close();
    if (this.process) {
      await this.process.stop();
    }
  }
}
