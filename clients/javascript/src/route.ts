import { BiDiClient } from './bidi';
import { Request } from './network';

/** Check if an error is a benign race condition that should be silently ignored. */
function isRaceConditionError(e: unknown): boolean {
  if (!(e instanceof Error)) return false;
  const msg = e.message;
  return msg === 'Connection closed' ||
    msg.includes('Invalid state') ||
    msg.includes('No blocked request') ||
    msg.includes('no such request');
}

/** Represents an intercepted network request that can be fulfilled, continued, or aborted. */
export class Route {
  readonly request: Request;
  private client: BiDiClient;
  private requestId: string;

  constructor(client: BiDiClient, requestId: string, request: Request) {
    this.client = client;
    this.requestId = requestId;
    this.request = request;
  }

  /** Fulfill the request with a custom response (mock response). */
  async fulfill(response: {
    status?: number;
    headers?: Record<string, string>;
    contentType?: string;
    body?: string;
  } = {}): Promise<void> {
    try {
      const params: Record<string, unknown> = { request: this.requestId };
      if (response.status !== undefined) params.statusCode = response.status;
      if (response.headers) params.headers = response.headers;
      if (response.contentType) params.contentType = response.contentType;
      if (response.body !== undefined) params.body = response.body;
      await this.client.send('vibium:network.fulfill', params);
    } catch (e) {
      if (isRaceConditionError(e)) return;
      throw e;
    }
  }

  /** Continue the request with optional overrides. */
  async continue(overrides?: {
    url?: string;
    method?: string;
    headers?: Record<string, string>;
    postData?: string;
  }): Promise<void> {
    try {
      const params: Record<string, unknown> = { request: this.requestId };
      if (overrides?.url) params.url = overrides.url;
      if (overrides?.method) params.method = overrides.method;
      if (overrides?.headers) params.headers = overrides.headers;
      if (overrides?.postData) params.postData = overrides.postData;
      await this.client.send('vibium:network.continue', params);
    } catch (e) {
      if (isRaceConditionError(e)) return;
      throw e;
    }
  }

  /** Abort the request. */
  async abort(): Promise<void> {
    try {
      await this.client.send('network.failRequest', {
        request: this.requestId,
      });
    } catch (e) {
      if (isRaceConditionError(e)) return;
      throw e;
    }
  }
}
