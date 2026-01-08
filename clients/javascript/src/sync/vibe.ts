import { SyncBridge } from './bridge';
import { ElementSync } from './element';
import { ElementInfo } from '../element';
import { FindOptions, RecordingOptions } from '../vibe';

export class VibeSync {
  private bridge: SyncBridge;

  constructor(bridge: SyncBridge) {
    this.bridge = bridge;
  }

  go(url: string): void {
    this.bridge.call('go', [url]);
  }

  screenshot(): Buffer {
    const result = this.bridge.call<{ data: string }>('screenshot');
    return Buffer.from(result.data, 'base64');
  }

  /**
   * Execute JavaScript in the page context.
   */
  evaluate<T = unknown>(script: string): T {
    const result = this.bridge.call<{ result: T }>('evaluate', [script]);
    return result.result;
  }

  /**
   * Find an element by CSS selector.
   * Waits for element to exist before returning.
   */
  find(selector: string, options?: FindOptions): ElementSync {
    const result = this.bridge.call<{ elementId: number; info: ElementInfo }>('find', [selector, options]);
    return new ElementSync(this.bridge, result.elementId, result.info);
  }

  /**
   * Start recording the browser session as a video.
   * Requires FFmpeg to be installed on the system.
   */
  startRecording(options?: RecordingOptions): void {
    this.bridge.call('startRecording', [options]);
  }

  /**
   * Stop recording and save the video file.
   * @returns Path to the saved video file
   */
  stopRecording(): string {
    const result = this.bridge.call<{ outputPath: string }>('stopRecording');
    return result.outputPath;
  }

  quit(): void {
    this.bridge.call('quit');
    this.bridge.terminate();
  }
}
