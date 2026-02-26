import { SyncBridge } from './bridge';
import { PageSync } from './page';
import { BrowserContextSync } from './context';

export interface LaunchOptions {
  headless?: boolean;
}

export class BrowserSync {
  /** @internal */
  readonly _bridge: SyncBridge;
  private _nextHandlerId = 0;
  private _pageHandlerId?: string;
  private _popupHandlerId?: string;

  constructor(bridge: SyncBridge) {
    this._bridge = bridge;
  }

  page(): PageSync {
    const result = this._bridge.call<{ pageId: number }>('browser.page');
    return new PageSync(this._bridge, result.pageId);
  }

  newPage(): PageSync {
    const result = this._bridge.call<{ pageId: number }>('browser.newPage');
    return new PageSync(this._bridge, result.pageId);
  }

  pages(): PageSync[] {
    const result = this._bridge.call<{ pageIds: number[] }>('browser.pages');
    return result.pageIds.map(id => new PageSync(this._bridge, id));
  }

  newContext(): BrowserContextSync {
    const result = this._bridge.call<{ contextId: number }>('browser.newContext');
    return new BrowserContextSync(this._bridge, result.contextId);
  }

  waitForPage(options?: { timeout?: number }): PageSync {
    const result = this._bridge.call<{ pageId: number }>('browser.waitForPage', [options]);
    return new PageSync(this._bridge, result.pageId);
  }

  waitForPopup(options?: { timeout?: number }): PageSync {
    const result = this._bridge.call<{ pageId: number }>('browser.waitForPopup', [options]);
    return new PageSync(this._bridge, result.pageId);
  }

  onPage(callback: (page: PageSync) => void): void {
    if (this._pageHandlerId) {
      this._bridge.unregisterHandler(this._pageHandlerId);
    }
    const handlerId = `page_${this._nextHandlerId++}`;
    this._bridge.registerHandler(handlerId, (data: { pageId: number }) => {
      callback(new PageSync(this._bridge, data.pageId));
    });
    this._pageHandlerId = handlerId;
    this._bridge.call('browser.onPage', [handlerId]);
  }

  onPopup(callback: (page: PageSync) => void): void {
    if (this._popupHandlerId) {
      this._bridge.unregisterHandler(this._popupHandlerId);
    }
    const handlerId = `popup_${this._nextHandlerId++}`;
    this._bridge.registerHandler(handlerId, (data: { pageId: number }) => {
      callback(new PageSync(this._bridge, data.pageId));
    });
    this._popupHandlerId = handlerId;
    this._bridge.call('browser.onPopup', [handlerId]);
  }

  removeAllListeners(event?: 'page' | 'popup'): void {
    if (!event || event === 'page') {
      if (this._pageHandlerId) {
        this._bridge.unregisterHandler(this._pageHandlerId);
        this._pageHandlerId = undefined;
      }
    }
    if (!event || event === 'popup') {
      if (this._popupHandlerId) {
        this._bridge.unregisterHandler(this._popupHandlerId);
        this._popupHandlerId = undefined;
      }
    }
    this._bridge.call('browser.removeAllListeners', [event]);
  }

  close(): void {
    this._bridge.tryQuit();
  }
}

export const browser = {
  launch(options: LaunchOptions = {}): BrowserSync {
    const bridge = SyncBridge.create();
    bridge.call('browser.launch', [options]);
    return new BrowserSync(bridge);
  },
};
