import * as fs from 'fs';
import * as nodePath from 'path';
import { SyncBridge } from './bridge';
import { ElementSync } from './element';
import { KeyboardSync, MouseSync, TouchSync } from './keyboard';
import { ClockSync } from './clock';
import { BrowserContextSync } from './context';
import { RouteSync, RouteRequest } from './route';
import { DialogSync, DialogData } from './dialog';
import { ElementInfo, SelectorOptions } from '../element';
import { A11yNode, ScreenshotOptions, FindOptions } from '../page';

const customInspect = Symbol.for('nodejs.util.inspect.custom');

export interface RequestData {
  url: string;
  method: string;
  headers: Record<string, string>;
  postData: string | null;
}

export interface ResponseData {
  url: string;
  status: number;
  headers: Record<string, string>;
  body: string | null;
}

export class DownloadData {
  readonly url: string;
  readonly suggestedFilename: string;
  readonly path: string | null;

  constructor(data: { url: string; suggestedFilename: string; path: string | null }) {
    this.url = data.url;
    this.suggestedFilename = data.suggestedFilename;
    this.path = data.path;
  }

  /** Save the downloaded file to a destination path. */
  saveAs(destPath: string): void {
    if (!this.path) {
      throw new Error('Download failed or path not available');
    }
    fs.mkdirSync(nodePath.dirname(destPath), { recursive: true });
    fs.copyFileSync(this.path, destPath);
  }
}

type MessageHandler = (data: string, info: { direction: 'sent' | 'received' }) => void;
type CloseHandler = (code?: number, reason?: string) => void;

export class WebSocketInfoSync {
  private _url: string;
  private _isClosed = false;
  private _messageHandlers: MessageHandler[] = [];
  private _closeHandlers: CloseHandler[] = [];

  constructor(url: string) {
    this._url = url;
  }

  url(): string {
    return this._url;
  }

  onMessage(fn: MessageHandler): void {
    this._messageHandlers.push(fn);
  }

  onClose(fn: CloseHandler): void {
    this._closeHandlers.push(fn);
  }

  isClosed(): boolean {
    return this._isClosed;
  }

  /** @internal */
  _emitMessage(data: string, direction: 'sent' | 'received'): void {
    for (const fn of this._messageHandlers) fn(data, { direction });
  }

  /** @internal */
  _emitClose(code?: number, reason?: string): void {
    this._isClosed = true;
    for (const fn of this._closeHandlers) fn(code, reason);
  }
}

export class PageSync {
  /** @internal */
  readonly _bridge: SyncBridge;
  /** @internal */
  readonly _pageId: number;

  readonly keyboard: KeyboardSync;
  readonly mouse: MouseSync;
  readonly touch: TouchSync;
  readonly clock: ClockSync;

  private _nextHandlerId = 0;
  private _routeHandlerIds = new Map<string, string>(); // pattern → handlerId
  private _dialogHandlerId: string | null = null;
  private _requestHandlerId: string | null = null;
  private _responseHandlerId: string | null = null;
  private _downloadHandlerId: string | null = null;
  private _wsHandlerId: string | null = null;
  private _wsInstances = new Map<number, WebSocketInfoSync>();
  private _cachedContext: BrowserContextSync | null = null;

  constructor(bridge: SyncBridge, pageId: number) {
    this._bridge = bridge;
    this._pageId = pageId;
    this.keyboard = new KeyboardSync(bridge, pageId);
    this.mouse = new MouseSync(bridge, pageId);
    this.touch = new TouchSync(bridge, pageId);
    this.clock = new ClockSync(bridge, pageId);

    // Initialize waitUntil namespace
    this.waitUntil = Object.assign(
      (fn: string, options?: { timeout?: number }) => {
        const result = bridge.call<{ value: unknown }>('page.waitForFunction', [pageId, fn, options]);
        return result.value;
      },
      {
        url: (pattern: string, options?: { timeout?: number }) => {
          bridge.call('page.waitForURL', [pageId, pattern, options]);
        },
        loaded: (state?: string, options?: { timeout?: number }) => {
          bridge.call('page.waitForLoad', [pageId, state, options]);
        },
      }
    );
  }

  [customInspect](): string {
    try {
      const u = this.url();
      const t = this.title();
      return `Page { url: '${u}', title: '${t}' }`;
    } catch {
      return `Page { id: ${this._pageId} }`;
    }
  }

  /** The parent BrowserContext that owns this page. */
  get context(): BrowserContextSync {
    if (!this._cachedContext) {
      const result = this._bridge.call<{ contextId: number }>('page.context', [this._pageId]);
      this._cachedContext = new BrowserContextSync(this._bridge, result.contextId);
    }
    return this._cachedContext;
  }

  // --- Navigation ---

  go(url: string): void {
    this._bridge.call('page.go', [this._pageId, url]);
  }

  back(): void {
    this._bridge.call('page.back', [this._pageId]);
  }

  forward(): void {
    this._bridge.call('page.forward', [this._pageId]);
  }

  reload(): void {
    this._bridge.call('page.reload', [this._pageId]);
  }

  // --- Info ---

  url(): string {
    const result = this._bridge.call<{ url: string }>('page.url', [this._pageId]);
    return result.url;
  }

  title(): string {
    const result = this._bridge.call<{ title: string }>('page.title', [this._pageId]);
    return result.title;
  }

  content(): string {
    const result = this._bridge.call<{ content: string }>('page.content', [this._pageId]);
    return result.content;
  }

  // --- Finding ---

  find(selector: string | SelectorOptions, options?: FindOptions): ElementSync {
    const result = this._bridge.call<{ elementId: number; info: ElementInfo }>('page.find', [this._pageId, selector, options]);
    return new ElementSync(this._bridge, result.elementId, result.info);
  }

  findAll(selector: string | SelectorOptions, options?: FindOptions): ElementSync[] {
    const result = this._bridge.call<{ elements: { elementId: number; info: ElementInfo }[] }>('page.findAll', [this._pageId, selector, options]);
    return result.elements.map(e => new ElementSync(this._bridge, e.elementId, e.info));
  }

  // --- Waiting ---

  /** Capture namespace — set up a listener before performing an action. */
  get capture(): {
    response(pattern: string, fn?: () => void, options?: { timeout?: number }): { url: string; status: number; headers: Record<string, string>; body: string | null };
    request(pattern: string, fn?: () => void, options?: { timeout?: number }): { url: string; method: string; headers: Record<string, string>; postData: string | null };
    navigation(fn?: () => void, options?: { timeout?: number }): { url: string };
    download(fn?: () => void, options?: { timeout?: number }): DownloadData;
    dialog(fn?: () => void, options?: { timeout?: number }): { type: string; message: string; defaultValue: string };
    event(name: string, fn?: () => void, options?: { timeout?: number }): unknown;
  } {
    const bridge = this._bridge;
    const pageId = this._pageId;
    return {
      response(pattern: string, fn?: () => void, options?: { timeout?: number }) {
        if (fn) {
          bridge.call('page.captureResponseStart', [pageId, pattern, options]);
          fn();
          return bridge.call('page.captureResponseFinish', [pageId]);
        }
        return bridge.call('page.waitForResponse', [pageId, pattern, options]);
      },
      request(pattern: string, fn?: () => void, options?: { timeout?: number }) {
        if (fn) {
          bridge.call('page.captureRequestStart', [pageId, pattern, options]);
          fn();
          return bridge.call('page.captureRequestFinish', [pageId]);
        }
        return bridge.call('page.waitForRequest', [pageId, pattern, options]);
      },
      navigation(fn?: () => void, options?: { timeout?: number }) {
        bridge.call('page.captureNavigationStart', [pageId, options]);
        if (fn) fn();
        return bridge.call('page.captureNavigationFinish', [pageId]);
      },
      download(fn?: () => void, options?: { timeout?: number }) {
        bridge.call('page.captureDownloadStart', [pageId, options]);
        if (fn) fn();
        const raw = bridge.call<{ url: string; suggestedFilename: string; path: string | null }>('page.captureDownloadFinish', [pageId]);
        return new DownloadData(raw);
      },
      dialog(fn?: () => void, options?: { timeout?: number }) {
        bridge.call('page.captureDialogStart', [pageId, options]);
        if (fn) fn();
        return bridge.call('page.captureDialogFinish', [pageId]);
      },
      event(name: string, fn?: () => void, options?: { timeout?: number }) {
        bridge.call('page.captureEventStart', [pageId, name, options]);
        if (fn) fn();
        return bridge.call('page.captureEventFinish', [pageId]);
      },
    };
  }

  /** Wait until a condition is met. Callable with a function, or use .url() / .loaded() sub-methods. */
  readonly waitUntil: ((fn: string, options?: { timeout?: number }) => unknown) & {
    url(pattern: string, options?: { timeout?: number }): void;
    loaded(state?: string, options?: { timeout?: number }): void;
  };

  wait(ms: number): void {
    this._bridge.call('page.wait', [this._pageId, ms]);
  }

  // --- Screenshots & PDF ---

  screenshot(options?: ScreenshotOptions): Buffer {
    const result = this._bridge.call<{ data: string }>('page.screenshot', [this._pageId, options]);
    return Buffer.from(result.data, 'base64');
  }

  pdf(): Buffer {
    const result = this._bridge.call<{ data: string }>('page.pdf', [this._pageId]);
    return Buffer.from(result.data, 'base64');
  }

  // --- Evaluation ---

  evaluate<T = unknown>(expression: string): T {
    const result = this._bridge.call<{ value: T }>('page.eval', [this._pageId, expression]);
    return result.value;
  }

  addScript(source: string): void {
    this._bridge.call('page.addScript', [this._pageId, source]);
  }

  addStyle(source: string): void {
    this._bridge.call('page.addStyle', [this._pageId, source]);
  }

  expose(name: string, fn: string): void {
    this._bridge.call('page.expose', [this._pageId, name, fn]);
  }

  // --- Lifecycle ---

  bringToFront(): void {
    this._bridge.call('page.bringToFront', [this._pageId]);
  }

  close(): void {
    this._bridge.call('page.close', [this._pageId]);
  }

  scroll(direction?: string, amount?: number, selector?: string): void {
    this._bridge.call('page.scroll', [this._pageId, direction, amount, selector]);
  }

  // --- Emulation ---

  setViewport(size: { width: number; height: number }): void {
    this._bridge.call('page.setViewport', [this._pageId, size]);
  }

  viewport(): { width: number; height: number } {
    return this._bridge.call<{ width: number; height: number }>('page.viewport', [this._pageId]);
  }

  emulateMedia(opts: {
    media?: 'screen' | 'print' | null;
    colorScheme?: 'light' | 'dark' | 'no-preference' | null;
    reducedMotion?: 'reduce' | 'no-preference' | null;
    forcedColors?: 'active' | 'none' | null;
    contrast?: 'more' | 'no-preference' | null;
  }): void {
    this._bridge.call('page.emulateMedia', [this._pageId, opts]);
  }

  setContent(html: string): void {
    this._bridge.call('page.setContent', [this._pageId, html]);
  }

  setGeolocation(coords: { latitude: number; longitude: number; accuracy?: number }): void {
    this._bridge.call('page.setGeolocation', [this._pageId, coords]);
  }

  setWindow(options: {
    width?: number;
    height?: number;
    x?: number;
    y?: number;
    state?: 'normal' | 'maximized' | 'minimized' | 'fullscreen';
  }): void {
    this._bridge.call('page.setWindow', [this._pageId, options]);
  }

  window(): { state: string; width: number; height: number; x: number; y: number } {
    return this._bridge.call('page.window', [this._pageId]);
  }

  // --- Frames ---

  frames(): PageSync[] {
    const result = this._bridge.call<{ frameIds: number[] }>('page.frames', [this._pageId]);
    return result.frameIds.map(id => new PageSync(this._bridge, id));
  }

  frame(nameOrUrl: string): PageSync | null {
    const result = this._bridge.call<{ frameId: number | null }>('page.frame', [this._pageId, nameOrUrl]);
    if (result.frameId === null) return null;
    return new PageSync(this._bridge, result.frameId);
  }

  mainFrame(): PageSync {
    return this;
  }

  // --- Accessibility ---

  a11yTree(options?: { everything?: boolean; root?: string }): A11yNode {
    const result = this._bridge.call<{ tree: A11yNode }>('page.a11yTree', [this._pageId, options]);
    return result.tree;
  }

  // --- Network ---

  route(pattern: string, action: 'continue' | 'abort' | { status?: number; body?: string; headers?: Record<string, string> } | ((route: RouteSync) => void)): void {
    if (typeof action === 'function') {
      const handlerId = `route_${this._pageId}_${this._nextHandlerId++}`;
      this._bridge.registerHandler(handlerId, (data: RouteRequest) => {
        const route = new RouteSync(data);
        action(route);
        return route._decision;
      });
      this._routeHandlerIds.set(pattern, handlerId);
      this._bridge.call('page.routeWithCallback', [this._pageId, pattern, handlerId]);
    } else {
      this._bridge.call('page.route', [this._pageId, pattern, action]);
    }
  }

  unroute(pattern: string): void {
    const handlerId = this._routeHandlerIds.get(pattern);
    if (handlerId) {
      this._bridge.unregisterHandler(handlerId);
      this._routeHandlerIds.delete(pattern);
    }
    this._bridge.call('page.unroute', [this._pageId, pattern]);
  }

  setHeaders(headers: Record<string, string>): void {
    this._bridge.call('page.setHeaders', [this._pageId, headers]);
  }

  // --- Events ---

  onDialog(action: 'accept' | 'dismiss' | ((dialog: DialogSync) => void)): void {
    if (typeof action === 'function') {
      const handlerId = `dialog_${this._pageId}_${this._nextHandlerId++}`;
      this._bridge.registerHandler(handlerId, (data: DialogData) => {
        const dialog = new DialogSync(data);
        action(dialog);
        return dialog._decision;
      });
      if (this._dialogHandlerId) {
        this._bridge.unregisterHandler(this._dialogHandlerId);
      }
      this._dialogHandlerId = handlerId;
      this._bridge.call('page.onDialogWithCallback', [this._pageId, handlerId]);
    } else {
      this._bridge.call('page.onDialog', [this._pageId, action]);
    }
  }

  onConsole(mode: 'collect'): void {
    this._bridge.call('page.onConsole', [this._pageId, mode]);
  }

  consoleMessages(): { type: string; text: string }[] {
    const result = this._bridge.call<{ messages: { type: string; text: string }[] }>('page.consoleMessages', [this._pageId]);
    return result.messages;
  }

  onError(mode: 'collect'): void {
    this._bridge.call('page.onError', [this._pageId, mode]);
  }

  errors(): { message: string }[] {
    const result = this._bridge.call<{ errors: { message: string }[] }>('page.errors', [this._pageId]);
    return result.errors;
  }

  onRequest(fn: (req: RequestData) => void): void {
    const handlerId = `request_${this._pageId}_${this._nextHandlerId++}`;
    this._bridge.registerHandler(handlerId, (data: RequestData) => {
      fn(data);
      return null;
    });
    if (this._requestHandlerId) {
      this._bridge.unregisterHandler(this._requestHandlerId);
    }
    this._requestHandlerId = handlerId;
    this._bridge.call('page.onRequestWithCallback', [this._pageId, handlerId]);
  }

  onResponse(fn: (resp: ResponseData) => void): void {
    const handlerId = `response_${this._pageId}_${this._nextHandlerId++}`;
    this._bridge.registerHandler(handlerId, (data: ResponseData) => {
      fn(data);
      return null;
    });
    if (this._responseHandlerId) {
      this._bridge.unregisterHandler(this._responseHandlerId);
    }
    this._responseHandlerId = handlerId;
    this._bridge.call('page.onResponseWithCallback', [this._pageId, handlerId]);
  }

  onDownload(fn: (dl: DownloadData) => void): void {
    const handlerId = `download_${this._pageId}_${this._nextHandlerId++}`;
    this._bridge.registerHandler(handlerId, (data: { url: string; suggestedFilename: string; path: string | null }) => {
      fn(new DownloadData(data));
      return null;
    });
    if (this._downloadHandlerId) {
      this._bridge.unregisterHandler(this._downloadHandlerId);
    }
    this._downloadHandlerId = handlerId;
    this._bridge.call('page.onDownloadWithCallback', [this._pageId, handlerId]);
  }

  onWebSocket(fn: (ws: WebSocketInfoSync) => void): void {
    const handlerId = `ws_${this._pageId}_${this._nextHandlerId++}`;
    this._bridge.registerHandler(handlerId, (data: { type: string; wsId: number; url?: string; data?: string; direction?: 'sent' | 'received'; code?: number; reason?: string }) => {
      if (data.type === 'created') {
        const ws = new WebSocketInfoSync(data.url!);
        this._wsInstances.set(data.wsId, ws);
        fn(ws);
      } else if (data.type === 'message') {
        const ws = this._wsInstances.get(data.wsId);
        if (ws) ws._emitMessage(data.data!, data.direction!);
      } else if (data.type === 'close') {
        const ws = this._wsInstances.get(data.wsId);
        if (ws) {
          ws._emitClose(data.code, data.reason);
          this._wsInstances.delete(data.wsId);
        }
      }
      return null;
    });
    if (this._wsHandlerId) {
      this._bridge.unregisterHandler(this._wsHandlerId);
    }
    this._wsHandlerId = handlerId;
    this._bridge.call('page.onWebSocketWithCallback', [this._pageId, handlerId]);
  }

  removeAllListeners(event?: 'request' | 'response' | 'dialog' | 'console' | 'error' | 'download' | 'websocket'): void {
    // Clean up callback handlers
    if (!event || event === 'dialog') {
      if (this._dialogHandlerId) {
        this._bridge.unregisterHandler(this._dialogHandlerId);
        this._dialogHandlerId = null;
      }
    }
    if (!event || event === 'request') {
      if (this._requestHandlerId) {
        this._bridge.unregisterHandler(this._requestHandlerId);
        this._requestHandlerId = null;
      }
      for (const [, handlerId] of this._routeHandlerIds) {
        this._bridge.unregisterHandler(handlerId);
      }
      this._routeHandlerIds.clear();
    }
    if (!event || event === 'response') {
      if (this._responseHandlerId) {
        this._bridge.unregisterHandler(this._responseHandlerId);
        this._responseHandlerId = null;
      }
    }
    if (!event || event === 'download') {
      if (this._downloadHandlerId) {
        this._bridge.unregisterHandler(this._downloadHandlerId);
        this._downloadHandlerId = null;
      }
    }
    if (!event || event === 'websocket') {
      if (this._wsHandlerId) {
        this._bridge.unregisterHandler(this._wsHandlerId);
        this._wsHandlerId = null;
      }
      this._wsInstances.clear();
    }
    this._bridge.call('page.removeAllListeners', [this._pageId, event]);
  }
}
