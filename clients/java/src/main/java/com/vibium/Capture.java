package com.vibium;

import com.vibium.errors.VibiumException;
import com.vibium.errors.VibiumTimeoutException;
import com.vibium.types.WaitOptions;

import java.util.concurrent.CompletableFuture;
import java.util.concurrent.ExecutionException;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.TimeoutException;
import java.util.function.Consumer;

/**
 * One-shot event capture helpers. Each method registers its listener before
 * starting the action, then removes that listener on every exit path.
 */
public final class Capture {

    private static final long DEFAULT_TIMEOUT_MS = 10_000;
    private static final ExecutorService ACTIONS = Executors.newCachedThreadPool(r -> {
        Thread thread = new Thread(r, "vibium-capture-action");
        thread.setDaemon(true);
        return thread;
    });

    private final Page page;

    Capture(Page page) {
        this.page = page;
    }

    public Request request(String pattern, Runnable action) {
        return request(pattern, action, null);
    }

    public Request request(String pattern, Runnable action, WaitOptions options) {
        return await(
            handler -> page.addRequestListener(handler),
            handler -> page.removeRequestListener(handler),
            request -> Page.matchPattern(pattern, request.url()),
            action,
            options,
            "request matching '" + pattern + "'");
    }

    public Response response(String pattern, Runnable action) {
        return response(pattern, action, null);
    }

    public Response response(String pattern, Runnable action, WaitOptions options) {
        return await(
            handler -> page.addResponseListener(handler),
            handler -> page.removeResponseListener(handler),
            response -> Page.matchPattern(pattern, response.url()),
            action,
            options,
            "response matching '" + pattern + "'");
    }

    public String navigation(Runnable action) {
        return navigation(action, null);
    }

    public String navigation(Runnable action, WaitOptions options) {
        return await(
            handler -> page.addNavigationListener(handler),
            handler -> page.removeNavigationListener(handler),
            value -> true,
            action,
            options,
            "navigation");
    }

    public Download download(Runnable action) {
        return download(action, null);
    }

    public Download download(Runnable action, WaitOptions options) {
        return await(
            handler -> page.addDownloadListener(handler),
            handler -> page.removeDownloadListener(handler),
            value -> true,
            action,
            options,
            "download");
    }

    public Dialog dialog(Runnable action) {
        return dialog(action, null);
    }

    public Dialog dialog(Runnable action, WaitOptions options) {
        return await(
            handler -> page.addDialogListener(handler),
            handler -> page.removeDialogListener(handler),
            value -> true,
            action,
            options,
            "dialog");
    }

    public Object event(String name, Runnable action) {
        return event(name, action, null);
    }

    public Object event(String name, Runnable action, WaitOptions options) {
        switch (name) {
            case "request":
                return awaitAny(page::addRequestListener, page::removeRequestListener,
                    action, options, name);
            case "response":
                return awaitAny(page::addResponseListener, page::removeResponseListener,
                    action, options, name);
            case "navigation":
                return awaitAny(page::addNavigationListener, page::removeNavigationListener,
                    action, options, name);
            case "download":
                return awaitAny(page::addDownloadListener, page::removeDownloadListener,
                    action, options, name);
            case "dialog":
                return awaitAny(page::addDialogListener, page::removeDialogListener,
                    action, options, name);
            case "console":
                return awaitAny(page::addConsoleListener, page::removeConsoleListener,
                    action, options, name);
            case "error":
                return awaitAny(page::addErrorListener, page::removeErrorListener,
                    action, options, name);
            default:
                throw new IllegalArgumentException("Unknown event name: '" + name + "'");
        }
    }

    private <T> T awaitAny(Consumer<Consumer<T>> register,
                           Consumer<Consumer<T>> unregister,
                           Runnable action, WaitOptions options, String name) {
        return await(register, unregister, value -> true, action, options,
            "event '" + name + "'");
    }

    private <T> T await(Consumer<Consumer<T>> register,
                        Consumer<Consumer<T>> unregister,
                        java.util.function.Predicate<T> matches,
                        Runnable action, WaitOptions options, String description) {
        if (action == null) {
            throw new IllegalArgumentException("capture action must not be null");
        }

        CompletableFuture<T> captured = new CompletableFuture<>();
        Consumer<T> handler = value -> {
            if (matches.test(value)) {
                captured.complete(value);
            }
        };

        register.accept(handler);
        CompletableFuture<Void> actionFuture;
        try {
            actionFuture = CompletableFuture.runAsync(action, ACTIONS);
        } catch (RuntimeException error) {
            unregister.accept(handler);
            throw error;
        }

        // A normal action completion does not end capture: the event can arrive
        // after the action returns. A failure before the event does end it.
        actionFuture.whenComplete((ignored, error) -> {
            if (error != null) captured.completeExceptionally(unwrap(error));
        });

        try {
            return captured.get(timeout(options), TimeUnit.MILLISECONDS);
        } catch (TimeoutException error) {
            actionFuture.cancel(true);
            throw new VibiumTimeoutException("Timeout waiting for " + description);
        } catch (ExecutionException error) {
            Throwable cause = unwrap(error.getCause());
            if (cause instanceof RuntimeException) throw (RuntimeException) cause;
            throw new VibiumException("Capture action failed: " + cause.getMessage(), cause);
        } catch (InterruptedException error) {
            actionFuture.cancel(true);
            Thread.currentThread().interrupt();
            throw new VibiumException("Interrupted while waiting for " + description, error);
        } finally {
            unregister.accept(handler);
        }
    }

    private static long timeout(WaitOptions options) {
        return options != null && options.timeout() != null
            ? options.timeout()
            : DEFAULT_TIMEOUT_MS;
    }

    private static Throwable unwrap(Throwable error) {
        Throwable current = error;
        while ((current instanceof java.util.concurrent.CompletionException
                || current instanceof ExecutionException)
                && current.getCause() != null) {
            current = current.getCause();
        }
        return current;
    }
}
