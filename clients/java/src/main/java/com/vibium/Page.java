package com.vibium;

import com.google.gson.Gson;
import com.google.gson.JsonArray;
import com.google.gson.JsonElement;
import com.google.gson.JsonObject;
import com.vibium.internal.BiDiClient;
import com.vibium.types.*;

import java.util.*;
import java.util.concurrent.CopyOnWriteArrayList;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;
import java.util.function.Consumer;

/**
 * Represents a browser tab. The primary interface for page automation.
 */
public class Page {

    private static final Gson GSON = new Gson();
    private static final ExecutorService NETWORK_CALLBACKS = Executors.newCachedThreadPool(r -> {
        Thread thread = new Thread(r, "vibium-network-callback");
        thread.setDaemon(true);
        return thread;
    });

    private final BiDiClient client;
    private final String contextId;
    private final BrowserContext browserContext;

    // Sub-objects
    private final Keyboard keyboard;
    private final Mouse mouse;
    private final Touch touch;
    private final Clock clock;
    private final Capture capture;

    // Event listeners
    private final CopyOnWriteArrayList<Consumer<Request>> requestListeners = new CopyOnWriteArrayList<>();
    private final CopyOnWriteArrayList<Consumer<Response>> responseListeners = new CopyOnWriteArrayList<>();
    private final CopyOnWriteArrayList<Consumer<Dialog>> dialogListeners = new CopyOnWriteArrayList<>();
    private final CopyOnWriteArrayList<Consumer<ConsoleMessage>> consoleListeners = new CopyOnWriteArrayList<>();
    private final CopyOnWriteArrayList<Consumer<String>> errorListeners = new CopyOnWriteArrayList<>();
    private final CopyOnWriteArrayList<Consumer<Download>> downloadListeners = new CopyOnWriteArrayList<>();
    private final CopyOnWriteArrayList<Consumer<WebSocketInfo>> webSocketListeners = new CopyOnWriteArrayList<>();
    private final CopyOnWriteArrayList<Consumer<String>> navigationListeners = new CopyOnWriteArrayList<>();
    private final Map<Integer, WebSocketInfo> webSockets = new java.util.concurrent.ConcurrentHashMap<>();
    private final Object webSocketSetupLock = new Object();
    private volatile boolean webSocketMonitorInstalled = false;

    // Buffered events
    private final List<ConsoleMessage> bufferedConsole = Collections.synchronizedList(new ArrayList<>());
    private final List<String> bufferedErrors = Collections.synchronizedList(new ArrayList<>());

    // Network routes
    private final List<RouteEntry> routes = new CopyOnWriteArrayList<>();
    private final Object dataCollectorLock = new Object();
    private String dataCollectorId;

    // Active downloads keyed by navigation ID
    private final Map<String, Download> activeDownloads = Collections.synchronizedMap(new HashMap<>());

    // Event handler reference for cleanup
    private final Consumer<JsonObject> eventHandler;

    Page(BiDiClient client, String contextId, BrowserContext browserContext) {
        this.client = client;
        this.contextId = contextId;
        this.browserContext = browserContext;
        this.keyboard = new Keyboard(client, contextId);
        this.mouse = new Mouse(client, contextId);
        this.touch = new Touch(client, contextId);
        this.clock = new Clock(client, contextId);
        this.capture = new Capture(this);

        // Register event handler
        this.eventHandler = this::handleEvent;
        client.onEvent(eventHandler);
    }

    // ── Properties ──────────────────────────────────────────────

    /** Get the browsing context ID. */
    public String id() { return contextId; }

    /** Get the Keyboard for this page. */
    public Keyboard keyboard() { return keyboard; }

    /** Get the Mouse for this page. */
    public Mouse mouse() { return mouse; }

    /** Get the Touch for this page. */
    public Touch touch() { return touch; }

    /** Get the Clock for this page. */
    public Clock clock() { return clock; }

    /** Get the one-shot event capture helpers for this page. */
    public Capture capture() { return capture; }

    /** Get the parent BrowserContext. */
    public BrowserContext context() { return browserContext; }

    // ── Navigation ──────────────────────────────────────────────

    /** Navigate to a URL. */
    public void go(String url) {
        JsonObject params = contextParams();
        params.addProperty("url", url);
        client.send("vibium:page.navigate", params);
    }

    /** Go back in history. */
    public void back() {
        client.send("vibium:page.back", contextParams());
    }

    /** Go forward in history. */
    public void forward() {
        client.send("vibium:page.forward", contextParams());
    }

    /** Reload the page. */
    public void reload() {
        client.send("vibium:page.reload", contextParams());
    }

    // ── Page Info ───────────────────────────────────────────────

    /** Get the current URL. */
    public String url() {
        JsonObject result = client.send("vibium:page.url", contextParams());
        return result.get("url").getAsString();
    }

    /** Get the page title. */
    public String title() {
        JsonObject result = client.send("vibium:page.title", contextParams());
        return result.get("title").getAsString();
    }

    /** Get the full page HTML. */
    public String content() {
        JsonObject result = client.send("vibium:page.content", contextParams());
        return result.get("content").getAsString();
    }

    // ── Finding Elements ────────────────────────────────────────

    /** Find a single element by CSS selector. */
    public Element find(String selector) {
        return find(selector, (FindOptions) null);
    }

    /** Find a single element by CSS selector with options. */
    public Element find(String selector, FindOptions options) {
        JsonObject params = contextParams();
        params.addProperty("selector", selector);
        if (options != null && options.timeout() != null) {
            params.addProperty("timeout", options.timeout());
        }
        JsonObject result = client.send("vibium:page.find", params);
        return elementFromResult(result, selector, 0);
    }

    /** Find a single element by semantic selector. */
    public Element find(SelectorOptions options) {
        JsonObject params = contextParams();
        for (Map.Entry<String, Object> entry : options.toParams().entrySet()) {
            params.add(entry.getKey(), GSON.toJsonTree(entry.getValue()));
        }
        JsonObject result = client.send("vibium:page.find", params);
        return elementFromResult(result, "", 0, locatorParams(options));
    }

    /**
     * Find all matching elements by CSS selector. Waits up to the timeout
     * for at least one match, then returns an empty list if there is none.
     */
    public List<Element> findAll(String selector) {
        return findAll(selector, (FindOptions) null);
    }

    /**
     * Find all matching elements by CSS selector with options. Waits up to
     * the timeout for at least one match, then returns an empty list if
     * there is none. A timeout of 0 checks once without waiting.
     */
    public List<Element> findAll(String selector, FindOptions options) {
        JsonObject params = contextParams();
        params.addProperty("selector", selector);
        if (options != null && options.timeout() != null) {
            params.addProperty("timeout", options.timeout());
        }
        JsonObject result = client.send("vibium:page.findAll", params);
        return elementsFromResult(result, selector);
    }

    /**
     * Find all matching elements by semantic selector. Waits up to the
     * timeout for at least one match, then returns an empty list if there
     * is none.
     */
    public List<Element> findAll(SelectorOptions options) {
        JsonObject params = contextParams();
        for (Map.Entry<String, Object> entry : options.toParams().entrySet()) {
            params.add(entry.getKey(), GSON.toJsonTree(entry.getValue()));
        }
        JsonObject result = client.send("vibium:page.findAll", params);
        return elementsFromResult(result, "", locatorParams(options));
    }

    // ── Screenshots & PDF ───────────────────────────────────────

    /** Take a screenshot, returns PNG bytes. */
    public byte[] screenshot() {
        return screenshot(null);
    }

    /** Take a screenshot with options, returns PNG bytes. */
    public byte[] screenshot(ScreenshotOptions options) {
        JsonObject params = contextParams();
        if (options != null) {
            if (options.fullPage() != null) params.addProperty("fullPage", options.fullPage());
            if (options.clip() != null) {
                JsonObject clip = new JsonObject();
                clip.addProperty("x", options.clip().x());
                clip.addProperty("y", options.clip().y());
                clip.addProperty("width", options.clip().width());
                clip.addProperty("height", options.clip().height());
                params.add("clip", clip);
            }
        }
        JsonObject result = client.send("vibium:page.screenshot", params);
        String data = result.get("data").getAsString();
        return Base64.getDecoder().decode(data);
    }

    /** Generate a PDF, returns PDF bytes (headless only). */
    public byte[] pdf() {
        JsonObject result = client.send("vibium:page.pdf", contextParams());
        String data = result.get("data").getAsString();
        return Base64.getDecoder().decode(data);
    }

    // ── JavaScript Evaluation ───────────────────────────────────

    /** Evaluate a JavaScript expression. */
    public Object evaluate(String expression) {
        JsonObject params = contextParams();
        params.addProperty("expression", expression);
        JsonObject result = client.send("vibium:page.eval", params);
        if (result.has("value")) {
            return jsonToJava(result.get("value"));
        }
        return null;
    }

    /** Add a script tag to the page. Pass a URL or inline JavaScript. */
    public void addScript(String source) {
        client.send("vibium:page.addScript", sourceParams(source));
    }

    /** Add a style tag to the page. Pass a URL or inline CSS. */
    public void addStyle(String source) {
        client.send("vibium:page.addStyle", sourceParams(source));
    }

    /**
     * Build params for addScript/addStyle. The engine expects either a "url" or
     * a "content" key (not "source"); send the right one based on the input
     * (issue #130).
     */
    private JsonObject sourceParams(String source) {
        JsonObject params = contextParams();
        boolean isUrl = source.startsWith("http://") || source.startsWith("https://") || source.startsWith("//");
        params.addProperty(isUrl ? "url" : "content", source);
        return params;
    }

    /**
     * Define window[name] in the page from JS function source, in the current
     * document and every later one, matching the JS and Python clients:
     * page.expose("double", "(n) => n * 2"). This replaces a callback-taking
     * overload that sent no fn, which the server rejected on every call, and
     * subscribed to an event the server never emits. Host-side callbacks,
     * where the page calls back into the Java program, are tracked in #298.
     */
    public void expose(String name, String fn) {
        JsonObject params = contextParams();
        params.addProperty("name", name);
        params.addProperty("fn", fn);
        client.send("vibium:page.expose", params);
    }

    // ── Waiting ─────────────────────────────────────────────────

    /** Wait for a fixed duration in milliseconds. */
    public void sleep(long ms) {
        JsonObject params = contextParams();
        params.addProperty("ms", ms);
        client.send("vibium:page.wait", params);
    }

    /** Wait for an element matching the selector. */
    public Element waitFor(String selector) {
        return waitFor(selector, (FindOptions) null);
    }

    /** Wait for an element matching the selector with options. */
    public Element waitFor(String selector, FindOptions options) {
        JsonObject params = contextParams();
        params.addProperty("selector", selector);
        if (options != null && options.timeout() != null) {
            params.addProperty("timeout", options.timeout());
        }
        JsonObject result = client.send("vibium:page.waitFor", params);
        return elementFromResult(result, selector, 0);
    }

    /** Wait for a JS function to return truthy. */
    public Object waitForFunction(String fn) {
        return waitForFunction(fn, null);
    }

    /** Wait for a JS function to return truthy with options. */
    public Object waitForFunction(String fn, WaitOptions options) {
        JsonObject params = contextParams();
        params.addProperty("fn", fn);
        if (options != null && options.timeout() != null) {
            params.addProperty("timeout", options.timeout());
        }
        JsonObject result = client.send("vibium:page.waitForFunction", params);
        if (result.has("value")) {
            return jsonToJava(result.get("value"));
        }
        return null;
    }

    /** Wait for URL to match a pattern. */
    public void waitForURL(String pattern) {
        waitForURL(pattern, null);
    }

    /** Wait for URL to match a pattern with options. */
    public void waitForURL(String pattern, WaitOptions options) {
        JsonObject params = contextParams();
        params.addProperty("pattern", pattern);
        if (options != null && options.timeout() != null) {
            params.addProperty("timeout", options.timeout());
        }
        client.send("vibium:page.waitForURL", params);
    }

    /** Wait for page load. */
    public void waitForLoad() {
        waitForLoad(null);
    }

    /** Wait for page load with options. */
    public void waitForLoad(WaitOptions options) {
        JsonObject params = contextParams();
        if (options != null && options.timeout() != null) {
            params.addProperty("timeout", options.timeout());
        }
        client.send("vibium:page.waitForLoad", params);
    }

    // ── Viewport & Window ───────────────────────────────────────

    /** Set the viewport size. */
    public void setViewport(ViewportSize size) {
        JsonObject params = contextParams();
        params.addProperty("width", size.width());
        params.addProperty("height", size.height());
        client.send("vibium:page.setViewport", params);
    }

    /** Get the viewport size. */
    public ViewportSize viewport() {
        JsonObject result = client.send("vibium:page.viewport", contextParams());
        int width = result.get("width").getAsInt();
        int height = result.get("height").getAsInt();
        return new ViewportSize(width, height);
    }

    /** Set window size/position. */
    public void setWindow(WindowOptions options) {
        JsonObject params = contextParams();
        for (Map.Entry<String, Object> entry : options.toParams().entrySet()) {
            params.add(entry.getKey(), GSON.toJsonTree(entry.getValue()));
        }
        client.send("vibium:page.setWindow", params);
    }

    /** Get window info. */
    public WindowInfo window() {
        JsonObject result = client.send("vibium:page.window", contextParams());
        return GSON.fromJson(result, WindowInfo.class);
    }

    // ── Emulation ───────────────────────────────────────────────

    /** Override CSS media features. */
    public void emulateMedia(MediaOptions options) {
        JsonObject params = contextParams();
        for (Map.Entry<String, Object> entry : options.toParams().entrySet()) {
            params.add(entry.getKey(), GSON.toJsonTree(entry.getValue()));
        }
        client.send("vibium:page.emulateMedia", params);
    }

    /** Set page HTML content. */
    public void setContent(String html) {
        JsonObject params = contextParams();
        params.addProperty("html", html);
        client.send("vibium:page.setContent", params);
    }

    /** Override geolocation. */
    public void setGeolocation(GeoCoords coords) {
        JsonObject params = contextParams();
        params.addProperty("latitude", coords.latitude());
        params.addProperty("longitude", coords.longitude());
        if (coords.accuracy() != null) {
            params.addProperty("accuracy", coords.accuracy());
        }
        client.send("vibium:page.setGeolocation", params);
    }

    // ── Accessibility ───────────────────────────────────────────

    /** Get the accessibility tree. */
    public A11yNode a11yTree() {
        return a11yTree(null);
    }

    /** Get the accessibility tree with options. */
    public A11yNode a11yTree(A11yOptions options) {
        JsonObject params = contextParams();
        if (options != null) {
            for (Map.Entry<String, Object> entry : options.toParams().entrySet()) {
                params.add(entry.getKey(), GSON.toJsonTree(entry.getValue()));
            }
        }
        JsonObject result = client.send("vibium:page.a11yTree", params);
        JsonObject tree = result.has("tree") ? result.getAsJsonObject("tree") : result;
        return GSON.fromJson(tree, A11yNode.class);
    }

    // ── Frames ──────────────────────────────────────────────────

    /** List all frames. */
    public List<Page> frames() {
        JsonObject result = client.send("vibium:page.frames", contextParams());
        JsonArray arr = result.getAsJsonArray("frames");
        List<Page> pages = new ArrayList<>();
        for (JsonElement el : arr) {
            String frameId = el.getAsJsonObject().get("context").getAsString();
            pages.add(new Page(client, frameId, browserContext));
        }
        return pages;
    }

    /** Get a frame by name or URL. */
    public Page frame(String nameOrUrl) {
        JsonObject params = contextParams();
        params.addProperty("nameOrUrl", nameOrUrl);
        JsonObject result = client.send("vibium:page.frame", params);
        String frameId = result.get("context").getAsString();
        return new Page(client, frameId, browserContext);
    }

    /** Get the main frame (self for top-level pages). */
    public Page mainFrame() {
        return this;
    }

    // ── Scrolling ───────────────────────────────────────────────

    /** Scroll the page (default: down). */
    public void scroll() {
        scroll(null);
    }

    /** Scroll the page with options. */
    public void scroll(ScrollOptions options) {
        JsonObject params = contextParams();
        if (options != null) {
            if (options.direction() != null) params.addProperty("direction", options.direction());
            if (options.amount() != null) params.addProperty("amount", options.amount());
            if (options.selector() != null) params.addProperty("selector", options.selector());
        }
        client.send("vibium:page.scroll", params);
    }

    // ── Lifecycle ───────────────────────────────────────────────

    /** Bring this page/tab to the front. */
    public void bringToFront() {
        JsonObject params = new JsonObject();
        params.addProperty("context", contextId);
        client.send("browsingContext.activate", params);
    }

    /** Close this page/tab. */
    public void close() {
        JsonObject params = new JsonObject();
        params.addProperty("context", contextId);
        client.send("browsingContext.close", params);
        teardownDataCollector();
        webSockets.clear();
        client.offEvent(eventHandler);
    }

    // ── HTTP Headers ────────────────────────────────────────────

    /** Set extra HTTP headers for this page. */
    public void setHeaders(Map<String, String> headers) {
        JsonObject params = contextParams();
        params.add("headers", GSON.toJsonTree(headers));
        client.send("vibium:page.setHeaders", params);
    }

    // ── Network Interception ────────────────────────────────────

    /** Register a route handler for URL pattern matching. */
    public void route(String pattern, Consumer<Route> handler) {
        ensureDataCollector();
        // Subscribe to network events if this is the first route
        if (routes.isEmpty()) {
            JsonObject params = contextParams();
            client.send("vibium:page.route", params);
        }

        routes.add(new RouteEntry(pattern, handler));
    }

    /** Remove a route handler. */
    public void unroute(String pattern) {
        routes.removeIf(r -> r.pattern.equals(pattern));

        if (routes.isEmpty()) {
            try {
                JsonObject params = contextParams();
                params.addProperty("pattern", pattern);
                client.send("network.removeIntercept", params);
            } catch (Exception ignored) {}
            if (requestListeners.isEmpty() && responseListeners.isEmpty()) {
                teardownDataCollector();
            }
        }
    }

    // ── Event Listeners ─────────────────────────────────────────

    /** Listen for network requests. */
    public void onRequest(Consumer<Request> callback) {
        ensureDataCollector();
        requestListeners.add(callback);
    }

    /** Listen for network responses. */
    public void onResponse(Consumer<Response> callback) {
        ensureDataCollector();
        responseListeners.add(callback);
    }

    /** Listen for dialogs. */
    public void onDialog(Consumer<Dialog> callback) {
        dialogListeners.add(callback);
    }

    /** Listen for console messages. */
    public void onConsole(Consumer<ConsoleMessage> callback) {
        consoleListeners.add(callback);
    }

    /**
     * Collect console messages into a buffer (retrieve with consoleMessages()).
     */
    public void collectConsole() {
        consoleListeners.add(msg -> bufferedConsole.add(msg));
    }

    /** Listen for page errors. */
    public void onError(Consumer<String> callback) {
        errorListeners.add(callback);
    }

    /**
     * Collect errors into a buffer (retrieve with errors()).
     */
    public void collectErrors() {
        errorListeners.add(err -> bufferedErrors.add(err));
    }

    /** Listen for downloads. */
    public void onDownload(Consumer<Download> callback) {
        downloadListeners.add(callback);
    }

    /** Listen for WebSocket connections. */
    public void onWebSocket(Consumer<WebSocketInfo> callback) {
        webSocketListeners.add(callback);
        if (webSocketMonitorInstalled) return;

        synchronized (webSocketSetupLock) {
            if (webSocketMonitorInstalled) return;
            try {
                // Synchronous by design: the monitor is installed before the
                // caller can issue the command that opens the socket (#351).
                client.send("vibium:page.onWebSocket", contextParams());
                webSocketMonitorInstalled = true;
            } catch (RuntimeException error) {
                webSocketListeners.remove(callback);
                throw error;
            }
        }
    }

    /** Get buffered console messages. */
    public List<ConsoleMessage> consoleMessages() {
        return new ArrayList<>(bufferedConsole);
    }

    /** Get buffered page errors. */
    public List<String> errors() {
        return new ArrayList<>(bufferedErrors);
    }

    /** Remove all event listeners, optionally by event name. */
    public void removeAllListeners(String event) {
        if (event == null) {
            requestListeners.clear();
            responseListeners.clear();
            dialogListeners.clear();
            consoleListeners.clear();
            errorListeners.clear();
            downloadListeners.clear();
            webSocketListeners.clear();
            navigationListeners.clear();
            if (routes.isEmpty()) teardownDataCollector();
            return;
        }
        switch (event) {
            case "request": requestListeners.clear(); break;
            case "response": responseListeners.clear(); break;
            case "dialog": dialogListeners.clear(); break;
            case "console": consoleListeners.clear(); break;
            case "error": errorListeners.clear(); break;
            case "download": downloadListeners.clear(); break;
            case "websocket": webSocketListeners.clear(); break;
            case "navigation": navigationListeners.clear(); break;
        }
        if (requestListeners.isEmpty() && responseListeners.isEmpty() && routes.isEmpty()) {
            teardownDataCollector();
        }
    }

    /** Remove all event listeners. */
    public void removeAllListeners() {
        removeAllListeners(null);
    }

    // ── Internal ────────────────────────────────────────────────

    BiDiClient getClient() { return client; }

    private void ensureDataCollector() {
        synchronized (dataCollectorLock) {
            if (dataCollectorId != null) return;
            JsonObject params = new JsonObject();
            JsonArray dataTypes = new JsonArray();
            dataTypes.add("request");
            dataTypes.add("response");
            params.add("dataTypes", dataTypes);
            params.addProperty("maxEncodedDataSize", 10 * 1024 * 1024);
            JsonObject result = client.send("network.addDataCollector", params);
            dataCollectorId = result.has("collector")
                ? result.get("collector").getAsString()
                : null;
        }
    }

    private void teardownDataCollector() {
        synchronized (dataCollectorLock) {
            if (dataCollectorId == null) return;
            JsonObject params = new JsonObject();
            params.addProperty("collector", dataCollectorId);
            dataCollectorId = null;
            try {
                client.send("network.removeDataCollector", params);
            } catch (Exception ignored) {
                // The collector disappears with the browser connection.
            }
        }
    }

    private JsonObject contextParams() {
        JsonObject params = new JsonObject();
        params.addProperty("context", contextId);
        return params;
    }

    private Element elementFromResult(JsonObject result, String selector, int index) {
        return elementFromResult(result, selector, index, null);
    }

    private Element elementFromResult(JsonObject result, String selector, int index,
                                      Map<String, Object> locator) {
        ElementInfo info = parseElementInfo(result);
        return new Element(client, contextId, selector, index, info, locator);
    }

    /** The locator an element was found by, minus per-call options. */
    private static Map<String, Object> locatorParams(SelectorOptions options) {
        Map<String, Object> p = new LinkedHashMap<>(options.toParams());
        p.remove("timeout");
        return p;
    }

    private List<Element> elementsFromResult(JsonObject result, String selector) {
        return elementsFromResult(result, selector, null);
    }

    private List<Element> elementsFromResult(JsonObject result, String selector,
                                             Map<String, Object> locator) {
        List<Element> elements = new ArrayList<>();
        JsonArray arr = result.has("elements") ? result.getAsJsonArray("elements") : new JsonArray();
        for (int i = 0; i < arr.size(); i++) {
            JsonObject el = arr.get(i).getAsJsonObject();
            ElementInfo info = parseElementInfo(el);
            elements.add(new Element(client, contextId, selector, i, info, locator));
        }
        return elements;
    }

    private ElementInfo parseElementInfo(JsonObject obj) {
        String tag = obj.has("tag") ? obj.get("tag").getAsString() : "";
        String text = obj.has("text") ? obj.get("text").getAsString() : "";
        BoundingBox box = null;
        if (obj.has("box") && obj.get("box").isJsonObject()) {
            JsonObject b = obj.getAsJsonObject("box");
            box = new BoundingBox(
                b.get("x").getAsDouble(),
                b.get("y").getAsDouble(),
                b.get("width").getAsDouble(),
                b.get("height").getAsDouble()
            );
        }
        return new ElementInfo(tag, text, box);
    }

    static Object jsonToJava(JsonElement el) {
        if (el == null || el.isJsonNull()) return null;
        if (el.isJsonPrimitive()) {
            if (el.getAsJsonPrimitive().isBoolean()) return el.getAsBoolean();
            if (el.getAsJsonPrimitive().isNumber()) {
                double d = el.getAsDouble();
                if (d == Math.floor(d) && !Double.isInfinite(d)) {
                    long l = (long) d;
                    if (l >= Integer.MIN_VALUE && l <= Integer.MAX_VALUE) return (int) l;
                    return l;
                }
                return d;
            }
            return el.getAsString();
        }
        if (el.isJsonArray()) {
            List<Object> list = new ArrayList<>();
            for (JsonElement item : el.getAsJsonArray()) {
                list.add(jsonToJava(item));
            }
            return list;
        }
        if (el.isJsonObject()) {
            Map<String, Object> map = new LinkedHashMap<>();
            for (Map.Entry<String, JsonElement> entry : el.getAsJsonObject().entrySet()) {
                map.put(entry.getKey(), jsonToJava(entry.getValue()));
            }
            return map;
        }
        return el.toString();
    }

    private void handleEvent(JsonObject event) {
        String method = event.has("method") ? event.get("method").getAsString() : "";
        JsonObject params = event.has("params") ? event.getAsJsonObject("params") : new JsonObject();

        // Only handle events for this page's context
        String eventContext = params.has("context") ? params.get("context").getAsString() : "";
        if (!contextId.equals(eventContext) && !eventContext.isEmpty()) {
            return;
        }

        switch (method) {
            case "network.beforeRequestSent":
                handleRequestEvent(params);
                break;
            case "network.responseCompleted":
                handleResponseEvent(params);
                break;
            case "browsingContext.userPromptOpened":
                handleDialogEvent(params);
                break;
            case "log.entryAdded":
                // BiDi log.entryAdded carries both console output and uncaught
                // JS errors, distinguished by "type" ("console" vs "javascript").
                // Routing everything to the console handler meant onError() never
                // fired (issue #136).
                if ("javascript".equals(params.has("type") ? params.get("type").getAsString() : "")) {
                    handleErrorEvent(params);
                } else {
                    handleConsoleEvent(params);
                }
                break;
            case "browsingContext.downloadWillBegin":
                handleDownloadStarted(params);
                break;
            case "browsingContext.downloadEnd":
                handleDownloadCompleted(params);
                break;
            case "vibium:ws.created":
                handleWebSocketCreated(params);
                break;
            case "vibium:ws.message":
                handleWebSocketMessage(params);
                break;
            case "vibium:ws.closed":
                handleWebSocketClosed(params);
                break;
            case "browsingContext.load":
            case "browsingContext.fragmentNavigated":
            case "browsingContext.historyUpdated":
                handleNavigationEvent(params);
                break;
        }
    }

    private void handleRequestEvent(JsonObject params) {
        boolean isBlocked = params.has("isBlocked") && params.get("isBlocked").getAsBoolean();
        if (isBlocked) {
            handleBlockedRequest(params);
            return;
        }
        if (requestListeners.isEmpty()) return;
        Request request = new Request(client, params);
        for (Consumer<Request> listener : requestListeners) {
            NETWORK_CALLBACKS.execute(() -> {
                try { listener.accept(request); } catch (Exception ignored) {}
            });
        }
    }

    private void handleBlockedRequest(JsonObject params) {
        Request request = new Request(client, params, true);
        String requestId = request.requestId();

        for (RouteEntry entry : routes) {
            if (matchPattern(entry.pattern, request.url())) {
                Route route = new Route(client, contextId, requestId, request);
                NETWORK_CALLBACKS.execute(() -> {
                    try {
                        entry.handler.accept(route);
                    } catch (Exception ignored) {}
                });
                return;
            }
        }

        // No matching route — continue the request so the page does not hang
        NETWORK_CALLBACKS.execute(() -> {
            try {
                JsonObject continueParams = new JsonObject();
                continueParams.addProperty("request", requestId);
                client.send("vibium:network.continue", continueParams);
            } catch (Exception ignored) {}
        });
    }

    private void handleResponseEvent(JsonObject params) {
        if (responseListeners.isEmpty()) return;
        Response response = new Response(client, params);
        for (Consumer<Response> listener : responseListeners) {
            NETWORK_CALLBACKS.execute(() -> {
                try { listener.accept(response); } catch (Exception ignored) {}
            });
        }
    }

    private void handleDialogEvent(JsonObject params) {
        Dialog dialog = new Dialog(client, params);
        if (dialogListeners.isEmpty()) {
            // Auto-dismiss if no handler
            try { dialog.dismiss(); } catch (Exception ignored) {}
            return;
        }
        for (Consumer<Dialog> listener : dialogListeners) {
            try { listener.accept(dialog); } catch (Exception ignored) {}
        }
    }

    private void handleConsoleEvent(JsonObject params) {
        if (consoleListeners.isEmpty()) return;
        ConsoleMessage msg = new ConsoleMessage(params);
        for (Consumer<ConsoleMessage> listener : consoleListeners) {
            try { listener.accept(msg); } catch (Exception ignored) {}
        }
    }

    private void handleErrorEvent(JsonObject params) {
        if (errorListeners.isEmpty()) return;
        String text = params.has("text") ? params.get("text").getAsString() : "";
        for (Consumer<String> listener : errorListeners) {
            try { listener.accept(text); } catch (Exception ignored) {}
        }
    }

    private void handleDownloadStarted(JsonObject params) {
        if (downloadListeners.isEmpty()) return;
        Download download = new Download(client, params);
        String navId = params.has("navigation") ? params.get("navigation").getAsString() : "";
        if (!navId.isEmpty()) {
            activeDownloads.put(navId, download);
        }
        for (Consumer<Download> listener : downloadListeners) {
            try { listener.accept(download); } catch (Exception ignored) {}
        }
    }

    private void handleDownloadCompleted(JsonObject params) {
        String navId = params.has("navigation") ? params.get("navigation").getAsString() : "";
        Download download = activeDownloads.remove(navId);
        if (download != null) {
            String status = params.has("status") ? params.get("status").getAsString() : "complete";
            String path = params.has("filepath") ? params.get("filepath").getAsString() : null;
            download.complete(status, path);
        }
    }

    private void handleWebSocketCreated(JsonObject params) {
        if (!params.has("id")) return;
        int id = params.get("id").getAsInt();
        WebSocketInfo info = new WebSocketInfo(params);
        webSockets.put(id, info);
        for (Consumer<WebSocketInfo> listener : webSocketListeners) {
            try { listener.accept(info); } catch (Exception ignored) {}
        }
    }

    private void handleWebSocketMessage(JsonObject params) {
        if (!params.has("id")) return;
        WebSocketInfo info = webSockets.get(params.get("id").getAsInt());
        if (info == null) return;
        String data = params.has("data") ? params.get("data").getAsString() : "";
        String direction = params.has("direction") ? params.get("direction").getAsString() : "";
        info.emitMessage(data, direction);
    }

    private void handleWebSocketClosed(JsonObject params) {
        if (!params.has("id")) return;
        WebSocketInfo info = webSockets.remove(params.get("id").getAsInt());
        if (info == null) return;
        Integer code = params.has("code") ? params.get("code").getAsInt() : null;
        String reason = params.has("reason") ? params.get("reason").getAsString() : null;
        info.emitClose(code, reason);
    }

    private void handleNavigationEvent(JsonObject params) {
        String url = params.has("url") ? params.get("url").getAsString() : "";
        if (url.isEmpty()) return;
        for (Consumer<String> listener : navigationListeners) {
            try { listener.accept(url); } catch (Exception ignored) {}
        }
    }

    void addRequestListener(Consumer<Request> listener) { requestListeners.add(listener); }
    void removeRequestListener(Consumer<Request> listener) { requestListeners.remove(listener); }
    void addResponseListener(Consumer<Response> listener) { responseListeners.add(listener); }
    void removeResponseListener(Consumer<Response> listener) { responseListeners.remove(listener); }
    void addNavigationListener(Consumer<String> listener) { navigationListeners.add(listener); }
    void removeNavigationListener(Consumer<String> listener) { navigationListeners.remove(listener); }
    void addDownloadListener(Consumer<Download> listener) { downloadListeners.add(listener); }
    void removeDownloadListener(Consumer<Download> listener) { downloadListeners.remove(listener); }
    void addDialogListener(Consumer<Dialog> listener) { dialogListeners.add(listener); }
    void removeDialogListener(Consumer<Dialog> listener) { dialogListeners.remove(listener); }
    void addConsoleListener(Consumer<ConsoleMessage> listener) { consoleListeners.add(listener); }
    void removeConsoleListener(Consumer<ConsoleMessage> listener) { consoleListeners.remove(listener); }
    void addErrorListener(Consumer<String> listener) { errorListeners.add(listener); }
    void removeErrorListener(Consumer<String> listener) { errorListeners.remove(listener); }

    static boolean matchPattern(String pattern, String url) {
        if (pattern == null || pattern.isEmpty()) return true;
        if (pattern.equals("**") || pattern.equals("**/*")) return true;

        // Convert glob to regex
        StringBuilder regex = new StringBuilder();
        for (int i = 0; i < pattern.length(); i++) {
            char c = pattern.charAt(i);
            if (c == '*' && i + 1 < pattern.length() && pattern.charAt(i + 1) == '*') {
                regex.append(".*");
                i++; // skip second *
                if (i + 1 < pattern.length() && pattern.charAt(i + 1) == '/') {
                    i++; // skip /
                }
            } else if (c == '*') {
                regex.append("[^/]*");
            } else if (c == '?' || c == '.' || c == '(' || c == ')' || c == '[' || c == ']'
                    || c == '{' || c == '}' || c == '^' || c == '$' || c == '|' || c == '\\' || c == '+') {
                regex.append('\\').append(c);
            } else {
                regex.append(c);
            }
        }

        return url.matches(".*" + regex.toString() + ".*");
    }

    private static class RouteEntry {
        final String pattern;
        final Consumer<Route> handler;

        RouteEntry(String pattern, Consumer<Route> handler) {
            this.pattern = pattern;
            this.handler = handler;
        }
    }
}
