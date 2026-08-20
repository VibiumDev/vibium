import { VibiumProcess } from './clicker';
import { BiDiClient, BiDiEvent } from './bidi';
import { Page } from './page';
import { BrowserContext } from './context';
import { debug, info } from './utils/debug';
import { checkNodeVersion } from './utils/version';

const customInspect = Symbol.for('nodejs.util.inspect.custom');

export interface StartOptions {
  /** Browser engine to launch: 'chrome' (default) or 'firefox'. */
  engine?: 'chrome' | 'firefox';
  /**
   * Release channel of the engine to install and run, e.g. 'beta'.
   * Currently honored by Firefox only.
   */
  channel?: string;
  headless?: boolean;
  headers?: Record<string, string>;
  executablePath?: string;
}

export class Browser {
  private client: BiDiClient;
  private process: VibiumProcess | null;
  private pageCallbacks: ((page: Page) => void)[] = [];
  private popupCallbacks: ((page: Page) => void)[] = [];

  constructor(client: BiDiClient, process: VibiumProcess | null) {
    this.client = client;
    this.process = process;

    // Listen for browsingContext.contextCreated events
    this.client.onEvent((event: BiDiEvent) => {
      if (event.method === 'browsingContext.contextCreated') {
        const params = event.params as {
          context: string;
          url: string;
          userContext?: string;
          originalOpener?: string;
        };

        // Only create a Page if there are callbacks to deliver it to.
        // Creating a Page registers event handlers (network, dialog) that
        // would otherwise leak and interfere with the real Page's handlers.
        const callbacks = params.originalOpener ? this.popupCallbacks : this.pageCallbacks;
        if (callbacks.length > 0) {
          const page = new Page(this.client, params.context, params.userContext || 'default');
          for (const cb of callbacks) {
            cb(page);
          }
        }
      }
    });
  }

  [customInspect](): string {
    return 'Browser { connected: true }';
  }

  /** Get the default page (first browsing context). */
  async page(): Promise<Page> {
    const result = await this.client.send<{ context: string; userContext: string }>('vibium:browser.page', {});
    return new Page(this.client, result.context, result.userContext);
  }

  /** Create a new page (tab) in the default context. */
  async newPage(): Promise<Page> {
    const result = await this.client.send<{ context: string; userContext: string }>('vibium:browser.newPage', {});
    return new Page(this.client, result.context, result.userContext);
  }

  /** Create a new browser context (isolated, incognito-like). */
  async newContext(): Promise<BrowserContext> {
    const result = await this.client.send<{ userContext: string }>('vibium:browser.newContext', {});
    return new BrowserContext(this.client, result.userContext);
  }

  /** Get all open pages. */
  async pages(): Promise<Page[]> {
    const result = await this.client.send<{ pages: { context: string; url: string; userContext: string }[] }>('vibium:browser.pages', {});
    return result.pages.map(p => new Page(this.client, p.context, p.userContext));
  }

  /** Register a callback for when a new page is created (e.g. new tab). */
  onPage(callback: (page: Page) => void): void {
    this.pageCallbacks.push(callback);
  }

  /** Register a callback for when a popup is opened (window.open or target=_blank). */
  onPopup(callback: (page: Page) => void): void {
    this.popupCallbacks.push(callback);
  }

  /**
   * Remove all listeners for a given event, or all events if no event specified.
   * Supported events: 'page', 'popup'.
   */
  removeAllListeners(event?: 'page' | 'popup'): void {
    if (!event || event === 'page') {
      this.pageCallbacks = [];
    }
    if (!event || event === 'popup') {
      this.popupCallbacks = [];
    }
  }

  /** Stop the browser and clean up. */
  async stop(): Promise<void> {
    try {
      await this.client.send('vibium:browser.stop', {});
    } catch {
      // Browser or connection may already be closed
    }
    await this.client.close();
    if (this.process) {
      await this.process.stop();
    }
  }
}

function envHeaders(): Record<string, string> {
  const apiKey = process.env.VIBIUM_CONNECT_API_KEY;
  return apiKey ? { Authorization: `Bearer ${apiKey}` } : {};
}

export const browser = {
  async start(urlOrOptions?: string | StartOptions, options: StartOptions = {}): Promise<Browser> {
    let url: string | undefined;
    if (typeof urlOrOptions === 'object') {
      options = urlOrOptions;
      url = undefined;
    } else {
      url = urlOrOptions;
    }
    const connectURL = url || process.env.VIBIUM_CONNECT_URL;
    if (connectURL) {
      const headers = { ...envHeaders(), ...options.headers };
      debug('connecting to remote browser', { url: connectURL });

      const proc = await VibiumProcess.start({
        connectURL,
        connectHeaders: Object.keys(headers).length ? headers : undefined,
        executablePath: options.executablePath,
      });
      debug('vibium started (connect mode)');

      const client = BiDiClient.fromStreams(
        proc.stdin,
        proc.stdout,
        proc.preReadyLines,
      );
      info('browser connected (pipe → remote)');

      return new Browser(client, proc);
    }

    const { engine, channel, headless = false, executablePath } = options;
    debug('launching browser', { engine, channel, headless, executablePath });

    const proc = await VibiumProcess.start({
      engine,
      channel,
      headless,
      executablePath,
    });
    debug('vibium started');

    const client = BiDiClient.fromStreams(
      proc.stdin,
      proc.stdout,
      proc.preReadyLines,
    );
    info('browser launched (pipe)');

    return new Browser(client, proc);
  },
};

/**
 * Named engine launchers, Playwright-style. `firefox.start()` is
 * `browser.start({ engine: 'firefox' })`; options are otherwise identical.
 */
export const firefox = {
  start(options: Omit<StartOptions, 'engine'> = {}): Promise<Browser> {
    return browser.start({ ...options, engine: 'firefox' });
  },
};

export const chrome = {
  start(options: Omit<StartOptions, 'engine'> = {}): Promise<Browser> {
    return browser.start({ ...options, engine: 'chrome' });
  },
};
