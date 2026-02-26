import { SyncBridge } from './bridge';
import { TracingStartOptions, TracingStopOptions } from '../tracing';

export class TracingSync {
  private bridge: SyncBridge;
  private contextId: number;

  constructor(bridge: SyncBridge, contextId: number) {
    this.bridge = bridge;
    this.contextId = contextId;
  }

  start(options: TracingStartOptions = {}): void {
    this.bridge.call('tracing.start', [this.contextId, options]);
  }

  stop(options: TracingStopOptions = {}): Buffer {
    const result = this.bridge.call<{ data: string }>('tracing.stop', [this.contextId, options]);
    return Buffer.from(result.data, 'base64');
  }

  startChunk(options: { name?: string; title?: string } = {}): void {
    this.bridge.call('tracing.startChunk', [this.contextId, options]);
  }

  stopChunk(options: TracingStopOptions = {}): Buffer {
    const result = this.bridge.call<{ data: string }>('tracing.stopChunk', [this.contextId, options]);
    return Buffer.from(result.data, 'base64');
  }

  startGroup(name: string, options: { location?: { file: string; line?: number; column?: number } } = {}): void {
    this.bridge.call('tracing.startGroup', [this.contextId, name, options]);
  }

  stopGroup(): void {
    this.bridge.call('tracing.stopGroup', [this.contextId]);
  }
}
