package com.vibium;

import com.google.gson.JsonArray;
import com.google.gson.JsonObject;
import com.vibium.bidi.BiDiClient;
import com.vibium.bidi.BiDiEvent;
import com.vibium.process.ClickerProcess;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.util.Base64;
import java.util.Objects;
import java.util.concurrent.CompletableFuture;
import java.util.concurrent.TimeUnit;
import java.util.function.Consumer;

public final class Vibe implements AutoCloseable {
    private static final Logger log = LoggerFactory.getLogger(Vibe.class);

    private final BiDiClient client;
    private final ClickerProcess process;
    private String contextId;
    private boolean closed = false;

    Vibe(BiDiClient client, ClickerProcess process) {
        this.client = Objects.requireNonNull(client, "client");
        this.process = process;
    }

    public void go(String url) {
        go(url, null);
    }

    public void navigate(String url) {
        go(url);
    }

    public void go(String url, NavigateOptions options) {
        Objects.requireNonNull(url, "url");
        String ctx = getContextId();

        JsonObject params = new JsonObject();
        params.addProperty("context", ctx);
        params.addProperty("url", url);
        params.addProperty("wait", "complete");

        long requestTimeoutMs = options != null && options.timeoutMs != null ? options.timeoutMs : 0;
        if (requestTimeoutMs > 0) {
            client.send("browsingContext.navigate", params, requestTimeoutMs);
        } else {
            client.send("browsingContext.navigate", params);
        }
    }

    public byte[] screenshot() {
        String ctx = getContextId();
        JsonObject params = new JsonObject();
        params.addProperty("context", ctx);

        JsonObject result = client.sendObj("browsingContext.captureScreenshot", params);
        String data = result.get("data").getAsString();
        return Base64.getDecoder().decode(data);
    }

    public <T> T evaluate(String script, Class<T> clazz) {
        Objects.requireNonNull(script, "script");
        Objects.requireNonNull(clazz, "clazz");
        String ctx = getContextId();

        JsonObject params = new JsonObject();
        params.addProperty("functionDeclaration", "() => { " + script + " }");
        JsonObject target = new JsonObject();
        target.addProperty("context", ctx);
        params.add("target", target);
        params.add("arguments", new JsonArray());
        params.addProperty("awaitPromise", true);
        params.addProperty("resultOwnership", "root");

        JsonObject result = client.sendObj("script.callFunction", params);
        if (result == null) {
            throw new VibiumException("script.callFunction returned null result");
        }
        if (result.has("exceptionDetails")) {
            throw new VibiumRemoteErrorException("script", result.get("exceptionDetails").toString());
        }
        if (!result.has("result") || !result.get("result").isJsonObject()) {
            throw new VibiumRemoteErrorException("script", "Unexpected script.callFunction result: " + result);
        }

        JsonObject inner = result.getAsJsonObject("result");
        if (!inner.has("value") || inner.get("value").isJsonNull()) {
            return null;
        }
        return BiDiClient.GSON.fromJson(inner.get("value"), clazz);
    }

    public String evaluate(String script) {
        Object val = evaluate(script, Object.class);
        return val == null ? null : val.toString();
    }

    public String title() {
        return evaluate("return document.title");
    }

    public String url() {
        return evaluate("return window.location.href");
    }

    public Element find(String selector) {
        return find(selector, null);
    }

    public Element find(String selector, int timeoutMs) {
        return find(selector, FindOptions.timeoutMs(timeoutMs));
    }

    public Element find(String selector, FindOptions options) {
        Objects.requireNonNull(selector, "selector");
        String ctx = getContextId();

        JsonObject params = new JsonObject();
        params.addProperty("context", ctx);
        params.addProperty("selector", selector);
        long requestTimeoutMs = 0;
        if (options != null && options.timeoutMs != null) {
            params.addProperty("timeout", options.timeoutMs);
            requestTimeoutMs = options.timeoutMs;
        }

        JsonObject result = requestTimeoutMs > 0
                ? client.sendObj("vibium:find", params, requestTimeoutMs)
                : client.sendObj("vibium:find", params);

        String tag = result.get("tag").getAsString();
        String text = result.get("text").getAsString();
        JsonObject boxObj = result.getAsJsonObject("box");
        BoundingBox box = new BoundingBox(
                boxObj.get("x").getAsDouble(),
                boxObj.get("y").getAsDouble(),
                boxObj.get("width").getAsDouble(),
                boxObj.get("height").getAsDouble()
        );

        return new Element(client, ctx, selector, new ElementInfo(tag, text, box));
    }

    /**
     * Subscribe to BiDi events forwarded by Clicker.
     *
     * Returns an unsubscribe handle.
     */
    public Runnable onEvent(Consumer<BiDiEvent> handler) {
        return client.onEvent(handler);
    }

    /**
     * Subscribe to BiDi event streams via {@code session.subscribe}.
     *
     * This is optional: Clicker may only forward events after subscription depending on the browser/session.
     */
    public void subscribe(String... events) {
        if (events == null || events.length == 0) {
            return;
        }

        String ctx = getContextId();
        JsonObject params = new JsonObject();

        JsonArray ev = new JsonArray();
        for (String e : events) {
            if (e == null || e.isBlank()) continue;
            ev.add(e);
        }
        params.add("events", ev);

        JsonArray contexts = new JsonArray();
        contexts.add(ctx);
        params.add("contexts", contexts);

        client.send("session.subscribe", params);
    }

    /**
     * Wait for the next event with a given method name.
     */
    public BiDiEvent waitForEvent(String method, long timeoutMs) {
        try {
            return waitForEventAsync(method).get(timeoutMs, TimeUnit.MILLISECONDS);
        } catch (java.util.concurrent.TimeoutException e) {
            throw new VibiumTimeoutException("event:" + method, timeoutMs, e);
        } catch (Exception e) {
            throw new VibiumException("Failed while waiting for event: " + method, e);
        }
    }

    /**
     * Get a future that completes with the next event matching {@code method}.
     *
     * Caller should use {@link CompletableFuture#orTimeout(long, TimeUnit)} if a timeout is desired.
     */
    public CompletableFuture<BiDiEvent> waitForEventAsync(String method) {
        Objects.requireNonNull(method, "method");

        CompletableFuture<BiDiEvent> future = new CompletableFuture<>();
        Runnable unsub = onEvent(event -> {
            if (method.equals(event.method) && !future.isDone()) {
                future.complete(event);
            }
        });

        future.whenComplete((ignored, ignoredErr) -> {
            try {
                unsub.run();
            } catch (Exception ignoredEx) {
            }
        });

        return future;
    }

    public void quit() {
        close();
    }

    public int port() {
        return process != null ? process.getPort() : -1;
    }

    @Override
    public void close() {
        if (closed) {
            return;
        }
        closed = true;

        try {
            client.close();
        } catch (Exception e) {
            log.debug("Error closing BiDi client", e);
        }
        if (process != null) {
            try {
                // Wait for Clicker to close the browser session after the WS disconnect,
                // then stop the Clicker process (which otherwise runs until killed).
                process.waitForSessionClosed(5_000);
                process.stop();
            } catch (Exception e) {
                log.debug("Error stopping Clicker process", e);
            }
        }
    }

    private String getContextId() {
        if (contextId != null) {
            return contextId;
        }

        JsonObject result = client.sendObj("browsingContext.getTree", new JsonObject());
        if (!result.has("contexts") || result.get("contexts").isJsonNull()) {
            throw new VibiumException("No browsing context available");
        }
        if (result.getAsJsonArray("contexts").size() == 0) {
            throw new VibiumException("No browsing context available");
        }

        contextId = result.getAsJsonArray("contexts").get(0).getAsJsonObject().get("context").getAsString();
        return contextId;
    }

    public static final class FindOptions {
        private final Integer timeoutMs;

        private FindOptions(Integer timeoutMs) {
            this.timeoutMs = timeoutMs;
        }

        public static FindOptions timeoutMs(int timeoutMs) {
            return new FindOptions(timeoutMs);
        }
    }

    public static final class NavigateOptions {
        private final Long timeoutMs;

        private NavigateOptions(Long timeoutMs) {
            this.timeoutMs = timeoutMs;
        }

        public static NavigateOptions timeoutMs(long timeoutMs) {
            return new NavigateOptions(timeoutMs);
        }
    }
}
