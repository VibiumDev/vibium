import { BiDiClient } from './bidi';

export interface ClockInstallOptions {
  /** Initial time as epoch ms, ISO date string, or Date object. */
  time?: number | string | Date;
  /** IANA timezone ID to override (e.g. "America/New_York", "Europe/London"). */
  timezone?: string;
}

/** Normalize a time value to epoch milliseconds. */
function normalizeTime(time: number | string | Date): number {
  if (typeof time === 'number') return time;
  if (time instanceof Date) return time.getTime();
  return new Date(time).getTime();
}

/** Page-scoped clock control for faking timers and Date. */
export class Clock {
  private client: BiDiClient;
  private contextId: string;

  constructor(client: BiDiClient, contextId: string) {
    this.client = client;
    this.contextId = contextId;
  }

  /** Install fake clock, overriding Date, setTimeout, setInterval, etc. */
  async install(options?: ClockInstallOptions): Promise<void> {
    const params: Record<string, unknown> = { context: this.contextId };
    if (options?.time !== undefined) {
      params.time = normalizeTime(options.time);
    }
    if (options?.timezone !== undefined) {
      params.timezone = options.timezone;
    }
    await this.client.send('vibium:clock.install', params);
  }

  /** Jump forward by ticks ms, fire each due timer at most once. */
  async fastForward(ticks: number): Promise<void> {
    await this.client.send('vibium:clock.fastForward', {
      context: this.contextId,
      ticks,
    });
  }

  /** Advance ticks ms, firing all callbacks systematically (including interval reschedules). */
  async runFor(ticks: number): Promise<void> {
    await this.client.send('vibium:clock.runFor', {
      context: this.contextId,
      ticks,
    });
  }

  /** Jump to a specific time and pause — no timers fire until resumed/advanced. */
  async pauseAt(time: number | string | Date): Promise<void> {
    await this.client.send('vibium:clock.pauseAt', {
      context: this.contextId,
      time: normalizeTime(time),
    });
  }

  /** Resume real-time progression from current fake time. */
  async resume(): Promise<void> {
    await this.client.send('vibium:clock.resume', {
      context: this.contextId,
    });
  }

  /** Freeze Date.now() at a value permanently. Timers still run. */
  async setFixedTime(time: number | string | Date): Promise<void> {
    await this.client.send('vibium:clock.setFixedTime', {
      context: this.contextId,
      time: normalizeTime(time),
    });
  }

  /** Set Date.now() without triggering timers. */
  async setSystemTime(time: number | string | Date): Promise<void> {
    await this.client.send('vibium:clock.setSystemTime', {
      context: this.contextId,
      time: normalizeTime(time),
    });
  }

  /** Override the browser timezone. Pass an IANA timezone ID, or empty string to reset to system default. */
  async setTimezone(timezone: string): Promise<void> {
    await this.client.send('vibium:clock.setTimezone', {
      context: this.contextId,
      timezone,
    });
  }
}
