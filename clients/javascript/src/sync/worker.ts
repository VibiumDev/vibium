import { parentPort, workerData, MessagePort } from 'worker_threads';
import { browser, Browser } from '../browser';
import { Page } from '../page';
import { BrowserContext } from '../context';
import { Element, SelectorOptions } from '../element';
import { ElementList, FilterOptions } from '../element-list';

interface WorkerData {
  signal: Int32Array;
  callbackPort: MessagePort;
}

interface Command {
  id: number;
  method: string;
  args: unknown[];
}

type Handler = (args: unknown[]) => Promise<unknown>;

const { signal, callbackPort } = workerData as WorkerData;

// --- Callback mutex ---
// Ensures only one invokeMainThread call is in flight at a time.
let callbackChain = Promise.resolve<unknown>(undefined);

/**
 * Invoke a handler on the main thread and return its decision.
 * Posts data to the callback port, signals the main thread (signal[0]=2),
 * then waits for the decision to arrive back via the port.
 */
function invokeMainThread(handlerId: string, data: unknown): Promise<unknown> {
  const call = callbackChain.then(async () => {
    // Set up message listener BEFORE signaling main thread
    const decisionPromise = new Promise<unknown>((resolve) => {
      callbackPort.once('message', (msg: { decision: unknown }) => {
        resolve(msg.decision);
      });
    });

    // Post callback request to main thread
    callbackPort.postMessage({ handlerId, data });

    // Signal main thread: callback needed
    Atomics.store(signal, 0, 2);
    Atomics.notify(signal, 0);

    // Wait for decision via port message (reliable, no Atomics race)
    return await decisionPromise;
  });
  // Chain: next call waits for this one to finish
  callbackChain = call.then(() => {}, () => {});
  return call;
}

// --- Object registries ---
let browserInstance: Browser | null = null;
let nextId = 1;
const pages = new Map<number, Page>();
const contexts = new Map<number, BrowserContext>();
const elements = new Map<number, Element>();
const elementLists = new Map<number, ElementList>();
const pageContextMap = new Map<number, number>(); // pageId → contextId (sync worker IDs)

// Default page ID (first page from launch)
let defaultPageId = 0;

function allocId(): number {
  return nextId++;
}

function getPage(id: number): Page {
  const p = pages.get(id);
  if (!p) throw new Error(`Page ${id} not found`);
  return p;
}

function getContext(id: number): BrowserContext {
  const c = contexts.get(id);
  if (!c) throw new Error(`Context ${id} not found`);
  return c;
}

function getElement(id: number): Element {
  const el = elements.get(id);
  if (!el) throw new Error(`Element ${id} not found`);
  return el;
}

function getElementList(id: number): ElementList {
  const list = elementLists.get(id);
  if (!list) throw new Error(`ElementList ${id} not found`);
  return list;
}

function storePage(page: Page): number {
  const id = allocId();
  pages.set(id, page);

  // Track page→context mapping: find or create sync context for this page's BrowserContext
  const pageContext = page.context;
  let contextSyncId: number | undefined;
  for (const [cid, ctx] of contexts) {
    if (ctx.id === pageContext.id) {
      contextSyncId = cid;
      break;
    }
  }
  if (contextSyncId === undefined) {
    contextSyncId = storeContext(pageContext);
  }
  pageContextMap.set(id, contextSyncId);

  return id;
}

function storeContext(ctx: BrowserContext): number {
  const id = allocId();
  contexts.set(id, ctx);
  return id;
}

function storeElement(el: Element): number {
  const id = allocId();
  elements.set(id, el);
  return id;
}

function storeElementList(list: ElementList): number {
  const id = allocId();
  elementLists.set(id, list);
  return id;
}

// --- Event buffers ---
// Events are buffered here during command processing. After the command completes,
// buffered events are delivered via invokeMainThread BEFORE the result is sent.
// This avoids the signal race between callback (signal=2) and result (signal=1).
const pageEventBuffer: Array<{ type: 'page' | 'popup'; pageId: number }> = [];
let onPageHandlerId: string | null = null;
let onPopupHandlerId: string | null = null;

// Network/download/websocket event buffers — same pattern as page events.
const networkEventBuffer: Array<{ handlerId: string; data: unknown }> = [];
let onRequestHandlerIds = new Map<number, string>();  // pageId → handlerId
let onResponseHandlerIds = new Map<number, string>(); // pageId → handlerId
let onDownloadHandlerIds = new Map<number, string>(); // pageId → handlerId
let onWebSocketHandlerIds = new Map<number, string>(); // pageId → handlerId

// --- Dispatch table ---

const handlers: Record<string, Handler> = {
  // ========================
  // Browser commands
  // ========================

  'browser.launch': async (args) => {
    const [options] = args as [{ headless?: boolean } | undefined];
    browserInstance = await browser.launch(options);
    const page = await browserInstance.page();
    defaultPageId = storePage(page);
    return { pageId: defaultPageId };
  },

  'browser.close': async () => {
    if (!browserInstance) throw new Error('Browser not launched');
    await browserInstance.close();
    browserInstance = null;
    pages.clear();
    contexts.clear();
    elements.clear();
    elementLists.clear();
    return { success: true };
  },

  'browser.page': async () => {
    if (!browserInstance) throw new Error('Browser not launched');
    if (defaultPageId && pages.has(defaultPageId)) {
      return { pageId: defaultPageId };
    }
    const page = await browserInstance.page();
    defaultPageId = storePage(page);
    return { pageId: defaultPageId };
  },

  'browser.newPage': async () => {
    if (!browserInstance) throw new Error('Browser not launched');
    const page = await browserInstance.newPage();
    const id = storePage(page);
    return { pageId: id };
  },

  'browser.newContext': async () => {
    if (!browserInstance) throw new Error('Browser not launched');
    const ctx = await browserInstance.newContext();
    const id = storeContext(ctx);
    return { contextId: id };
  },

  'browser.pages': async () => {
    if (!browserInstance) throw new Error('Browser not launched');
    const allPages = await browserInstance.pages();
    const ids: number[] = [];
    for (const p of allPages) {
      let found = false;
      for (const [id, stored] of pages) {
        if (stored.id === p.id) {
          ids.push(id);
          found = true;
          break;
        }
      }
      if (!found) {
        ids.push(storePage(p));
      }
    }
    return { pageIds: ids };
  },

  'browser.onPage': async (args) => {
    if (!browserInstance) throw new Error('Browser not launched');
    const [handlerId] = args as [string];
    onPageHandlerId = handlerId;
    // Snapshot existing page context IDs to skip late contextCreated events
    const existingPages = await browserInstance.pages();
    const knownContextIds = new Set(existingPages.map(p => p.id));
    browserInstance.removeAllListeners('page');
    // Yield to let any in-flight contextCreated events be processed
    await new Promise(r => setImmediate(r));
    browserInstance.onPage((page) => {
      if (knownContextIds.has(page.id)) return;
      knownContextIds.add(page.id); // Prevent duplicate events for same context
      const id = storePage(page);
      pageEventBuffer.push({ type: 'page', pageId: id });
    });
    return { success: true };
  },

  'browser.onPopup': async (args) => {
    if (!browserInstance) throw new Error('Browser not launched');
    const [handlerId] = args as [string];
    onPopupHandlerId = handlerId;
    // Snapshot existing page context IDs to skip late contextCreated events
    const existingPages = await browserInstance.pages();
    const knownContextIds = new Set(existingPages.map(p => p.id));
    browserInstance.removeAllListeners('popup');
    // Yield to let any in-flight contextCreated events be processed
    await new Promise(r => setImmediate(r));
    browserInstance.onPopup((page) => {
      if (knownContextIds.has(page.id)) return;
      knownContextIds.add(page.id); // Prevent duplicate events for same context
      const id = storePage(page);
      pageEventBuffer.push({ type: 'popup', pageId: id });
    });
    return { success: true };
  },

  'browser.removeAllListeners': async (args) => {
    if (!browserInstance) throw new Error('Browser not launched');
    const [event] = args as [string | undefined];
    browserInstance.removeAllListeners(event as any);
    if (!event || event === 'page') onPageHandlerId = null;
    if (!event || event === 'popup') onPopupHandlerId = null;
    return { success: true };
  },

  'browser.waitForPage': async (args) => {
    if (!browserInstance) throw new Error('Browser not launched');
    const [options] = args as [{ timeout?: number } | undefined];
    const timeout = options?.timeout ?? 10000;
    const page = await new Promise<Page>((resolve, reject) => {
      const timer = setTimeout(() => {
        browserInstance?.removeAllListeners('page');
        reject(new Error('Timeout waiting for new page'));
      }, timeout);
      browserInstance!.onPage((p) => {
        clearTimeout(timer);
        resolve(p);
      });
    });
    const id = storePage(page);
    return { pageId: id };
  },

  'browser.waitForPopup': async (args) => {
    if (!browserInstance) throw new Error('Browser not launched');
    const [options] = args as [{ timeout?: number } | undefined];
    const timeout = options?.timeout ?? 10000;
    const page = await new Promise<Page>((resolve, reject) => {
      const timer = setTimeout(() => {
        browserInstance?.removeAllListeners('popup');
        reject(new Error('Timeout waiting for popup'));
      }, timeout);
      browserInstance!.onPopup((p) => {
        clearTimeout(timer);
        resolve(p);
      });
    });
    const id = storePage(page);
    return { pageId: id };
  },

  // ========================
  // Page commands
  // ========================

  'page.go': async (args) => {
    const [pageId, url] = args as [number, string];
    await getPage(pageId).go(url);
    return { success: true };
  },

  'page.back': async (args) => {
    const [pageId] = args as [number];
    await getPage(pageId).back();
    return { success: true };
  },

  'page.forward': async (args) => {
    const [pageId] = args as [number];
    await getPage(pageId).forward();
    return { success: true };
  },

  'page.reload': async (args) => {
    const [pageId] = args as [number];
    await getPage(pageId).reload();
    return { success: true };
  },

  'page.url': async (args) => {
    const [pageId] = args as [number];
    const url = await getPage(pageId).url();
    return { url };
  },

  'page.title': async (args) => {
    const [pageId] = args as [number];
    const title = await getPage(pageId).title();
    return { title };
  },

  'page.content': async (args) => {
    const [pageId] = args as [number];
    const content = await getPage(pageId).content();
    return { content };
  },

  'page.screenshot': async (args) => {
    const [pageId, options] = args as [number, unknown];
    const buffer = await getPage(pageId).screenshot(options as any);
    return { data: buffer.toString('base64') };
  },

  'page.pdf': async (args) => {
    const [pageId] = args as [number];
    const buffer = await getPage(pageId).pdf();
    return { data: buffer.toString('base64') };
  },

  'page.evaluate': async (args) => {
    const [pageId, script] = args as [number, string];
    const result = await getPage(pageId).evaluate(script);
    return { result };
  },

  'page.eval': async (args) => {
    const [pageId, expression] = args as [number, string];
    const value = await getPage(pageId).eval(expression);
    return { value };
  },

  'page.evalHandle': async (args) => {
    const [pageId, expression] = args as [number, string];
    const handle = await getPage(pageId).evalHandle(expression);
    return { handle };
  },

  'page.addScript': async (args) => {
    const [pageId, source] = args as [number, string];
    await getPage(pageId).addScript(source);
    return { success: true };
  },

  'page.addStyle': async (args) => {
    const [pageId, source] = args as [number, string];
    await getPage(pageId).addStyle(source);
    return { success: true };
  },

  'page.expose': async (args) => {
    const [pageId, name, fn] = args as [number, string, string];
    await getPage(pageId).expose(name, fn);
    return { success: true };
  },

  'page.find': async (args) => {
    const [pageId, selector, options] = args as [number, string | SelectorOptions, unknown];
    const element = await getPage(pageId).find(selector, options as any);
    const elementId = storeElement(element);
    return { elementId, info: element.info };
  },

  'page.findAll': async (args) => {
    const [pageId, selector, options] = args as [number, string | SelectorOptions, unknown];
    const list = await getPage(pageId).findAll(selector, options as any);
    const listId = storeElementList(list);
    const elementIds: number[] = [];
    for (const el of list.toArray()) {
      elementIds.push(storeElement(el));
    }
    return { listId, elementIds, count: list.count() };
  },

  'page.wait': async (args) => {
    const [pageId, ms] = args as [number, number];
    await getPage(pageId).wait(ms);
    return { success: true };
  },

  'page.waitForURL': async (args) => {
    const [pageId, pattern, options] = args as [number, string, { timeout?: number } | undefined];
    await getPage(pageId).waitUntil.url(pattern, options);
    return { success: true };
  },

  'page.waitForLoad': async (args) => {
    const [pageId, state, options] = args as [number, string | undefined, { timeout?: number } | undefined];
    await getPage(pageId).waitUntil.loaded(state, options);
    return { success: true };
  },

  'page.waitForFunction': async (args) => {
    const [pageId, fn, options] = args as [number, string, { timeout?: number } | undefined];
    const value = await getPage(pageId).waitUntil(fn, options);
    return { value };
  },

  'page.bringToFront': async (args) => {
    const [pageId] = args as [number];
    await getPage(pageId).bringToFront();
    return { success: true };
  },

  'page.close': async (args) => {
    const [pageId] = args as [number];
    await getPage(pageId).close();
    pages.delete(pageId);
    return { success: true };
  },

  // --- Page emulation ---

  'page.setViewport': async (args) => {
    const [pageId, size] = args as [number, { width: number; height: number }];
    await getPage(pageId).setViewport(size);
    return { success: true };
  },

  'page.viewport': async (args) => {
    const [pageId] = args as [number];
    return await getPage(pageId).viewport();
  },

  'page.emulateMedia': async (args) => {
    const [pageId, opts] = args as [number, any];
    await getPage(pageId).emulateMedia(opts);
    return { success: true };
  },

  'page.setContent': async (args) => {
    const [pageId, html] = args as [number, string];
    await getPage(pageId).setContent(html);
    return { success: true };
  },

  'page.setGeolocation': async (args) => {
    const [pageId, coords] = args as [number, { latitude: number; longitude: number; accuracy?: number }];
    await getPage(pageId).setGeolocation(coords);
    return { success: true };
  },

  'page.setWindow': async (args) => {
    const [pageId, options] = args as [number, any];
    await getPage(pageId).setWindow(options);
    return { success: true };
  },

  'page.window': async (args) => {
    const [pageId] = args as [number];
    return await getPage(pageId).window();
  },

  // --- Page frames ---

  'page.frames': async (args) => {
    const [pageId] = args as [number];
    const frames = await getPage(pageId).frames();
    const frameIds = frames.map(f => storePage(f));
    return { frameIds };
  },

  'page.frame': async (args) => {
    const [pageId, nameOrUrl] = args as [number, string];
    const frame = await getPage(pageId).frame(nameOrUrl);
    if (!frame) return { frameId: null };
    const id = storePage(frame);
    return { frameId: id };
  },

  'page.mainFrame': async (args) => {
    const [pageId] = args as [number];
    return { frameId: pageId };
  },

  // --- Page accessibility ---

  'page.a11yTree': async (args) => {
    const [pageId, options] = args as [number, any];
    const tree = await getPage(pageId).a11yTree(options);
    return { tree };
  },

  'page.context': async (args) => {
    const [pageId] = args as [number];
    const contextId = pageContextMap.get(pageId);
    if (contextId === undefined) throw new Error(`No context found for page ${pageId}`);
    return { contextId };
  },

  // --- Page network ---

  'page.route': async (args) => {
    const [pageId, pattern, action] = args as [number, string, string | Record<string, unknown>];
    const page = getPage(pageId);
    await page.route(pattern, (route) => {
      if (action === 'continue') {
        route.continue();
      } else if (action === 'abort') {
        route.abort();
      } else if (typeof action === 'object') {
        route.fulfill(action as any);
      }
    });
    return { success: true };
  },

  'page.routeWithCallback': async (args) => {
    const [pageId, pattern, handlerId] = args as [number, string, string];
    const page = getPage(pageId);
    await page.route(pattern, async (route) => {
      // Serialize request data for main thread
      // postData() is async (sends network.getData) which may not be supported
      // by the proxy, so use a short timeout to avoid hanging indefinitely.
      let postData: string | null = null;
      try {
        postData = await Promise.race([
          route.request.postData(),
          new Promise<null>(resolve => setTimeout(() => resolve(null), 500)),
        ]);
      } catch {
        // Ignore errors — postData is best-effort
      }
      const requestData = {
        url: route.request.url(),
        method: route.request.method(),
        headers: route.request.headers(),
        postData,
      };

      // Invoke handler on main thread and get decision
      const decision = await invokeMainThread(handlerId, requestData) as any;

      // Apply decision (default: continue)
      if (!decision || decision.action === 'continue') {
        const overrides: Record<string, unknown> = {};
        if (decision?.url) overrides.url = decision.url;
        if (decision?.method) overrides.method = decision.method;
        if (decision?.headers) overrides.headers = decision.headers;
        if (decision?.postData) overrides.postData = decision.postData;
        await route.continue(Object.keys(overrides).length > 0 ? overrides as any : undefined);
      } else if (decision.action === 'fulfill') {
        await route.fulfill({
          status: decision.status,
          headers: decision.headers,
          contentType: decision.contentType,
          body: decision.body,
        });
      } else if (decision.action === 'abort') {
        await route.abort();
      }
    });
    return { success: true };
  },

  'page.unroute': async (args) => {
    const [pageId, pattern] = args as [number, string];
    await getPage(pageId).unroute(pattern);
    return { success: true };
  },

  'page.setHeaders': async (args) => {
    const [pageId, headers] = args as [number, Record<string, string>];
    await getPage(pageId).setHeaders(headers);
    return { success: true };
  },

  'page.waitForRequest': async (args) => {
    const [pageId, pattern, options] = args as [number, string, { timeout?: number } | undefined];
    const page = getPage(pageId);
    const request = await page.capture.request(pattern, undefined, options);
    return {
      url: request.url(),
      method: request.method(),
      headers: request.headers(),
      postData: await request.postData(),
    };
  },

  'page.waitForResponse': async (args) => {
    const [pageId, pattern, options] = args as [number, string, { timeout?: number } | undefined];
    const page = getPage(pageId);
    const response = await page.capture.response(pattern, undefined, options);
    return {
      url: response.url(),
      status: response.status(),
      headers: response.headers(),
      body: await response.body(),
    };
  },

  'page.captureResponseStart': async (args) => {
    const [pageId, pattern, options] = args as [number, string, { timeout?: number } | undefined];
    const page = getPage(pageId);
    (page as any)._pendingCaptureResponse = page.capture.response(pattern, undefined, options);
    return { success: true };
  },

  'page.captureResponseFinish': async (args) => {
    const [pageId] = args as [number];
    const page = getPage(pageId);
    const response = await (page as any)._pendingCaptureResponse;
    delete (page as any)._pendingCaptureResponse;
    return {
      url: response.url(),
      status: response.status(),
      headers: response.headers(),
      body: await response.body(),
    };
  },

  'page.captureRequestStart': async (args) => {
    const [pageId, pattern, options] = args as [number, string, { timeout?: number } | undefined];
    const page = getPage(pageId);
    (page as any)._pendingCaptureRequest = page.capture.request(pattern, undefined, options);
    return { success: true };
  },

  'page.captureRequestFinish': async (args) => {
    const [pageId] = args as [number];
    const page = getPage(pageId);
    const request = await (page as any)._pendingCaptureRequest;
    delete (page as any)._pendingCaptureRequest;
    return {
      url: request.url(),
      method: request.method(),
      headers: request.headers(),
      postData: await request.postData(),
    };
  },

  'page.captureNavigationStart': async (args) => {
    const [pageId, options] = args as [number, { timeout?: number } | undefined];
    const page = getPage(pageId);
    (page as any)._pendingCaptureNavigation = page.capture.navigation(undefined, options);
    return { success: true };
  },

  'page.captureNavigationFinish': async (args) => {
    const [pageId] = args as [number];
    const page = getPage(pageId);
    const url = await (page as any)._pendingCaptureNavigation;
    delete (page as any)._pendingCaptureNavigation;
    return { url };
  },

  'page.captureDownloadStart': async (args) => {
    const [pageId, options] = args as [number, { timeout?: number } | undefined];
    const page = getPage(pageId);
    (page as any)._pendingCaptureDownload = page.capture.download(undefined, options);
    return { success: true };
  },

  'page.captureDownloadFinish': async (args) => {
    const [pageId] = args as [number];
    const page = getPage(pageId);
    const download = await (page as any)._pendingCaptureDownload;
    delete (page as any)._pendingCaptureDownload;
    const filePath = await download.path();
    return {
      url: download.url(),
      suggestedFilename: download.suggestedFilename(),
      path: filePath,
    };
  },

  'page.captureDialogStart': async (args) => {
    const [pageId, options] = args as [number, { timeout?: number } | undefined];
    const page = getPage(pageId);
    (page as any)._pendingCaptureDialog = page.capture.dialog(undefined, options);
    return { success: true };
  },

  'page.captureDialogFinish': async (args) => {
    const [pageId] = args as [number];
    const page = getPage(pageId);
    const dialog = await (page as any)._pendingCaptureDialog;
    delete (page as any)._pendingCaptureDialog;
    return {
      type: dialog.type(),
      message: dialog.message(),
      defaultValue: dialog.defaultValue(),
    };
  },

  'page.captureEventStart': async (args) => {
    const [pageId, name, options] = args as [number, string, { timeout?: number } | undefined];
    const page = getPage(pageId);
    (page as any)._pendingCaptureEvent = page.capture.event(name, undefined, options);
    return { success: true };
  },

  'page.captureEventFinish': async (args) => {
    const [pageId] = args as [number];
    const page = getPage(pageId);
    const data = await (page as any)._pendingCaptureEvent;
    delete (page as any)._pendingCaptureEvent;
    // Return the event data — for typed events, serialize known shapes
    if (data && typeof data === 'object') {
      if ('url' in data && 'status' in data) {
        // Response-like
        return { url: data.url(), status: data.status(), headers: data.headers() };
      }
      if ('url' in data && 'method' in data) {
        // Request-like
        return { url: data.url(), method: data.method(), headers: data.headers() };
      }
      if ('type' in data && 'message' in data) {
        // Dialog-like
        return { type: data.type(), message: data.message(), defaultValue: data.defaultValue() };
      }
    }
    // For string (navigation URL) or other simple values, wrap in data
    return { data };
  },

  // --- Page events (simplified for sync) ---

  'page.onDialog': async (args) => {
    const [pageId, action] = args as [number, 'accept' | 'dismiss'];
    const page = getPage(pageId);
    page.onDialog((dialog) => {
      if (action === 'accept') {
        dialog.accept().catch(() => {});
      } else {
        dialog.dismiss().catch(() => {});
      }
    });
    return { success: true };
  },

  'page.onDialogWithCallback': async (args) => {
    const [pageId, handlerId] = args as [number, string];
    const page = getPage(pageId);
    page.onDialog(async (dialog) => {
      // Serialize dialog data for main thread
      const dialogData = {
        type: dialog.type(),
        message: dialog.message(),
        defaultValue: dialog.defaultValue(),
      };

      // Invoke handler on main thread and get decision
      const decision = await invokeMainThread(handlerId, dialogData) as any;

      // Apply decision (default: dismiss)
      if (!decision || decision.action === 'dismiss') {
        await dialog.dismiss();
      } else if (decision.action === 'accept') {
        await dialog.accept(decision.promptText);
      }
    });
    return { success: true };
  },

  'page.onConsole': async (args) => {
    const [pageId, mode] = args as [number, 'collect'];
    if (mode !== 'collect') throw new Error('Only "collect" mode supported');
    const page = getPage(pageId);
    if (!(page as any)._syncConsoleBuffer) {
      (page as any)._syncConsoleBuffer = [] as any[];
      page.onConsole((msg) => {
        (page as any)._syncConsoleBuffer.push({
          type: msg.type(),
          text: msg.text(),
        });
      });
    }
    return { success: true };
  },

  'page.consoleMessages': async (args) => {
    const [pageId] = args as [number];
    const page = getPage(pageId);
    const buffer = (page as any)._syncConsoleBuffer || [];
    (page as any)._syncConsoleBuffer = [];
    return { messages: buffer };
  },

  'page.onError': async (args) => {
    const [pageId, mode] = args as [number, 'collect'];
    if (mode !== 'collect') throw new Error('Only "collect" mode supported');
    const page = getPage(pageId);
    if (!(page as any)._syncErrorBuffer) {
      (page as any)._syncErrorBuffer = [] as any[];
      page.onError((error) => {
        (page as any)._syncErrorBuffer.push({
          message: error.message,
        });
      });
    }
    return { success: true };
  },

  'page.errors': async (args) => {
    const [pageId] = args as [number];
    const page = getPage(pageId);
    const buffer = (page as any)._syncErrorBuffer || [];
    (page as any)._syncErrorBuffer = [];
    return { errors: buffer };
  },

  'page.onRequestWithCallback': async (args) => {
    const [pageId, handlerId] = args as [number, string];
    const page = getPage(pageId);
    onRequestHandlerIds.set(pageId, handlerId);
    page.onRequest(async (req) => {
      const hid = onRequestHandlerIds.get(pageId);
      if (!hid) return;
      networkEventBuffer.push({
        handlerId: hid,
        data: { url: req.url(), method: req.method(), headers: req.headers(), postData: await req.postData() },
      });
    });
    return { success: true };
  },

  'page.onResponseWithCallback': async (args) => {
    const [pageId, handlerId] = args as [number, string];
    const page = getPage(pageId);
    onResponseHandlerIds.set(pageId, handlerId);
    page.onResponse(async (resp) => {
      const hid = onResponseHandlerIds.get(pageId);
      if (!hid) return;
      networkEventBuffer.push({
        handlerId: hid,
        data: { url: resp.url(), status: resp.status(), headers: resp.headers(), body: await resp.body() },
      });
    });
    return { success: true };
  },

  'page.onDownloadWithCallback': async (args) => {
    const [pageId, handlerId] = args as [number, string];
    const page = getPage(pageId);
    onDownloadHandlerIds.set(pageId, handlerId);
    page.onDownload(async (dl) => {
      const hid = onDownloadHandlerIds.get(pageId);
      if (!hid) return;
      const filePath = await dl.path();
      networkEventBuffer.push({
        handlerId: hid,
        data: { url: dl.url(), suggestedFilename: dl.suggestedFilename(), path: filePath },
      });
    });
    return { success: true };
  },

  'page.onWebSocketWithCallback': async (args) => {
    const [pageId, handlerId] = args as [number, string];
    const page = getPage(pageId);
    onWebSocketHandlerIds.set(pageId, handlerId);
    let nextWsId = 0;
    page.onWebSocket((ws) => {
      const hid = onWebSocketHandlerIds.get(pageId);
      if (!hid) return;
      const wsId = nextWsId++;
      networkEventBuffer.push({
        handlerId: hid,
        data: { type: 'created', wsId, url: ws.url() },
      });

      ws.onMessage((data, info) => {
        const hid2 = onWebSocketHandlerIds.get(pageId);
        if (!hid2) return;
        networkEventBuffer.push({
          handlerId: hid2,
          data: { type: 'message', wsId, data, direction: info.direction },
        });
      });

      ws.onClose((code, reason) => {
        const hid2 = onWebSocketHandlerIds.get(pageId);
        if (!hid2) return;
        networkEventBuffer.push({
          handlerId: hid2,
          data: { type: 'close', wsId, code, reason },
        });
      });
    });
    return { success: true };
  },

  'page.removeAllListeners': async (args) => {
    const [pageId, event] = args as [number, string | undefined];
    getPage(pageId).removeAllListeners(event as any);
    // Clear handler IDs for removed events
    if (!event || event === 'request') onRequestHandlerIds.delete(pageId);
    if (!event || event === 'response') onResponseHandlerIds.delete(pageId);
    if (!event || event === 'download') onDownloadHandlerIds.delete(pageId);
    if (!event || event === 'websocket') onWebSocketHandlerIds.delete(pageId);
    return { success: true };
  },

  // ========================
  // Context commands
  // ========================

  'context.newPage': async (args) => {
    const [contextId] = args as [number];
    const ctx = getContext(contextId);
    const page = await ctx.newPage();
    const id = storePage(page);
    return { pageId: id };
  },

  'context.close': async (args) => {
    const [contextId] = args as [number];
    await getContext(contextId).close();
    contexts.delete(contextId);
    return { success: true };
  },

  'context.cookies': async (args) => {
    const [contextId, urls] = args as [number, string[] | undefined];
    const cookies = await getContext(contextId).cookies(urls);
    return { cookies };
  },

  'context.setCookies': async (args) => {
    const [contextId, cookies] = args as [number, any[]];
    await getContext(contextId).setCookies(cookies);
    return { success: true };
  },

  'context.clearCookies': async (args) => {
    const [contextId] = args as [number];
    await getContext(contextId).clearCookies();
    return { success: true };
  },

  'context.storageState': async (args) => {
    const [contextId] = args as [number];
    return await getContext(contextId).storageState();
  },

  'context.addInitScript': async (args) => {
    const [contextId, script] = args as [number, string];
    const result = await getContext(contextId).addInitScript(script);
    return { script: result };
  },

  // ========================
  // Tracing commands (context-scoped)
  // ========================

  'tracing.start': async (args) => {
    const [contextId, options] = args as [number, any];
    await getContext(contextId).tracing.start(options);
    return { success: true };
  },

  'tracing.stop': async (args) => {
    const [contextId, options] = args as [number, any];
    const buffer = await getContext(contextId).tracing.stop(options);
    return { data: buffer.toString('base64') };
  },

  'tracing.startChunk': async (args) => {
    const [contextId, options] = args as [number, any];
    await getContext(contextId).tracing.startChunk(options);
    return { success: true };
  },

  'tracing.stopChunk': async (args) => {
    const [contextId, options] = args as [number, any];
    const buffer = await getContext(contextId).tracing.stopChunk(options);
    return { data: buffer.toString('base64') };
  },

  'tracing.startGroup': async (args) => {
    const [contextId, name, options] = args as [number, string, any];
    await getContext(contextId).tracing.startGroup(name, options);
    return { success: true };
  },

  'tracing.stopGroup': async (args) => {
    const [contextId] = args as [number];
    await getContext(contextId).tracing.stopGroup();
    return { success: true };
  },

  // ========================
  // Clock commands (page-scoped)
  // ========================

  'clock.install': async (args) => {
    const [pageId, options] = args as [number, any];
    await getPage(pageId).clock.install(options);
    return { success: true };
  },

  'clock.fastForward': async (args) => {
    const [pageId, ticks] = args as [number, number];
    await getPage(pageId).clock.fastForward(ticks);
    return { success: true };
  },

  'clock.runFor': async (args) => {
    const [pageId, ticks] = args as [number, number];
    await getPage(pageId).clock.runFor(ticks);
    return { success: true };
  },

  'clock.pauseAt': async (args) => {
    const [pageId, time] = args as [number, number | string];
    await getPage(pageId).clock.pauseAt(time);
    return { success: true };
  },

  'clock.resume': async (args) => {
    const [pageId] = args as [number];
    await getPage(pageId).clock.resume();
    return { success: true };
  },

  'clock.setFixedTime': async (args) => {
    const [pageId, time] = args as [number, number | string];
    await getPage(pageId).clock.setFixedTime(time);
    return { success: true };
  },

  'clock.setSystemTime': async (args) => {
    const [pageId, time] = args as [number, number | string];
    await getPage(pageId).clock.setSystemTime(time);
    return { success: true };
  },

  'clock.setTimezone': async (args) => {
    const [pageId, timezone] = args as [number, string];
    await getPage(pageId).clock.setTimezone(timezone);
    return { success: true };
  },

  // ========================
  // Keyboard commands (page-scoped)
  // ========================

  'keyboard.press': async (args) => {
    const [pageId, key] = args as [number, string];
    await getPage(pageId).keyboard.press(key);
    return { success: true };
  },

  'keyboard.down': async (args) => {
    const [pageId, key] = args as [number, string];
    await getPage(pageId).keyboard.down(key);
    return { success: true };
  },

  'keyboard.up': async (args) => {
    const [pageId, key] = args as [number, string];
    await getPage(pageId).keyboard.up(key);
    return { success: true };
  },

  'keyboard.type': async (args) => {
    const [pageId, text] = args as [number, string];
    await getPage(pageId).keyboard.type(text);
    return { success: true };
  },

  // ========================
  // Mouse commands (page-scoped)
  // ========================

  'mouse.click': async (args) => {
    const [pageId, x, y] = args as [number, number, number];
    await getPage(pageId).mouse.click(x, y);
    return { success: true };
  },

  'mouse.move': async (args) => {
    const [pageId, x, y] = args as [number, number, number];
    await getPage(pageId).mouse.move(x, y);
    return { success: true };
  },

  'mouse.down': async (args) => {
    const [pageId] = args as [number];
    await getPage(pageId).mouse.down();
    return { success: true };
  },

  'mouse.up': async (args) => {
    const [pageId] = args as [number];
    await getPage(pageId).mouse.up();
    return { success: true };
  },

  'mouse.wheel': async (args) => {
    const [pageId, deltaX, deltaY] = args as [number, number, number];
    await getPage(pageId).mouse.wheel(deltaX, deltaY);
    return { success: true };
  },

  // ========================
  // Touch commands (page-scoped)
  // ========================

  'touch.tap': async (args) => {
    const [pageId, x, y] = args as [number, number, number];
    await getPage(pageId).touch.tap(x, y);
    return { success: true };
  },

  // ========================
  // Element commands
  // ========================

  'element.click': async (args) => {
    const [elementId, options] = args as [number, any];
    await getElement(elementId).click(options);
    return { success: true };
  },

  'element.dblclick': async (args) => {
    const [elementId, options] = args as [number, any];
    await getElement(elementId).dblclick(options);
    return { success: true };
  },

  'element.fill': async (args) => {
    const [elementId, value, options] = args as [number, string, any];
    await getElement(elementId).fill(value, options);
    return { success: true };
  },

  'element.type': async (args) => {
    const [elementId, text, options] = args as [number, string, any];
    await getElement(elementId).type(text, options);
    return { success: true };
  },

  'element.press': async (args) => {
    const [elementId, key, options] = args as [number, string, any];
    await getElement(elementId).press(key, options);
    return { success: true };
  },

  'element.clear': async (args) => {
    const [elementId, options] = args as [number, any];
    await getElement(elementId).clear(options);
    return { success: true };
  },

  'element.check': async (args) => {
    const [elementId, options] = args as [number, any];
    await getElement(elementId).check(options);
    return { success: true };
  },

  'element.uncheck': async (args) => {
    const [elementId, options] = args as [number, any];
    await getElement(elementId).uncheck(options);
    return { success: true };
  },

  'element.selectOption': async (args) => {
    const [elementId, value, options] = args as [number, string, any];
    await getElement(elementId).selectOption(value, options);
    return { success: true };
  },

  'element.hover': async (args) => {
    const [elementId, options] = args as [number, any];
    await getElement(elementId).hover(options);
    return { success: true };
  },

  'element.focus': async (args) => {
    const [elementId, options] = args as [number, any];
    await getElement(elementId).focus(options);
    return { success: true };
  },

  'element.dragTo': async (args) => {
    const [elementId, targetId, options] = args as [number, number, any];
    const target = getElement(targetId);
    await getElement(elementId).dragTo(target, options);
    return { success: true };
  },

  'element.tap': async (args) => {
    const [elementId, options] = args as [number, any];
    await getElement(elementId).tap(options);
    return { success: true };
  },

  'element.scrollIntoView': async (args) => {
    const [elementId, options] = args as [number, any];
    await getElement(elementId).scrollIntoView(options);
    return { success: true };
  },

  'element.dispatchEvent': async (args) => {
    const [elementId, eventType, eventInit, options] = args as [number, string, any, any];
    await getElement(elementId).dispatchEvent(eventType, eventInit, options);
    return { success: true };
  },

  'element.setFiles': async (args) => {
    const [elementId, files, options] = args as [number, string[], any];
    await getElement(elementId).setFiles(files, options);
    return { success: true };
  },

  'element.text': async (args) => {
    const [elementId] = args as [number];
    const text = await getElement(elementId).text();
    return { text };
  },

  'element.innerText': async (args) => {
    const [elementId] = args as [number];
    const text = await getElement(elementId).innerText();
    return { text };
  },

  'element.html': async (args) => {
    const [elementId] = args as [number];
    const html = await getElement(elementId).html();
    return { html };
  },

  'element.value': async (args) => {
    const [elementId] = args as [number];
    const value = await getElement(elementId).value();
    return { value };
  },

  'element.attr': async (args) => {
    const [elementId, name] = args as [number, string];
    const value = await getElement(elementId).attr(name);
    return { value };
  },

  'element.getAttribute': async (args) => {
    const [elementId, name] = args as [number, string];
    const value = await getElement(elementId).getAttribute(name);
    return { value };
  },

  'element.bounds': async (args) => {
    const [elementId] = args as [number];
    const box = await getElement(elementId).bounds();
    return { box };
  },

  'element.boundingBox': async (args) => {
    const [elementId] = args as [number];
    const box = await getElement(elementId).boundingBox();
    return { box };
  },

  'element.isVisible': async (args) => {
    const [elementId] = args as [number];
    const visible = await getElement(elementId).isVisible();
    return { visible };
  },

  'element.isHidden': async (args) => {
    const [elementId] = args as [number];
    const hidden = await getElement(elementId).isHidden();
    return { hidden };
  },

  'element.isEnabled': async (args) => {
    const [elementId] = args as [number];
    const enabled = await getElement(elementId).isEnabled();
    return { enabled };
  },

  'element.isChecked': async (args) => {
    const [elementId] = args as [number];
    const checked = await getElement(elementId).isChecked();
    return { checked };
  },

  'element.isEditable': async (args) => {
    const [elementId] = args as [number];
    const editable = await getElement(elementId).isEditable();
    return { editable };
  },

  'element.role': async (args) => {
    const [elementId] = args as [number];
    const role = await getElement(elementId).role();
    return { role };
  },

  'element.label': async (args) => {
    const [elementId] = args as [number];
    const label = await getElement(elementId).label();
    return { label };
  },

  'element.eval': async (args) => {
    const [elementId, fn] = args as [number, string];
    const value = await getElement(elementId).eval(fn);
    return { value };
  },

  'element.screenshot': async (args) => {
    const [elementId] = args as [number];
    const buffer = await getElement(elementId).screenshot();
    return { data: buffer.toString('base64') };
  },

  'element.waitUntil': async (args) => {
    const [elementId, state, options] = args as [number, string | undefined, any];
    await getElement(elementId).waitUntil(state, options);
    return { success: true };
  },

  'element.find': async (args) => {
    const [elementId, selector, options] = args as [number, string | SelectorOptions, any];
    const child = await getElement(elementId).find(selector, options);
    const childId = storeElement(child);
    return { elementId: childId, info: child.info };
  },

  'element.findAll': async (args) => {
    const [elementId, selector, options] = args as [number, string | SelectorOptions, any];
    const list = await getElement(elementId).findAll(selector, options);
    const listId = storeElementList(list);
    const elementIds: number[] = [];
    for (const el of list.toArray()) {
      elementIds.push(storeElement(el));
    }
    return { listId, elementIds, count: list.count() };
  },

  // ========================
  // ElementList commands
  // ========================

  'elementList.count': async (args) => {
    const [listId] = args as [number];
    return { count: getElementList(listId).count() };
  },

  'elementList.first': async (args) => {
    const [listId] = args as [number];
    const el = getElementList(listId).first();
    const id = storeElement(el);
    return { elementId: id, info: el.info };
  },

  'elementList.last': async (args) => {
    const [listId] = args as [number];
    const el = getElementList(listId).last();
    const id = storeElement(el);
    return { elementId: id, info: el.info };
  },

  'elementList.nth': async (args) => {
    const [listId, index] = args as [number, number];
    const el = getElementList(listId).nth(index);
    const id = storeElement(el);
    return { elementId: id, info: el.info };
  },

  'elementList.filter': async (args) => {
    const [listId, opts] = args as [number, FilterOptions];
    const filtered = await getElementList(listId).filter(opts);
    const newListId = storeElementList(filtered);
    const elementIds: number[] = [];
    for (const el of filtered.toArray()) {
      elementIds.push(storeElement(el));
    }
    return { listId: newListId, elementIds, count: filtered.count() };
  },

  'elementList.toArray': async (args) => {
    const [listId] = args as [number];
    const arr = getElementList(listId).toArray();
    const elementIds: number[] = [];
    for (const el of arr) {
      elementIds.push(storeElement(el));
    }
    return { elementIds };
  },

  // ========================
  // Quit
  // ========================

  'quit': async () => {
    if (!browserInstance) throw new Error('Browser not launched');
    await browserInstance.close();
    browserInstance = null;
    pages.clear();
    contexts.clear();
    elements.clear();
    elementLists.clear();
    return { success: true };
  },
};

async function handleCommand(cmd: Command): Promise<unknown> {
  const handler = handlers[cmd.method];
  if (!handler) {
    throw new Error(`Unknown method: ${cmd.method}`);
  }
  return handler(cmd.args);
}

parentPort!.on('message', async ({ cmd, port }: { cmd: Command; port: MessagePort }) => {
  let result: unknown;
  let error: string | null = null;

  try {
    result = await handleCommand(cmd);
  } catch (err) {
    error = err instanceof Error ? err.message : String(err);
  }

  // Deliver buffered events BEFORE the result.
  // Events are buffered during command processing (e.g. contextCreated fires
  // during newPage, or network events fire during navigate/evaluate).
  // We deliver them here, after the command completes, to avoid
  // a signal race between callback (signal=2) and result (signal=1).

  // Yield so any pending BiDi events are processed
  if (onPageHandlerId || onPopupHandlerId || networkEventBuffer.length > 0) {
    await new Promise(r => setImmediate(r));
  }

  // Drain page/popup events
  while (pageEventBuffer.length > 0) {
    const event = pageEventBuffer.shift()!;
    const handlerId = event.type === 'page' ? onPageHandlerId : onPopupHandlerId;
    if (handlerId) {
      await invokeMainThread(handlerId, { pageId: event.pageId });
    }
  }

  // Drain network/download/websocket events
  while (networkEventBuffer.length > 0) {
    const event = networkEventBuffer.shift()!;
    await invokeMainThread(event.handlerId, event.data);
  }

  port.postMessage({ result, error });

  Atomics.store(signal, 0, 1);
  Atomics.notify(signal, 0);
});
