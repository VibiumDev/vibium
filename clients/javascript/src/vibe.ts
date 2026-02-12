import { BiDiClient, BrowsingContextTree, NavigationResult, ScreenshotResult } from './bidi';
import { ClickerProcess } from './clicker';
import { Element, ElementInfo } from './element';
import { debug } from './utils/debug';
import { TimeoutError } from './utils/errors';

export interface FindOptions {
  /** Timeout in milliseconds to wait for element. Default: 30000 */
  timeout?: number;
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

    const timeoutMs = options?.timeout ?? 30000;
    let result: VibiumFindResult;
    try {
      result = await this.client.send<VibiumFindResult>('vibium:find', {
        context,
        selector,
        timeout: timeoutMs,
      });
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : String(err);
      if (message.toLowerCase().includes('timeout') || message.includes('no node')) {
        throw new TimeoutError(selector, timeoutMs, message);
      }
      throw err;
    }

    const info: ElementInfo = {
      tag: result.tag,
      text: result.text,
      box: result.box,
    };
    debug('element found', { selector, tag: result.tag });

    return new Element(this.client, context, selector, info);
  }

  async quit(): Promise<void> {
    await this.client.close();
    if (this.process) {
      await this.process.stop();
    }
  }
}
