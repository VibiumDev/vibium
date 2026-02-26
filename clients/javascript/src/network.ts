/**
 * Network request and response data classes.
 * Wrap BiDi event data for onRequest/onResponse callbacks.
 */

import { BiDiClient } from './bidi';

interface BiDiHeaderEntry {
  name: string;
  value: { type: string; value: string };
}

/** Wraps a BiDi network.beforeRequestSent event. */
export class Request {
  private data: Record<string, unknown>;
  private client: BiDiClient | null;

  constructor(data: Record<string, unknown>, client?: BiDiClient) {
    this.data = data;
    this.client = client ?? null;
  }

  /** The request URL. */
  url(): string {
    const req = this.data.request as Record<string, unknown> | undefined;
    return (req?.url as string) ?? '';
  }

  /** The HTTP method (GET, POST, etc.). */
  method(): string {
    const req = this.data.request as Record<string, unknown> | undefined;
    return (req?.method as string) ?? '';
  }

  /** Request headers as a simple key-value object. */
  headers(): Record<string, string> {
    const req = this.data.request as Record<string, unknown> | undefined;
    const entries = (req?.headers as BiDiHeaderEntry[]) ?? [];
    return convertHeaders(entries);
  }

  /** The request ID (BiDi network request identifier). */
  requestId(): string {
    const req = this.data.request as Record<string, unknown> | undefined;
    return (req?.request as string) ?? '';
  }

  /** Request body / post data. Uses BiDi network.getData when a data collector is active. */
  async postData(): Promise<string | null> {
    if (!this.client) return null;
    const reqId = this.requestId();
    if (!reqId) return null;
    try {
      const result = await this.client.send<{ bytes: { type: string; value: string } }>(
        'network.getData',
        { dataType: 'request', request: reqId }
      );
      return result?.bytes?.value ?? null;
    } catch {
      return null;
    }
  }
}

/** Wraps a BiDi network.responseCompleted event. */
export class Response {
  private data: Record<string, unknown>;
  private client: BiDiClient | null;

  constructor(data: Record<string, unknown>, client?: BiDiClient) {
    this.data = data;
    this.client = client ?? null;
  }

  /** The response URL. */
  url(): string {
    const resp = this.data.response as Record<string, unknown> | undefined;
    return (resp?.url as string) ?? (this.data.url as string) ?? '';
  }

  /** The HTTP status code. */
  status(): number {
    const resp = this.data.response as Record<string, unknown> | undefined;
    return (resp?.status as number) ?? 0;
  }

  /** Response headers as a simple key-value object. */
  headers(): Record<string, string> {
    const resp = this.data.response as Record<string, unknown> | undefined;
    const entries = (resp?.headers as BiDiHeaderEntry[]) ?? [];
    return convertHeaders(entries);
  }

  /** The request ID associated with this response. */
  requestId(): string {
    const req = this.data.request as Record<string, unknown> | undefined;
    return (req?.request as string) ?? '';
  }

  /** Response body as a string. Requires a data collector (set up by onResponse/expect.response). */
  async body(): Promise<string | null> {
    if (!this.client) return null;
    const reqId = this.requestId();
    if (!reqId) return null;
    try {
      const result = await this.client.send<{ bytes: { type: string; value: string } }>(
        'network.getData',
        { dataType: 'response', request: reqId }
      );
      if (!result?.bytes?.value) return null;
      if (result.bytes.type === 'base64') {
        return Buffer.from(result.bytes.value, 'base64').toString('utf-8');
      }
      return result.bytes.value;
    } catch {
      return null;
    }
  }

  /** Response body parsed as JSON. Requires a data collector (set up by onResponse/expect.response). */
  async json(): Promise<unknown> {
    const text = await this.body();
    if (text === null) return null;
    return JSON.parse(text);
  }
}

/** Convert BiDi header entries [{name, value: {type, value}}] to Record<string, string>. */
function convertHeaders(entries: BiDiHeaderEntry[]): Record<string, string> {
  const result: Record<string, string> = {};
  for (const entry of entries) {
    result[entry.name] = entry.value?.value ?? '';
  }
  return result;
}
