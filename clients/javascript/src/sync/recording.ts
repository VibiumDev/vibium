import { SyncBridge } from './bridge';
import { RecordingResult, RecordingStartOptions, RecordingStopOptions } from '../recording';

export class RecordingSync {
  private bridge: SyncBridge;
  private contextId: number;

  constructor(bridge: SyncBridge, contextId: number) {
    this.bridge = bridge;
    this.contextId = contextId;
  }

  start(options: RecordingStartOptions = {}): void {
    this.bridge.call('recording.start', [this.contextId, options]);
  }

  stop(options: RecordingStopOptions = {}): RecordingResult {
    const { data, ...rest } = this.bridge.call<{ data?: string } & RecordingResult>(
      'recording.stop', [this.contextId, options]);
    const result: RecordingResult = { ...rest };
    if (data) {
      result.bytes = Buffer.from(data, 'base64');
    }
    return result;
  }

  startChunk(options: { name?: string; title?: string } = {}): void {
    this.bridge.call('recording.startChunk', [this.contextId, options]);
  }

  stopChunk(options: RecordingStopOptions = {}): RecordingResult {
    const { data, ...rest } = this.bridge.call<{ data?: string } & RecordingResult>(
      'recording.stopChunk', [this.contextId, options]);
    const result: RecordingResult = { ...rest };
    if (data) {
      result.bytes = Buffer.from(data, 'base64');
    }
    return result;
  }

  startGroup(name: string, options: { location?: { file: string; line?: number; column?: number } } = {}): void {
    this.bridge.call('recording.startGroup', [this.contextId, name, options]);
  }

  stopGroup(): void {
    this.bridge.call('recording.stopGroup', [this.contextId]);
  }
}
