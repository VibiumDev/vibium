import { BiDiClient, BiDiEvent, ScreenshotResult } from './bidi';
import { Element, ElementInfo, SelectorOptions, FluentElement, fluent } from './element';
import { BrowserContext } from './context';
import { Route } from './route';
import { Request, Response } from './network';
import { Dialog } from './dialog';
import { ConsoleMessage } from './console';
import { Download } from './download';
import { WebSocketInfo } from './websocket';
import { Clock } from './clock';
import { debug } from './utils/debug';

export interface FindOptions {
  /** Timeout in milliseconds to wait for element. Default: 30000 */
  timeout?: number;
}

export interface ScreenshotOptions {
  /** Capture full scrollable page instead of just the viewport. */
  fullPage?: boolean;
  /** Capture a specific region of the page. */
  clip?: { x: number; y: number; width: number; height: number };
}

/** Options for pdf(). Unset options keep the browser's print defaults. */
export interface PdfOptions {
  /** Landscape orientation (default: portrait). */
  landscape?: boolean;
  /** Print scale, 0.1-2 (default: 1). */
  scale?: number;
  /** Print background graphics (default: false). */
  background?: boolean;
  /** Top margin in cm (default: 1). */
  marginTop?: number;
  /** Bottom margin in cm (default: 1). */
  marginBottom?: number;
  /** Left margin in cm (default: 1). */
  marginLeft?: number;
  /** Right margin in cm (default: 1). */
  marginRight?: number;
  /** Page width in cm (default: 21.59). */
  pageWidth?: number;
  /** Page height in cm (default: 27.94). */
  pageHeight?: number;
  /** Pages to print, e.g. [1, '3-5'] (default: all). */
  pageRanges?: Array<number | string>;
  /** Shrink content to fit the page width (default: true). */
  shrinkToFit?: boolean;
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

interface VibiumFindAllResult {
  elements: Array<{ tag: string; text: string; box: { x: number; y: number; width: number; height: number }; index: number }>;
  count: number;
}

const customInspect = Symbol.for('nodejs.util.inspect.custom');

// Exposed host functions are connection-scoped, not Page-instance-scoped:
// browser.page() hands out a fresh Page object for the same context, and
// every instance sees every event. One registry and one dispatcher per
// client means one execution and one reply per call, whichever instance
// registered the function.
const exposeRegistries = new WeakMap<BiDiClient, Map<string, (...args: unknown[]) => unknown>>();

function exposeRegistry(client: BiDiClient): Map<string, (...args: unknown[]) => unknown> {
  let registry = exposeRegistries.get(client);
  if (!registry) {
    const fns = new Map<string, (...args: unknown[]) => unknown>();
    registry = fns;
    exposeRegistries.set(client, fns);
    client.onEvent((event) => {
      if (event.method !== 'vibium:expose.call') return;
      handleExposeCall(client, fns, event.params as Record<string, unknown>);
    });
  }
  return registry;
}

async function handleExposeCall(
  client: BiDiClient,
  fns: Map<string, (...args: unknown[]) => unknown>,
  params: Record<string, unknown>,
): Promise<void> {
  const name = params.name as string;
  const seq = params.seq as number;
  const context = params.context as string;
  const realm = params.realm as string | undefined;

  // Every outcome answers: an unanswered call leaves the page's promise
  // parked forever. The reply carries the calling realm back, so the engine
  // delivers into the document that made the call, not whatever the context
  // shows after a navigation.
  const reply = (body: { result?: unknown; error?: string }) => {
    client.send('vibium:expose.result', { context, realm, seq, ...body }).catch(() => {});
  };

  const fn = fns.get(name);
  if (!fn) {
    reply({ error: `${name} is not exposed` });
    return;
  }

  try {
    const result = await fn(...((params.args as unknown[]) ?? []));
    // Results cross as JSON; catching the serialization failure here turns
    // it into a page-side rejection instead of a forever-pending promise.
    JSON.stringify(result);
    reply({ result: result === undefined ? null : result });
  } catch (e) {
    reply({ error: e instanceof Error ? e.message : String(e) });
  }
}

/** Page-level keyboard input. */
export class Keyboard {
  private client: BiDiClient;
  private contextId: string;

  constructor(client: BiDiClient, contextId: string) {
    this.client = client;
    this.contextId = contextId;
  }

  /** Press and release a key. Supports combos like "Control+a". */
  async press(key: string): Promise<void> {
    await this.client.send('vibium:keyboard.press', {
      context: this.contextId,
      key,
    });
  }

  /** Press a key down (without releasing). */
  async down(key: string): Promise<void> {
    await this.client.send('vibium:keyboard.down', {
      context: this.contextId,
      key,
    });
  }

  /** Release a key. */
  async up(key: string): Promise<void> {
    await this.client.send('vibium:keyboard.up', {
      context: this.contextId,
      key,
    });
  }

  /** Type a string of text character by character. */
  async type(text: string): Promise<void> {
    await this.client.send('vibium:keyboard.type', {
      context: this.contextId,
      text,
    });
  }
}

/** Page-level mouse input. */
export class Mouse {
  private client: BiDiClient;
  private contextId: string;

  constructor(client: BiDiClient, contextId: string) {
    this.client = client;
    this.contextId = contextId;
  }

  /** Click at (x, y) coordinates. */
  async click(x: number, y: number): Promise<void> {
    await this.client.send('vibium:mouse.click', {
      context: this.contextId,
      x,
      y,
    });
  }

  /** Move mouse to (x, y) coordinates. */
  async move(x: number, y: number): Promise<void> {
    await this.client.send('vibium:mouse.move', {
      context: this.contextId,
      x,
      y,
    });
  }

  /** Press mouse button down. */
  async down(): Promise<void> {
    await this.client.send('vibium:mouse.down', {
      context: this.contextId,
    });
  }

  /** Release mouse button. */
  async up(): Promise<void> {
    await this.client.send('vibium:mouse.up', {
      context: this.contextId,
    });
  }

  /** Scroll the mouse wheel. */
  async wheel(deltaX: number, deltaY: number): Promise<void> {
    await this.client.send('vibium:mouse.wheel', {
      context: this.contextId,
      x: 0,
      y: 0,
      deltaX,
      deltaY,
    });
  }
}

/** Page-level touch input. */
export class Touch {
  private client: BiDiClient;
  private contextId: string;

  constructor(client: BiDiClient, contextId: string) {
    this.client = client;
    this.contextId = contextId;
  }

  /** Tap at (x, y) coordinates. */
  async tap(x: number, y: number): Promise<void> {
    await this.client.send('vibium:touch.tap', {
      context: this.contextId,
      x,
      y,
    });
  }
}

export interface A11yNode {
  role: string;
  name?: string;
  value?: string | number;
  description?: string;
  disabled?: boolean;
  expanded?: boolean;
  focused?: boolean;
  checked?: boolean | 'mixed';
  pressed?: boolean | 'mixed';
  selected?: boolean;
  required?: boolean;
  readonly?: boolean;
  level?: number;
  valuemin?: number;
  valuemax?: number;
  children?: A11yNode[];
}

export class Page {
  private client: BiDiClient;
  private contextId: string;
  private _context: BrowserContext;

  /** Page-level keyboard input. */
  readonly keyboard: Keyboard;
  /** Page-level mouse input. */
  readonly mouse: Mouse;
  /** Page-level touch input. */
  readonly touch: Touch;
  /** Page-level clock control for faking timers and Date. */
  readonly clock: Clock;

  // Network interception state
  private routes: { pattern: string; handler: (route: Route) => void; interceptId?: string }[] = [];
  private requestCallbacks: ((request: Request) => void)[] = [];
  private responseCallbacks: ((response: Response) => void)[] = [];
  private dialogCallbacks: ((dialog: Dialog) => void)[] = [];
  private dialogPolicyManual = false;
  private consoleCallbacks: ((msg: ConsoleMessage) => void)[] = [];
  private errorCallbacks: ((error: Error) => void)[] = [];
  private downloadCallbacks: ((download: Download) => void)[] = [];
  private navigationCallbacks: ((url: string) => void)[] = [];
  private wsCallbacks: ((ws: WebSocketInfo) => void)[] = [];
  private wsConnections: Map<number, WebSocketInfo> = new Map();
  private wsSetup: Promise<unknown> | null = null;
  private eventHandler: ((event: BiDiEvent) => void) | null = null;
  private interceptId: string | null = null;
  private dataCollectorId: string | null = null;

  // Console/error collect-mode buffers (null = not collecting)
  private _consoleBuffer: { type: string; text: string }[] | null = null;
  private _errorBuffer: { message: string }[] | null = null;

  constructor(client: BiDiClient, contextId: string, userContextId: string = 'default') {
    this.client = client;
    this.contextId = contextId;
    this._context = new BrowserContext(client, userContextId);
    this.keyboard = new Keyboard(client, contextId);
    this.mouse = new Mouse(client, contextId);
    this.touch = new Touch(client, contextId);
    this.clock = new Clock(client, contextId);

    // Initialize capture namespace
    const self = this;
    this.capture = {
      response(pattern: string, fn?: () => Promise<void>, options?: { timeout?: number }): Promise<Response> {
        const promise = self._captureResponse(pattern, options);
        if (fn) return Promise.all([promise, fn()]).then(([v]) => v);
        return promise;
      },
      request(pattern: string, fn?: () => Promise<void>, options?: { timeout?: number }): Promise<Request> {
        const promise = self._captureRequest(pattern, options);
        if (fn) return Promise.all([promise, fn()]).then(([v]) => v);
        return promise;
      },
      navigation(fn?: () => Promise<void>, options?: { timeout?: number }): Promise<string> {
        const promise = self._captureNavigation(options);
        if (fn) return Promise.all([promise, fn()]).then(([v]) => v);
        return promise;
      },
      download(fn?: () => Promise<void>, options?: { timeout?: number }): Promise<Download> {
        const promise = self._captureDownload(options);
        if (fn) return Promise.all([promise, fn()]).then(([v]) => v);
        return promise;
      },
      dialog(fn?: () => Promise<void>, options?: { timeout?: number }): Promise<Dialog> {
        const promise = self._captureDialog(options);
        if (fn) return Promise.all([promise, fn()]).then(([v]) => v);
        return promise;
      },
      event(name: string, fn?: () => Promise<void>, options?: { timeout?: number }): Promise<unknown> {
        const promise = self._captureEvent(name, options);
        if (fn) return Promise.all([promise, fn()]).then(([v]) => v);
        return promise;
      },
    };

    // Initialize waitUntil namespace
    this.waitUntil = Object.assign(
      (fn: string, options?: { timeout?: number }) => self._waitForFunction(fn, options),
      {
        url: (pattern: string, options?: { timeout?: number }) => self._waitForURL(pattern, options),
        loaded: (state?: string, options?: { timeout?: number }) => self._waitForLoad(state, options),
      }
    );

    // Listen for network and dialog events
    this.eventHandler = (event: BiDiEvent) => {
      const params = event.params as Record<string, unknown>;
      const eventContext = params.context as string | undefined;

      // Filter events to this page's context
      if (eventContext && eventContext !== this.contextId) return;

      if (event.method === 'network.beforeRequestSent') {
        this.handleBeforeRequestSent(params);
      } else if (event.method === 'network.responseCompleted') {
        this.handleResponseCompleted(params);
      } else if (event.method === 'browsingContext.userPromptOpened') {
        this.handleUserPromptOpened(params);
      } else if (event.method === 'browsingContext.downloadWillBegin') {
        this.handleDownloadWillBegin(params);
      } else if (event.method === 'log.entryAdded') {
        // log.entryAdded uses source.context, not params.context
        const source = params.source as { context?: string } | undefined;
        const logContext = source?.context;
        if (logContext && logContext !== this.contextId) return;
        this.handleLogEntryAdded(params);
      } else if (event.method === 'browsingContext.load') {
        const url = params.url as string | undefined;
        if (url) {
          for (const cb of this.navigationCallbacks) {
            cb(url);
          }
        }
      } else if (event.method === 'browsingContext.fragmentNavigated'
                 || event.method === 'browsingContext.historyUpdated') {
        const url = params.url as string | undefined;
        if (url) {
          for (const cb of this.navigationCallbacks) {
            cb(url);
          }
        }
      } else if (event.method === 'vibium:ws.created') {
        this.handleWsCreated(params);
      } else if (event.method === 'vibium:ws.message') {
        this.handleWsMessage(params);
      } else if (event.method === 'vibium:ws.closed') {
        this.handleWsClosed(params);
      }
    };
    this.client.onEvent(this.eventHandler);
  }

  /** The browsing context ID for this page. */
  get id(): string {
    return this.contextId;
  }

  [customInspect](): string {
    return `Page { contextId: '${this.contextId}' }`;
  }

  /** The parent BrowserContext that owns this page. */
  get context(): BrowserContext {
    return this._context;
  }

  /** Navigate to a URL. */
  async go(url: string): Promise<void> {
    debug('page.go', { url, context: this.contextId });
    await this.client.send('vibium:page.navigate', {
      context: this.contextId,
      url,
    });
  }

  /** Navigate back in history. */
  async back(): Promise<void> {
    await this.client.send('vibium:page.back', { context: this.contextId });
  }

  /** Navigate forward in history. */
  async forward(): Promise<void> {
    await this.client.send('vibium:page.forward', { context: this.contextId });
  }

  /** Reload the page. */
  async reload(): Promise<void> {
    await this.client.send('vibium:page.reload', { context: this.contextId });
  }

  /** Get the current page URL. */
  async url(): Promise<string> {
    const result = await this.client.send<{ url: string }>('vibium:page.url', {
      context: this.contextId,
    });
    return result.url;
  }

  /** Get the current page title. */
  async title(): Promise<string> {
    const result = await this.client.send<{ title: string }>('vibium:page.title', {
      context: this.contextId,
    });
    return result.title;
  }

  /** Get the full HTML content of the page. */
  async content(): Promise<string> {
    const result = await this.client.send<{ content: string }>('vibium:page.content', {
      context: this.contextId,
    });
    return result.content;
  }

  /** @internal Wait until the page URL matches a pattern. */
  async _waitForURL(pattern: string, options?: { timeout?: number }): Promise<void> {
    await this.client.send('vibium:page.waitForURL', {
      context: this.contextId,
      pattern,
      timeout: options?.timeout,
    });
  }

  /** @internal Wait until the page reaches a load state. */
  async _waitForLoad(state?: string, options?: { timeout?: number }): Promise<void> {
    await this.client.send('vibium:page.waitForLoad', {
      context: this.contextId,
      state,
      timeout: options?.timeout,
    });
  }

  // --- Frames ---

  /** Get all child frames of this page (recursive, flattened). */
  async frames(): Promise<Page[]> {
    const result = await this.client.send<{ frames: { context: string; url: string; name: string }[] }>('vibium:page.frames', {
      context: this.contextId,
    });
    return result.frames.map(f => new Page(this.client, f.context));
  }

  /** Find a frame by name attribute or URL substring. Returns null if not found. */
  async frame(nameOrUrl: string): Promise<Page | null> {
    const result = await this.client.send<{ context: string; url: string; name: string } | null>('vibium:page.frame', {
      context: this.contextId,
      nameOrUrl,
    });
    if (!result || !result.context) return null;
    return new Page(this.client, result.context);
  }

  /** Returns this page — the page IS its own main frame. */
  mainFrame(): Page {
    return this;
  }

  // --- Emulation ---

  /** Set the viewport size. */
  async setViewport(size: { width: number; height: number }): Promise<void> {
    await this.client.send('vibium:page.setViewport', {
      context: this.contextId,
      width: size.width,
      height: size.height,
    });
  }

  /** Get the current viewport size. */
  async viewport(): Promise<{ width: number; height: number }> {
    return await this.client.send<{ width: number; height: number }>('vibium:page.viewport', {
      context: this.contextId,
    });
  }

  /** Override CSS media features (colorScheme, reducedMotion, forcedColors, contrast, media type). */
  async emulateMedia(opts: {
    media?: 'screen' | 'print' | null;
    colorScheme?: 'light' | 'dark' | 'no-preference' | null;
    reducedMotion?: 'reduce' | 'no-preference' | null;
    forcedColors?: 'active' | 'none' | null;
    contrast?: 'more' | 'no-preference' | null;
  }): Promise<void> {
    await this.client.send('vibium:page.emulateMedia', {
      context: this.contextId,
      ...opts,
    });
  }

  /** Replace the page HTML content. */
  async setContent(html: string): Promise<void> {
    await this.client.send('vibium:page.setContent', {
      context: this.contextId,
      html,
    });
  }

  /** Override the browser's geolocation. */
  async setGeolocation(coords: { latitude: number; longitude: number; accuracy?: number }): Promise<void> {
    await this.client.send('vibium:page.setGeolocation', {
      context: this.contextId,
      ...coords,
    });
  }

  /** Set the OS browser window size, position, or state. */
  async setWindow(options: {
    width?: number;
    height?: number;
    x?: number;
    y?: number;
    state?: 'normal' | 'maximized' | 'minimized' | 'fullscreen';
  }): Promise<void> {
    await this.client.send('vibium:page.setWindow', {
      ...options,
    });
  }

  /** Get the current OS browser window state and dimensions. */
  async window(): Promise<{ state: string; width: number; height: number; x: number; y: number }> {
    return await this.client.send<{ state: string; width: number; height: number; x: number; y: number }>('vibium:page.window', {});
  }

  // --- Accessibility ---

  /** Get the accessibility tree for the page. */
  async a11yTree(options?: { everything?: boolean; root?: string }): Promise<A11yNode> {
    const result = await this.client.send<{ tree: A11yNode }>('vibium:page.a11yTree', {
      context: this.contextId,
      ...options,
    });
    return result.tree;
  }

  /** Bring this page/tab to the foreground. */
  async bringToFront(): Promise<void> {
    await this.client.send('browsingContext.activate', { context: this.contextId });
  }

  /** Close this page/tab. */
  async close(): Promise<void> {
    if (this.eventHandler) {
      this.client.offEvent(this.eventHandler);
    }
    await this.client.send('browsingContext.close', { context: this.contextId });
  }

  /** Scroll the page in a direction. */
  async scroll(direction?: string, amount?: number, selector?: string): Promise<void> {
    await this.client.send('vibium:page.scroll', {
      context: this.contextId,
      direction,
      amount,
      selector,
    });
  }

  // --- Screenshots & PDF ---

  /** Take a screenshot of the page. Returns a PNG buffer. */
  async screenshot(options?: ScreenshotOptions): Promise<Buffer> {
    const result = await this.client.send<ScreenshotResult>('vibium:page.screenshot', {
      context: this.contextId,
      fullPage: options?.fullPage,
      clip: options?.clip,
    });
    return Buffer.from(result.data, 'base64');
  }

  /** Print the page to PDF. Returns a PDF buffer. Only works in headless mode. */
  async pdf(options?: PdfOptions): Promise<Buffer> {
    const result = await this.client.send<{ data: string }>('vibium:page.pdf', {
      context: this.contextId,
      ...options,
    });
    return Buffer.from(result.data, 'base64');
  }

  // --- Evaluation ---

  /** Evaluate a JS expression and return the deserialized value. */
  async evaluate<T = unknown>(expression: string): Promise<T> {
    const result = await this.client.send<{ value: T }>('vibium:page.eval', {
      context: this.contextId,
      expression,
    });
    return result.value;
  }

  /** Inject a script into the page. Pass a URL or inline JavaScript. */
  async addScript(source: string): Promise<void> {
    const isURL = source.startsWith('http://') || source.startsWith('https://') || source.startsWith('//');
    await this.client.send('vibium:page.addScript', {
      context: this.contextId,
      ...(isURL ? { url: source } : { content: source }),
    });
  }

  /** Inject a stylesheet into the page. Pass a URL or inline CSS. */
  async addStyle(source: string): Promise<void> {
    const isURL = source.startsWith('http://') || source.startsWith('https://') || source.startsWith('//');
    await this.client.send('vibium:page.addStyle', {
      context: this.contextId,
      ...(isURL ? { url: source } : { content: source }),
    });
  }

  /**
   * Expose a function on window.
   *
   * Pass a function to expose a host callback: the page calls
   * window[name](...args), this function runs here, and its return value
   * resolves the page's promise. Arguments and results cross as JSON.
   * Pass a string to inject it as JS source instead, defining window[name]
   * inside the page.
   *
   * Either form survives navigation, and re-exposing a name replaces it.
   */
  async expose(name: string, fn: string | ((...args: unknown[]) => unknown)): Promise<void> {
    if (typeof fn === 'function') {
      exposeRegistry(this.client).set(name, fn);
      await this.client.send('vibium:page.exposeFunction', {
        context: this.contextId,
        name,
      });
      return;
    }
    exposeRegistries.get(this.client)?.delete(name);
    await this.client.send('vibium:page.expose', {
      context: this.contextId,
      name,
      fn,
    });
  }

  // --- Page-level Waiting ---

  /** Capture namespace — set up a listener before performing an action. */
  readonly capture: {
    /** Capture a response matching a URL pattern. Optionally pass fn to trigger the action. */
    response(pattern: string, fn?: () => Promise<void>, options?: { timeout?: number }): Promise<Response>;
    /** Capture a request matching a URL pattern. Optionally pass fn to trigger the action. */
    request(pattern: string, fn?: () => Promise<void>, options?: { timeout?: number }): Promise<Request>;
    /** Capture a navigation event. Optionally pass fn to trigger the action. Resolves with URL. */
    navigation(fn?: () => Promise<void>, options?: { timeout?: number }): Promise<string>;
    /** Capture a download. Optionally pass fn to trigger the action. */
    download(fn?: () => Promise<void>, options?: { timeout?: number }): Promise<Download>;
    /** Capture a dialog. Optionally pass fn to trigger the action. */
    dialog(fn?: () => Promise<void>, options?: { timeout?: number }): Promise<Dialog>;
    /** Capture a named event. Optionally pass fn to trigger the action. */
    event(name: string, fn?: () => Promise<void>, options?: { timeout?: number }): Promise<unknown>;
  };

  /**
   * Wait until a condition is met. Callable with a function, or use .url() / .loaded() sub-methods.
   * @deprecated Use waitForFunction(), waitForURL(), or waitForLoad().
   */
  readonly waitUntil: ((fn: string, options?: { timeout?: number }) => Promise<unknown>) & {
    /** @deprecated Use waitForURL(). */
    url(pattern: string, options?: { timeout?: number }): Promise<void>;
    /** @deprecated Use waitForLoad(). */
    loaded(state?: string, options?: { timeout?: number }): Promise<void>;
  };

  /** Wait until a function returns a truthy value. */
  async waitForFunction(fn: string, options?: { timeout?: number }): Promise<unknown> {
    return this._waitForFunction(fn, options);
  }

  /** Wait until the page URL matches a pattern. */
  async waitForURL(pattern: string, options?: { timeout?: number }): Promise<void> {
    await this._waitForURL(pattern, options);
  }

  /** Wait until the page reaches a load state. */
  async waitForLoad(state?: string, options?: { timeout?: number }): Promise<void> {
    await this._waitForLoad(state, options);
  }

  /** Wait for a fixed amount of time (milliseconds). Discouraged but useful for debugging. */
  async wait(ms: number): Promise<void> {
    await this.client.send('vibium:page.wait', {
      context: this.contextId,
      ms,
    });
  }

  /** @internal Wait until a function returns a truthy value. */
  async _waitForFunction<T = unknown>(fn: string, options?: { timeout?: number }): Promise<T> {
    const result = await this.client.send<{ value: T }>('vibium:page.waitForFunction', {
      context: this.contextId,
      fn,
      timeout: options?.timeout,
    });
    return result.value;
  }

  /** Find an element by CSS selector or semantic options. Waits for element to exist. */
  find(selector: string | SelectorOptions, options?: FindOptions): FluentElement {
    const promise = (async () => {
      const params: Record<string, unknown> = {
        context: this.contextId,
        timeout: options?.timeout,
      };

      if (typeof selector === 'string') {
        debug('page.find', { selector, timeout: options?.timeout });
        params.selector = selector;
      } else {
        debug('page.find', { ...selector, timeout: options?.timeout });
        Object.assign(params, selector);
        if (selector.timeout && !options?.timeout) params.timeout = selector.timeout;
      }

      const result = await this.client.send<VibiumFindResult>('vibium:page.find', params);

      const info: ElementInfo = {
        tag: result.tag,
        text: result.text,
        box: result.box,
      };

      const selectorStr = typeof selector === 'string' ? selector : '';
      const selectorParams = typeof selector === 'string' ? { selector } : { ...selector };
      return new Element(this.client, this.contextId, selectorStr, info, undefined, selectorParams);
    })();
    return fluent(promise);
  }

  /**
   * Find all elements matching a CSS selector or semantic options. Waits up
   * to the timeout for at least one match, then returns an empty array if
   * there is none. A timeout of 0 checks once without waiting.
   *
   * Each element carries a snapshot of its tag, text, and box taken at
   * findAll time, readable via `el.info` with no further round trips:
   * els.map(el => el.info.text). Live reads like el.text() re-resolve the
   * element and fail if the page has changed since findAll.
   */
  async findAll(selector: string | SelectorOptions, options?: FindOptions): Promise<Element[]> {
    const params: Record<string, unknown> = {
      context: this.contextId,
      timeout: options?.timeout,
    };

    if (typeof selector === 'string') {
      debug('page.findAll', { selector, timeout: options?.timeout });
      params.selector = selector;
    } else {
      debug('page.findAll', { ...selector, timeout: options?.timeout });
      Object.assign(params, selector);
      if (selector.timeout && !options?.timeout) params.timeout = selector.timeout;
    }

    const result = await this.client.send<VibiumFindAllResult>('vibium:page.findAll', params);

    const selectorStr = typeof selector === 'string' ? selector : '';
    const selectorParams = typeof selector === 'string' ? { selector } : { ...selector };
    return result.elements.map((el) => {
      const info: ElementInfo = { tag: el.tag, text: el.text, box: el.box };
      return new Element(this.client, this.contextId, selectorStr, info, el.index, selectorParams);
    });
  }

  // --- Network Interception ---

  /**
   * Intercept network requests matching a URL pattern.
   * The handler receives a Route object that can fulfill, continue, or abort the request.
   */
  async route(pattern: string, handler: (route: Route) => void): Promise<void> {
    // The binary compiles the pattern, owns the intercept lifecycle, and
    // annotates blocked request events with the patterns that matched, so
    // dispatch below never interprets the glob itself.
    const result = await this.client.send<{ intercept: string }>('vibium:page.route', {
      context: this.contextId,
      pattern,
    });
    this.interceptId = result.intercept;

    this.ensureDataCollector();
    this.routes.push({ pattern, handler, interceptId: result.intercept });
  }

  /** Remove a previously registered route. If no handler given, removes all routes for the pattern. */
  async unroute(pattern: string): Promise<void> {
    const removed = this.routes.filter(r => r.pattern === pattern).length;
    this.routes = this.routes.filter(r => r.pattern !== pattern);
    // The binary refcounts pattern registrations and tears the intercept
    // down when the last one goes.
    for (let i = 0; i < removed; i++) {
      await this.client.send('vibium:page.unroute', { context: this.contextId, pattern });
    }
    if (this.routes.length === 0) {
      this.interceptId = null;
    }
  }

  /** Register a callback for every outgoing request. */
  onRequest(fn: (request: Request) => void): void {
    this.ensureDataCollector();
    this.requestCallbacks.push(fn);
  }

  /** Register a callback for every completed response. */
  onResponse(fn: (response: Response) => void): void {
    this.ensureDataCollector();
    this.responseCallbacks.push(fn);
  }

  /**
   * Remove all listeners for a given event, or all events if no event specified.
   * Supported events: 'request', 'response', 'dialog', 'console', 'error', 'download', 'websocket'.
   */
  removeAllListeners(event?: 'request' | 'response' | 'dialog' | 'console' | 'error' | 'download' | 'navigation' | 'websocket'): void {
    if (!event || event === 'request') {
      this.requestCallbacks = [];
    }
    if (!event || event === 'response') {
      this.responseCallbacks = [];
    }
    if (!event || event === 'dialog') {
      this.dialogCallbacks = [];
      this.syncDialogPolicy();
    }
    if (!event || event === 'console') {
      this.consoleCallbacks = [];
      this._consoleBuffer = null;
    }
    if (!event || event === 'error') {
      this.errorCallbacks = [];
      this._errorBuffer = null;
    }
    if (!event || event === 'download') {
      this.downloadCallbacks = [];
    }
    if (!event || event === 'navigation') {
      this.navigationCallbacks = [];
    }
    if (!event || event === 'websocket') {
      this.wsCallbacks = [];
    }
    // Tear down data collector when no request/response listeners and no routes remain
    if (this.requestCallbacks.length === 0 && this.responseCallbacks.length === 0 && this.routes.length === 0) {
      this.teardownDataCollector();
    }
  }

  /**
   * @internal Capture a request matching a URL pattern. The binary matches
   * the pattern and waits for the event; this just awaits the command.
   */
  async _captureRequest(pattern: string, options?: { timeout?: number }): Promise<Request> {
    const result = await this.client.send<{ event: Record<string, unknown> }>('vibium:page.captureRequest', {
      context: this.contextId,
      pattern,
      timeout: options?.timeout ?? 10000,
    });
    return new Request(result.event, this.client);
  }

  /** @internal Capture a response matching a URL pattern. */
  async _captureResponse(pattern: string, options?: { timeout?: number }): Promise<Response> {
    this.ensureDataCollector();
    const result = await this.client.send<{ event: Record<string, unknown> }>('vibium:page.captureResponse', {
      context: this.contextId,
      pattern,
      timeout: options?.timeout ?? 10000,
    });
    return new Response(result.event, this.client);
  }

  /**
   * @internal One-shot capture, waited out in the engine
   * (vibium:page.captureEvent), so no client keeps its own listener and
   * timeout machinery (#446). Returns the raw event params.
   */
  private async captureEventParams(kind: string, options?: { timeout?: number }): Promise<Record<string, unknown>> {
    const result = await this.client.send<{ event: Record<string, unknown> }>('vibium:page.captureEvent', {
      context: this.contextId,
      kind,
      timeout: options?.timeout ?? 10000,
    });
    return result.event;
  }

  /** @internal Capture a navigation event. Resolves with the URL. */
  async _captureNavigation(options?: { timeout?: number }): Promise<string> {
    const params = await this.captureEventParams('navigation', options);
    return (params.url as string) ?? '';
  }

  /** @internal Capture a download event. */
  async _captureDownload(options?: { timeout?: number }): Promise<Download> {
    const params = await this.captureEventParams('download', options);
    return new Download(this.client, params);
  }

  /** @internal Capture a dialog event. The pending engine capture keeps the dialog from being auto-dismissed. */
  async _captureDialog(options?: { timeout?: number }): Promise<Dialog> {
    const params = await this.captureEventParams('dialog', options);
    return new Dialog(this.client, this.contextId, params);
  }

  /** @internal Capture a named event. */
  async _captureEvent(name: string, options?: { timeout?: number }): Promise<unknown> {
    switch (name) {
      case 'request':
        return this._captureRequest('**', options);
      case 'response':
        return this._captureResponse('**', options);
      case 'download':
        return this._captureDownload(options);
      case 'navigation':
        return this._captureNavigation(options);
      case 'dialog':
        return this._captureDialog(options);
      case 'console':
        return new ConsoleMessage(await this.captureEventParams('console', options));
      case 'error': {
        const params = await this.captureEventParams('error', options);
        return new Error((params.text as string) ?? 'Unknown error');
      }
      default:
        throw new Error(`Unknown event name: '${name}'`);
    }
  }

  /** Set extra HTTP headers for all requests in this page. */
  async setHeaders(headers: Record<string, string>): Promise<void> {
    const result = await this.client.send<{ intercept: string; headers: unknown }>('vibium:page.setHeaders', {
      context: this.contextId,
      headers,
    });

    // Store the intercept and headers for auto-continue in the event handler
    this.routes.push({
      pattern: '**',
      handler: (route: Route) => {
        // Merge custom headers with original request headers
        const merged = { ...route.request.headers(), ...headers };
        route.continue({ headers: merged });
      },
      interceptId: result.intercept,
    });
  }

  /** Intercept WebSocket connections. Not supported by BiDi. */
  routeWebSocket(_pattern: string, _handler: unknown): never {
    throw new Error('Not implemented: BiDi does not support WebSocket interception');
  }

  /**
   * Listen for WebSocket connections opened by the page.
   *
   * Monitoring is installed in the engine before the next command on this
   * connection is sent, so a socket opened by the very next call cannot be
   * missed (#351).
   */
  onWebSocket(fn: (ws: WebSocketInfo) => void): void {
    this.wsCallbacks.push(fn);
    // Keyed on the setup state, not the callback count: after a failed
    // install the callbacks are still registered, and the next registration
    // must retry the install or they can never fire.
    if (this.wsSetup === null) {
      const setup = this.client.sendSetup('vibium:page.onWebSocket', { context: this.contextId });
      this.wsSetup = setup;
      setup.catch(err => {
        // Reset so a later listener retries; sockets are unmonitored until
        // then. Guarded: a retry made in the meantime owns the state.
        if (this.wsSetup === setup) this.wsSetup = null;
        debug('page.onWebSocket setup failed', { error: String(err) });
      });
    }
  }

  /**
   * @internal Resolve when this page's WebSocket monitor is installed,
   * rejecting if the install failed. The sync wrapper awaits it so its
   * blocking onWebSocket() reports a failure the async caller cannot see.
   */
  async _whenWebSocketSetup(): Promise<void> {
    // Captured before awaiting: a failed install resets wsSetup to null, and
    // the raise must come from the setup this caller registered under.
    const setup = this.wsSetup;
    if (setup) await setup;
  }

  /**
   * @internal Remove one registered WebSocket callback. The sync wrapper
   * unregisters on a failed install so its raised call has no effect.
   */
  _removeWebSocketCallback(fn: (ws: WebSocketInfo) => void): void {
    const i = this.wsCallbacks.indexOf(fn);
    if (i !== -1) this.wsCallbacks.splice(i, 1);
  }

  // --- Dialog Handling ---

  /**
   * Register a handler for browser dialogs (alert, confirm, prompt).
   * If no handler is registered, dialogs are automatically dismissed.
   */
  onDialog(handler: (dialog: Dialog) => void): void {
    this.addDialogCallback(handler);
  }

  /**
   * The engine dismisses dialogs itself while no handler is registered
   * (#446); handlers flip it to manual so the dialog stays open for them.
   * sendSetup, so the policy is acknowledged before any later command can
   * trigger a dialog.
   */
  private syncDialogPolicy(): void {
    const manual = this.dialogCallbacks.length > 0;
    if (manual === this.dialogPolicyManual) return;
    this.dialogPolicyManual = manual;
    this.client.sendSetup('vibium:dialog.setPolicy', {
      context: this.contextId,
      policy: manual ? 'manual' : 'dismiss',
    }).catch(() => {});
  }

  private addDialogCallback(handler: (dialog: Dialog) => void): void {
    this.dialogCallbacks.push(handler);
    this.syncDialogPolicy();
  }

  private removeDialogCallback(handler: (dialog: Dialog) => void): void {
    this.dialogCallbacks = this.dialogCallbacks.filter(cb => cb !== handler);
    this.syncDialogPolicy();
  }

  /** Register a handler for console messages, or pass 'collect' to buffer them for consoleMessages(). */
  onConsole(handler: ((message: ConsoleMessage) => void) | 'collect'): void {
    if (handler === 'collect') {
      if (this._consoleBuffer === null) {
        this._consoleBuffer = [];
        this.consoleCallbacks.push((msg) => {
          this._consoleBuffer?.push({ type: msg.type(), text: msg.text() });
        });
      }
    } else {
      this.consoleCallbacks.push(handler);
    }
  }

  /** Return collected console messages and clear the buffer. Returns [] if not collecting. */
  consoleMessages(): { type: string; text: string }[] {
    const msgs = this._consoleBuffer || [];
    if (this._consoleBuffer) this._consoleBuffer = [];
    return msgs;
  }

  /** Register a handler for uncaught page errors, or pass 'collect' to buffer them for errors(). */
  onError(handler: ((error: Error) => void) | 'collect'): void {
    if (handler === 'collect') {
      if (this._errorBuffer === null) {
        this._errorBuffer = [];
        this.errorCallbacks.push((error) => {
          this._errorBuffer?.push({ message: error.message });
        });
      }
    } else {
      this.errorCallbacks.push(handler);
    }
  }

  /** Return collected errors and clear the buffer. Returns [] if not collecting. */
  errors(): { message: string }[] {
    const errs = this._errorBuffer || [];
    if (this._errorBuffer) this._errorBuffer = [];
    return errs;
  }

  /** Register a handler for file downloads. */
  onDownload(handler: (download: Download) => void): void {
    this.downloadCallbacks.push(handler);
  }

  // --- Event Handlers (internal) ---

  private ensureDataCollector(): void {
    if (this.dataCollectorId !== null) return;
    this.dataCollectorId = 'pending';
    // sendSetup, not send: the collector must exist before the request whose
    // body a route/onResponse handler is about to read (#351).
    this.client.sendSetup<{ collector: string }>(
      'network.addDataCollector',
      { dataTypes: ['request', 'response'], maxEncodedDataSize: 10 * 1024 * 1024 }
    ).then(result => {
      this.dataCollectorId = result.collector;
    }).catch(err => {
      // Reset so a later listener retries; bodies are unavailable until then.
      this.dataCollectorId = null;
      debug('page.ensureDataCollector failed', { error: String(err) });
    });
  }

  private teardownDataCollector(): void {
    const id = this.dataCollectorId;
    if (!id || id === 'pending') {
      this.dataCollectorId = null;
      return;
    }
    this.dataCollectorId = null;
    this.client.send('network.removeDataCollector', { collector: id }).catch(() => {});
  }

  private handleBeforeRequestSent(params: Record<string, unknown>): void {
    const isBlocked = params.isBlocked as boolean | undefined;
    const request = params.request as Record<string, unknown> | undefined;
    const requestId = request?.request as string | undefined;

    if (isBlocked && requestId) {
      // This is an intercepted request. The binary already matched the URL
      // against every registered pattern (vibiumMatchedPatterns), so
      // dispatch is a membership check, not a glob evaluation.
      const matched = (params.vibiumMatchedPatterns as string[] | undefined) ?? [];
      const req = new Request(params, this.client);

      for (const routeEntry of this.routes) {
        if (matched.includes(routeEntry.pattern)) {
          const route = new Route(this.client, requestId, req);
          // Catch errors from async route handlers (fire-and-forget pattern)
          try {
            const result = routeEntry.handler(route) as unknown;
            if (result && typeof (result as Promise<void>).catch === 'function') {
              (result as Promise<void>).catch(() => {});
            }
          } catch (_) { /* ignore sync errors from handler */ }
          return;
        }
      }

      // No matching route — auto-continue
      this.client.send('network.continueRequest', { request: requestId }).catch(() => {});
    } else {
      // Not blocked — notify onRequest listeners
      const req = new Request(params, this.client);
      for (const cb of this.requestCallbacks) {
        cb(req);
      }
    }
  }

  private handleResponseCompleted(params: Record<string, unknown>): void {
    const resp = new Response(params, this.client);
    for (const cb of this.responseCallbacks) {
      cb(resp);
    }
  }

  private handleUserPromptOpened(params: Record<string, unknown>): void {
    // With no handler registered the engine dismisses the dialog itself
    // (#446), so there is nothing to do here but deliver.
    const dialog = new Dialog(this.client, this.contextId, params);

    for (const cb of this.dialogCallbacks) {
      // Catch errors from async handlers (dialog.accept/dismiss are fire-and-forget)
      try {
        const result = cb(dialog) as unknown;
        if (result && typeof (result as Promise<void>).catch === 'function') {
          (result as Promise<void>).catch(() => {});
        }
      } catch (_) { /* ignore sync errors from handler */ }
    }
  }

  private handleLogEntryAdded(params: Record<string, unknown>): void {
    const entryType = params.type as string;

    if (entryType === 'console') {
      const msg = new ConsoleMessage(params);
      for (const cb of this.consoleCallbacks) {
        cb(msg);
      }
    } else if (entryType === 'javascript') {
      const text = (params.text as string) ?? 'Unknown error';
      const error = new Error(text);
      for (const cb of this.errorCallbacks) {
        cb(error);
      }
    }
  }

  private handleDownloadWillBegin(params: Record<string, unknown>): void {
    // Completion is awaited in the engine by navigation id (#446), so
    // there is no client-side pending map to feed on downloadEnd.
    const download = new Download(this.client, params);
    for (const cb of this.downloadCallbacks) {
      cb(download);
    }
  }

  private handleWsCreated(params: Record<string, unknown>): void {
    const id = params.id as number;
    const url = params.url as string;
    const ws = new WebSocketInfo(url);
    this.wsConnections.set(id, ws);
    for (const cb of this.wsCallbacks) {
      cb(ws);
    }
  }

  private handleWsMessage(params: Record<string, unknown>): void {
    const id = params.id as number;
    const data = params.data as string;
    const direction = params.direction as 'sent' | 'received';
    const ws = this.wsConnections.get(id);
    if (ws) {
      ws._emitMessage(data, direction);
    }
  }

  private handleWsClosed(params: Record<string, unknown>): void {
    const id = params.id as number;
    const code = params.code as number | undefined;
    const reason = params.reason as string | undefined;
    const ws = this.wsConnections.get(id);
    if (ws) {
      ws._emitClose(code, reason);
      this.wsConnections.delete(id);
    }
  }
}
