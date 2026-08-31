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
import java.util.concurrent.atomic.AtomicBoolean;
import java.util.function.Consumer;
import java.util.function.Supplier;

/**
 * One-shot event capture helpers. The binary owns the matching and the wait
 * (vibium:page.captureRequest/captureResponse/captureEvent); each method
 * starts that command, runs the action while it is in flight, and awaits the
 * result. Downloads still capture through a local listener, because the
 * captured Download must be the same object the downloadEnd event completes.
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
        long timeoutMs = timeout(options);
        return awaitWire(() -> page.sendCapture("vibium:page.captureRequest", pattern, timeoutMs),
            timeoutMs, action,
            event -> page.requestFromEvent(event),
            "request matching '" + pattern + "'");
    }

    public Response response(String pattern, Runnable action) {
        return response(pattern, action, null);
    }

    public Response response(String pattern, Runnable action, WaitOptions options) {
        long timeoutMs = timeout(options);
        return awaitWire(() -> page.sendCapture("vibium:page.captureResponse", pattern, timeoutMs),
            timeoutMs, action,
            event -> page.responseFromEvent(event),
            "response matching '" + pattern + "'");
    }

    public String navigation(Runnable action) {
        return navigation(action, null);
    }

    public String navigation(Runnable action, WaitOptions options) {
        long timeoutMs = timeout(options);
        return awaitWire(() -> page.sendCaptureEvent("navigation", timeoutMs),
            timeoutMs, action,
            event -> event.has("url") ? event.get("url").getAsString() : "",
            "navigation");
    }

    public Download download(Runnable action) {
        return download(action, null);
    }

    public Download download(Runnable action, WaitOptions options) {
        return await(
            handler -> page.addDownloadListener(handler),
            handler -> page.removeDownloadListener(handler),
            action,
            options,
            "download");
    }

    public Dialog dialog(Runnable action) {
        return dialog(action, null);
    }

    public Dialog dialog(Runnable action, WaitOptions options) {
        long timeoutMs = timeout(options);
        return awaitWire(() -> page.sendCaptureEvent("dialog", timeoutMs),
            timeoutMs, action,
            event -> page.dialogFromEvent(event),
            "dialog");
    }

    public Object event(String name, Runnable action) {
        return event(name, action, null);
    }

    public Object event(String name, Runnable action, WaitOptions options) {
        switch (name) {
            case "request":
                return request("**", action, options);
            case "response":
                return response("**", action, options);
            case "navigation":
                return navigation(action, options);
            case "download":
                return download(action, options);
            case "dialog":
                return dialog(action, options);
            case "console": {
                long timeoutMs = timeout(options);
                return awaitWire(() -> page.sendCaptureEvent("console", timeoutMs),
                    timeoutMs, action,
                    event -> page.consoleFromEvent(event),
                    "event 'console'");
            }
            case "error": {
                long timeoutMs = timeout(options);
                return awaitWire(() -> page.sendCaptureEvent("error", timeoutMs),
                    timeoutMs, action,
                    event -> event.has("text") ? event.get("text").getAsString() : "",
                    "event 'error'");
            }
            default:
                throw new IllegalArgumentException("Unknown event name: '" + name + "'");
        }
    }

    private <T> T awaitWire(Supplier<CompletableFuture<com.google.gson.JsonObject>> start, long timeoutMs,
                            Runnable action,
                            java.util.function.Function<com.google.gson.JsonObject, T> build, String what) {
        if (action == null) {
            throw new IllegalArgumentException("capture action must not be null");
        }
        // The capture command is written on this thread before the action can
        // send anything, so the engine registers the capture first, in
        // message order.
        CompletableFuture<com.google.gson.JsonObject> pending = start.get();
        // The action runs on the executor: a trigger that blocks until the
        // captured thing is handled — an eval stuck inside the alert it
        // opened (#146) — must not block the capture's return.
        CompletableFuture<Void> actionFuture = CompletableFuture.runAsync(action::run, ACTIONS);
        actionFuture.whenComplete((ignored, error) -> {
            if (error != null) pending.completeExceptionally(unwrap(error));
        });
        try {
            // The binary enforces timeoutMs; the slack only covers transport.
            com.google.gson.JsonObject result = pending.get(timeoutMs + 5_000, TimeUnit.MILLISECONDS);
            return build.apply(result.getAsJsonObject("event"));
        } catch (TimeoutException e) {
            pending.cancel(true);
            throw new VibiumTimeoutException("Timeout waiting for " + what);
        } catch (ExecutionException e) {
            Throwable cause = unwrap(e.getCause());
            if (cause instanceof RuntimeException) {
                throw (RuntimeException) cause;
            }
            throw new VibiumException("Capture action failed: " + cause.getMessage(), cause);
        } catch (InterruptedException e) {
            Thread.currentThread().interrupt();
            throw new VibiumTimeoutException("Interrupted waiting for " + what);
        }
    }

    private <T> T await(Consumer<Consumer<T>> register,
                        Consumer<Consumer<T>> unregister,
                        Runnable action, WaitOptions options, String description) {
        if (action == null) {
            throw new IllegalArgumentException("capture action must not be null");
        }

        CompletableFuture<T> captured = new CompletableFuture<>();
        AtomicBoolean actionStarted = new AtomicBoolean();
        Consumer<T> handler = value -> {
            if (actionStarted.get()) {
                captured.complete(value);
            }
        };

        register.accept(handler);
        CompletableFuture<Void> actionFuture;
        try {
            actionFuture = CompletableFuture.runAsync(() -> {
                actionStarted.set(true);
                action.run();
            }, ACTIONS);
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
