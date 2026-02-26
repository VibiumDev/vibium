import { SyncBridge } from './bridge';

export interface ClockInstallOptions {
  time?: number | string | Date;
  timezone?: string;
}

export class ClockSync {
  private bridge: SyncBridge;
  private pageId: number;

  constructor(bridge: SyncBridge, pageId: number) {
    this.bridge = bridge;
    this.pageId = pageId;
  }

  install(options?: ClockInstallOptions): void {
    const opts = options ? { ...options } : undefined;
    if (opts?.time instanceof Date) {
      opts.time = opts.time.getTime();
    }
    this.bridge.call('clock.install', [this.pageId, opts]);
  }

  fastForward(ticks: number): void {
    this.bridge.call('clock.fastForward', [this.pageId, ticks]);
  }

  runFor(ticks: number): void {
    this.bridge.call('clock.runFor', [this.pageId, ticks]);
  }

  pauseAt(time: number | string | Date): void {
    const t = time instanceof Date ? time.getTime() : time;
    this.bridge.call('clock.pauseAt', [this.pageId, t]);
  }

  resume(): void {
    this.bridge.call('clock.resume', [this.pageId]);
  }

  setFixedTime(time: number | string | Date): void {
    const t = time instanceof Date ? time.getTime() : time;
    this.bridge.call('clock.setFixedTime', [this.pageId, t]);
  }

  setSystemTime(time: number | string | Date): void {
    const t = time instanceof Date ? time.getTime() : time;
    this.bridge.call('clock.setSystemTime', [this.pageId, t]);
  }

  setTimezone(timezone: string): void {
    this.bridge.call('clock.setTimezone', [this.pageId, timezone]);
  }
}
