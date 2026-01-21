package com.vibium;

import java.util.Objects;
import java.util.concurrent.CompletableFuture;

/**
 * Async wrapper around {@link Element}.
 */
public final class ElementAsync {
    private final Element inner;

    ElementAsync(Element inner) {
        this.inner = Objects.requireNonNull(inner, "inner");
    }

    public ElementInfo info() {
        return inner.info();
    }

    public CompletableFuture<Void> click() {
        return CompletableFuture.runAsync(inner::click, AsyncSupport.executor());
    }

    public CompletableFuture<Void> click(int timeoutMs) {
        return CompletableFuture.runAsync(() -> inner.click(timeoutMs), AsyncSupport.executor());
    }

    public CompletableFuture<Void> type(String text) {
        return CompletableFuture.runAsync(() -> inner.type(text), AsyncSupport.executor());
    }

    public CompletableFuture<Void> type(String text, int timeoutMs) {
        return CompletableFuture.runAsync(() -> inner.type(text, timeoutMs), AsyncSupport.executor());
    }

    public CompletableFuture<String> text() {
        return CompletableFuture.supplyAsync(inner::text, AsyncSupport.executor());
    }

    public CompletableFuture<String> getAttribute(String name) {
        return CompletableFuture.supplyAsync(() -> inner.getAttribute(name), AsyncSupport.executor());
    }

    public CompletableFuture<BoundingBox> boundingBox() {
        return CompletableFuture.supplyAsync(inner::boundingBox, AsyncSupport.executor());
    }
}
