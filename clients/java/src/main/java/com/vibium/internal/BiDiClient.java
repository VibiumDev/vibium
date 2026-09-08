package com.vibium.internal;

import com.google.gson.Gson;
import com.google.gson.JsonElement;
import com.google.gson.JsonObject;
import com.google.gson.JsonParser;
import com.vibium.errors.*;

import java.io.BufferedReader;
import java.io.BufferedWriter;
import java.io.IOException;
import java.util.List;
import java.util.concurrent.*;
import java.util.concurrent.atomic.AtomicInteger;
import java.util.function.Consumer;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;

/**
 * Low-level ndjson protocol client for communicating with the vibium binary.
 *
 * Thread-safe: multiple threads can call {@link #send} concurrently.
 */
public class BiDiClient {

    private static final long DEFAULT_TIMEOUT_MS = 60_000;
    private static final Gson GSON = new Gson();

    private final BufferedWriter stdin;
    private final BufferedReader stdout;
    private final AtomicInteger nextId = new AtomicInteger(1);
    private final ConcurrentHashMap<Integer, CompletableFuture<JsonObject>> pendingRequests = new ConcurrentHashMap<>();
    private final CopyOnWriteArrayList<Consumer<JsonObject>> eventListeners = new CopyOnWriteArrayList<>();
    private final ExecutorService eventExecutor = Executors.newSingleThreadExecutor(r -> {
        Thread t = new Thread(r, "vibium-event-dispatcher");
        t.setDaemon(true);
        return t;
    });
    private final Thread readerThread;
    private volatile boolean closed = false;

    private BiDiClient(BufferedWriter stdin, BufferedReader stdout) {
        this.stdin = stdin;
        this.stdout = stdout;

        this.readerThread = new Thread(this::receiveLoop, "vibium-bidi-reader");
        this.readerThread.setDaemon(true);
    }

    /**
     * Create a BiDiClient from a VibiumProcess.
     */
    public static BiDiClient fromProcess(VibiumProcess process) {
        BiDiClient client = new BiDiClient(process.getStdin(), process.getStdout());

        // Replay pre-ready lines as events
        for (String line : process.getPreReadyLines()) {
            client.dispatchLine(line);
        }

        // Start the reader thread
        client.readerThread.start();

        return client;
    }

    /**
     * Send a command and wait for the response synchronously.
     */
    public JsonObject send(String method, JsonObject params) {
        return send(method, params, DEFAULT_TIMEOUT_MS);
    }

    /**
     * Send a command and wait for the response with a custom timeout.
     */
    public JsonObject send(String method, JsonObject params, long timeoutMs) {
        StackTraceElement[] callerStack = captureCallerStack();

        int id = nextId.getAndIncrement();
        CompletableFuture<JsonObject> future = writeCommand(id, method, params);

        // Wait for response
        try {
            JsonObject response = future.get(timeoutMs, TimeUnit.MILLISECONDS);
            return response;
        } catch (TimeoutException e) {
            pendingRequests.remove(id);
            throw new VibiumTimeoutException("Timeout after " + timeoutMs + "ms waiting for response to " + method);
        } catch (ExecutionException e) {
            pendingRequests.remove(id);
            Throwable cause = e.getCause();
            if (cause instanceof VibiumException) {
                throw attachCallerStack((VibiumException) cause, callerStack);
            }
            throw new VibiumException("Error executing " + method + ": " + cause.getMessage(), cause);
        } catch (InterruptedException e) {
            pendingRequests.remove(id);
            Thread.currentThread().interrupt();
            throw new VibiumException("Interrupted while waiting for response to " + method);
        }
    }

    /**
     * Write a command now and return the pending response as a future.
     *
     * The write happens on the caller's thread, so any command sent after
     * this returns is ordered behind it on the wire. Captures depend on that:
     * the engine must register the capture before the action that triggers
     * its event can be processed (#446). No client-side timeout is armed; the
     * engine answers every capture, with an error once its own timeout
     * passes.
     */
    public CompletableFuture<JsonObject> sendAsync(String method, JsonObject params) {
        return writeCommand(nextId.getAndIncrement(), method, params);
    }

    private CompletableFuture<JsonObject> writeCommand(int id, String method, JsonObject params) {
        if (closed) {
            throw new VibiumConnectionException("BiDi client is closed");
        }
        CompletableFuture<JsonObject> future = new CompletableFuture<>();
        pendingRequests.put(id, future);

        JsonObject command = new JsonObject();
        command.addProperty("id", id);
        command.addProperty("method", method);
        if (params != null) {
            command.add("params", params);
        } else {
            command.add("params", new JsonObject());
        }

        try {
            synchronized (stdin) {
                stdin.write(GSON.toJson(command));
                stdin.newLine();
                stdin.flush();
            }
        } catch (IOException e) {
            pendingRequests.remove(id);
            throw new VibiumConnectionException("Failed to send command: " + e.getMessage(), e);
        }
        return future;
    }

    static StackTraceElement[] captureCallerStack() {
        StackTraceElement[] stack = new Throwable().getStackTrace();
        int firstCaller = 0;
        while (firstCaller < stack.length
                && BiDiClient.class.getName().equals(stack[firstCaller].getClassName())) {
            firstCaller++;
        }
        return java.util.Arrays.copyOfRange(stack, firstCaller, stack.length);
    }

    static <T extends VibiumException> T attachCallerStack(T error, StackTraceElement[] callerStack) {
        error.setStackTrace(callerStack);
        return error;
    }

    /**
     * Register an event listener for messages with no id (events).
     */
    public void onEvent(Consumer<JsonObject> handler) {
        eventListeners.add(handler);
    }

    /**
     * Remove an event listener.
     */
    public void offEvent(Consumer<JsonObject> handler) {
        eventListeners.remove(handler);
    }

    /**
     * Close the client.
     */
    public void close() {
        closed = true;
        readerThread.interrupt();
        eventExecutor.shutdownNow();

        // Fail all pending requests
        VibiumConnectionException closedEx = new VibiumConnectionException("Connection closed");
        for (CompletableFuture<JsonObject> future : pendingRequests.values()) {
            future.completeExceptionally(closedEx);
        }
        pendingRequests.clear();
    }

    private void receiveLoop() {
        try {
            String line;
            while (!closed && (line = stdout.readLine()) != null) {
                dispatchLine(line);
            }
        } catch (IOException e) {
            if (!closed) {
                // Connection lost
                VibiumConnectionException ex = new VibiumConnectionException("Connection lost: " + e.getMessage(), e);
                for (CompletableFuture<JsonObject> future : pendingRequests.values()) {
                    future.completeExceptionally(ex);
                }
                pendingRequests.clear();
            }
        }
    }

    private void dispatchLine(String line) {
        if (line == null || line.trim().isEmpty()) return;

        try {
            JsonObject msg = JsonParser.parseString(line).getAsJsonObject();

            if (msg.has("id") && !msg.get("id").isJsonNull()) {
                // Response to a command
                int id = msg.get("id").getAsInt();
                CompletableFuture<JsonObject> future = pendingRequests.remove(id);
                if (future != null) {
                    String type = msg.has("type") ? msg.get("type").getAsString() : "";

                    if ("error".equals(type)) {
                        String error = msg.has("error") ? msg.get("error").getAsString() : "unknown";
                        String message = msg.has("message") ? msg.get("message").getAsString() : "Unknown error";
                        future.completeExceptionally(mapError(error, message));
                    } else {
                        // Success - return the result
                        JsonObject result = msg.has("result") ? msg.getAsJsonObject("result") : new JsonObject();
                        future.complete(result);
                    }
                }
            } else if (msg.has("method")) {
                // Event — dispatch to listeners on a separate thread to avoid deadlock
                final JsonObject event = msg;
                eventExecutor.submit(() -> {
                    for (Consumer<JsonObject> listener : eventListeners) {
                        try {
                            listener.accept(event);
                        } catch (Exception ignored) {
                            // Don't let a bad listener crash the event thread
                        }
                    }
                });
            }
        } catch (Exception ignored) {
            // Malformed JSON line, skip
        }
    }

    /**
     * Map wire error strings to Java exceptions.
     */
    private static VibiumException mapError(String error, String message) {
        if ("not_found".equals(error) || "no_such_element".equals(error)) {
            return new ElementNotFoundException(message);
        }

        if ("timeout".equals(error)) {
            String lower = message.toLowerCase();
            // "element not found" wrapped in a timeout is still an element error
            if (lower.contains("element not found") || lower.contains("no elements found")) {
                return new ElementNotFoundException(message);
            }
            return new VibiumTimeoutException(message);
        }

        String lower = message.toLowerCase();
        if (lower.contains("not found") || lower.contains("no elements")) {
            return new ElementNotFoundException(message);
        }

        return new VibiumException(message);
    }
}
