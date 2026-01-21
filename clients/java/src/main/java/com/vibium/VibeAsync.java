package com.vibium;

import com.vibium.bidi.BiDiEvent;

import java.util.Objects;
import java.util.concurrent.CompletableFuture;
import java.util.concurrent.CompletionException;
import java.util.concurrent.TimeUnit;
import java.util.function.Consumer;

/**
 * Async wrapper around {@link Vibe}.
 *
 * This mirrors the JS/Python async clients, exposing CompletableFuture-returning methods.
 */
public final class VibeAsync {
    private final Vibe inner;

    VibeAsync(Vibe inner) {
        this.inner = Objects.requireNonNull(inner, "inner");
    }

    public CompletableFuture<Void> go(String url) {
        return CompletableFuture.runAsync(() -> inner.go(url), AsyncSupport.executor());
    }

    public CompletableFuture<Void> navigate(String url) {
        return go(url);
    }

    public CompletableFuture<byte[]> screenshot() {
        return CompletableFuture.supplyAsync(inner::screenshot, AsyncSupport.executor());
    }

    public <T> CompletableFuture<T> evaluate(String script, Class<T> clazz) {
        Objects.requireNonNull(clazz, "clazz");
        return CompletableFuture.supplyAsync(() -> inner.evaluate(script, clazz), AsyncSupport.executor());
    }

    public CompletableFuture<String> evaluate(String script) {
        return CompletableFuture.supplyAsync(() -> inner.evaluate(script), AsyncSupport.executor());
    }

    public CompletableFuture<VibeAsync> subscribe(String... events) {
        return CompletableFuture.supplyAsync(() -> {
            inner.subscribe(events);
            return this;
        }, AsyncSupport.executor());
    }

    public CompletableFuture<BiDiEvent> waitForEvent(String method, long timeoutMs) {
        return inner.waitForEventAsync(method)
                .orTimeout(timeoutMs, TimeUnit.MILLISECONDS)
                .handle((event, err) -> {
                    if (err == null) {
                        return event;
                    }
                    Throwable cause = err instanceof CompletionException ? err.getCause() : err;
                    if (cause instanceof java.util.concurrent.TimeoutException) {
                        throw new VibiumTimeoutException("event:" + method, timeoutMs, cause);
                    }
                    throw new CompletionException(cause);
                });
    }

    public Runnable onEvent(Consumer<BiDiEvent> handler) {
        return inner.onEvent(handler);
    }

    public CompletableFuture<ElementAsync> find(String selector) {
        return CompletableFuture.supplyAsync(() -> new ElementAsync(inner.find(selector)), AsyncSupport.executor());
    }

    public CompletableFuture<ElementAsync> find(String selector, int timeoutMs) {
        return CompletableFuture.supplyAsync(() -> new ElementAsync(inner.find(selector, timeoutMs)), AsyncSupport.executor());
    }

    public CompletableFuture<Void> quit() {
        return CompletableFuture.runAsync(inner::quit, AsyncSupport.executor());
    }

    public int port() {
        return inner.port();
    }
}
