import { VibiumProcess } from './clicker';
import { BiDiClient, BiDiEvent } from './bidi';
import { Page } from './page';
import { BrowserContext } from './context';
import { debug, info } from './utils/debug';
import { checkNodeVersion } from './utils/version';

export interface LaunchOptions {
  headless?: boolean;
  port?: number;
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

  /** Close the browser and clean up. */
  async close(): Promise<void> {
    await this.client.send('vibium:browser.close', {});
    await this.client.close();
    if (this.process) {
      await this.process.stop();
    }
  }
}

export const browser = {
  async launch(options: LaunchOptions = {}): Promise<Browser> {
    const { headless = false, port, executablePath } = options;
    debug('launching browser', { headless, port, executablePath });

    // Start the vibium process
    const process = await VibiumProcess.start({
      headless,
      port,
      executablePath,
    });
    debug('vibium started', { port: process.port });

    // Connect to the proxy
    const client = await BiDiClient.connect(`ws://localhost:${process.port}`);
    info('browser launched', { port: process.port });

    return new Browser(client, process);
  },
};
