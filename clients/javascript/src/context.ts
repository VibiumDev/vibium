import { BiDiClient } from './bidi';
import { Page } from './page';
import { Tracing } from './tracing';

export interface Cookie {
  name: string;
  value: string;
  domain: string;
  path: string;
  size: number;
  httpOnly: boolean;
  secure: boolean;
  sameSite: string;
  expiry?: number;
}

export interface SetCookieParam {
  name: string;
  value: string;
  domain?: string;
  url?: string;
  path?: string;
  httpOnly?: boolean;
  secure?: boolean;
  sameSite?: string;
  expiry?: number;
}

export interface OriginState {
  origin: string;
  localStorage: { name: string; value: string }[];
  sessionStorage: { name: string; value: string }[];
}

export interface StorageState {
  cookies: Cookie[];
  origins: OriginState[];
}

export class BrowserContext {
  private client: BiDiClient;
  private userContextId: string;
  readonly tracing: Tracing;

  constructor(client: BiDiClient, userContextId: string) {
    this.client = client;
    this.userContextId = userContextId;
    this.tracing = new Tracing(client, userContextId);
  }

  /** The user context ID for this browser context. */
  get id(): string {
    return this.userContextId;
  }

  /** Create a new page (tab) in this context. */
  async newPage(): Promise<Page> {
    const result = await this.client.send<{ context: string }>('vibium:context.newPage', {
      userContext: this.userContextId,
    });
    return new Page(this.client, result.context, this.userContextId);
  }

  /** Close this context and all its pages. */
  async close(): Promise<void> {
    await this.client.send('browser.removeUserContext', {
      userContext: this.userContextId,
    });
  }

  /** Get cookies for this context. Optionally filter by URLs. */
  async cookies(urls?: string[]): Promise<Cookie[]> {
    const params: Record<string, unknown> = { userContext: this.userContextId };
    if (urls && urls.length > 0) {
      params.urls = urls;
    }
    const result = await this.client.send<{ cookies: Cookie[] }>('vibium:context.cookies', params);
    return result.cookies;
  }

  /** Set cookies in this context. */
  async setCookies(cookies: SetCookieParam[]): Promise<void> {
    await this.client.send('vibium:context.setCookies', {
      userContext: this.userContextId,
      cookies,
    });
  }

  /** Clear all cookies in this context. */
  async clearCookies(): Promise<void> {
    await this.client.send('vibium:context.clearCookies', {
      userContext: this.userContextId,
    });
  }

  /** Get the storage state (cookies + localStorage + sessionStorage). */
  async storageState(): Promise<StorageState> {
    const result = await this.client.send<StorageState>('vibium:context.storageState', {
      userContext: this.userContextId,
    });
    return result;
  }

  /** Add an init script that runs before page scripts in this context. */
  async addInitScript(script: string): Promise<string> {
    const result = await this.client.send<{ script: string }>('vibium:context.addInitScript', {
      userContext: this.userContextId,
      script,
    });
    return result.script;
  }
}
