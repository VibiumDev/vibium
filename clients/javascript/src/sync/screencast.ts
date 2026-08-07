import { SyncBridge } from './bridge';
import { ScreencastStartOptions, ScreencastStopOptions } from '../screencast';

export class ScreencastSync {
  private bridge: SyncBridge;
  private pageId: number;

  constructor(bridge: SyncBridge, pageId: number) {
    this.bridge = bridge;
    this.pageId = pageId;
  }

  start(options: ScreencastStartOptions = {}): void {
    this.bridge.call('screencast.start', [this.pageId, options]);
  }

  stop(options: ScreencastStopOptions = {}): Buffer {
    const result = this.bridge.call<{ data: string }>('screencast.stop', [this.pageId, options]);
    return Buffer.from(result.data, 'base64');
  }
}
