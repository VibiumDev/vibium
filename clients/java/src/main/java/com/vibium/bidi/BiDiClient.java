package com.vibium.bidi;

import com.google.gson.Gson;
import com.google.gson.JsonArray;
import com.google.gson.JsonElement;
import com.google.gson.JsonObject;
import com.vibium.VibiumException;
import com.vibium.VibiumRemoteErrorException;
import com.vibium.VibiumTimeoutException;

import java.util.Map;
import java.util.Objects;
import java.util.concurrent.CompletableFuture;
import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.TimeoutException;
import java.util.concurrent.atomic.AtomicLong;
import java.util.concurrent.CopyOnWriteArrayList;
import java.util.function.Consumer;

public final class BiDiClient {
    public static final Gson GSON = new Gson();

    private final BiDiConnection connection;
    private final AtomicLong nextId = new AtomicLong(1);
    private final Map<Long, PendingRequest> pending = new ConcurrentHashMap<>();
    private final CopyOnWriteArrayList<Consumer<BiDiEvent>> eventHandlers = new CopyOnWriteArrayList<>();

    private BiDiClient(BiDiConnection connection) {
        this.connection = connection;
        this.connection.onMessage(this::handleMessage);
    }

    public static BiDiClient connect(String wsUrl, long timeoutMs) {
        BiDiConnection conn = BiDiConnection.connect(wsUrl, timeoutMs);
        return new BiDiClient(conn);
    }

    public void close() {
        for (PendingRequest pendingRequest : pending.values()) {
            pendingRequest.future.completeExceptionally(new VibiumException("Connection closed"));
        }
        pending.clear();
        connection.closeConnection();
    }

    /**
     * Subscribe to BiDi events (messages without an id).
     *
     * Returns an unsubscribe handle.
     */
    public Runnable onEvent(Consumer<BiDiEvent> handler) {
        Objects.requireNonNull(handler, "handler");
        eventHandlers.add(handler);
        return () -> eventHandlers.remove(handler);
    }

    public JsonObject sendObj(String method, JsonObject params) {
        return sendObj(method, params, connection.getTimeoutMs());
    }

    public JsonObject sendObj(String method, JsonObject params, long timeoutMs) {
        Objects.requireNonNull(method, "method");
        long id = nextId.getAndIncrement();

        JsonObject msg = new JsonObject();
        msg.addProperty("id", id);
        msg.addProperty("method", method);
        msg.add("params", params == null ? new JsonObject() : params);

        CompletableFuture<JsonObject> future = new CompletableFuture<>();
        pending.put(id, new PendingRequest(method, timeoutMs, future));

        try {
            connection.sendText(GSON.toJson(msg));
        } catch (Exception e) {
            pending.remove(id);
            throw new VibiumException("Failed to send command: " + method, e);
        }

        try {
            JsonObject response = future.get(timeoutMs, TimeUnit.MILLISECONDS);
            JsonElement result = response.get("result");
            if (result == null || result.isJsonNull()) {
                return new JsonObject();
            }
            if (!result.isJsonObject()) {
                JsonObject wrapped = new JsonObject();
                wrapped.add("value", result);
                return wrapped;
            }
            return result.getAsJsonObject();
        } catch (TimeoutException e) {
            pending.remove(id);
            throw new VibiumTimeoutException(method, timeoutMs, e);
        } catch (Exception e) {
            pending.remove(id);
            if (e.getCause() instanceof VibiumException) {
                throw (VibiumException) e.getCause();
            }
            throw new VibiumException("Request failed: " + method, e);
        }
    }

    public void send(String method, JsonObject params) {
        sendObj(method, params, connection.getTimeoutMs());
    }

    public void send(String method, JsonObject params, long timeoutMs) {
        sendObj(method, params, timeoutMs);
    }

    private void handleMessage(String message) {
        JsonObject obj;
        try {
            obj = GSON.fromJson(message, JsonObject.class);
        } catch (Exception ignored) {
            return;
        }

        if (!obj.has("id")) {
            dispatchEvent(obj);
            return;
        }

        long id = obj.get("id").getAsLong();
        PendingRequest pendingRequest = pending.remove(id);
        if (pendingRequest == null) {
            return;
        }

        String type = obj.has("type") ? obj.get("type").getAsString() : "success";
        if ("error".equals(type)) {
            String error = obj.has("error") ? obj.get("error").getAsString() : "error";
            String msg = obj.has("message") ? obj.get("message").getAsString() : "unknown error";
            if ("timeout".equalsIgnoreCase(error)) {
                pendingRequest.future.completeExceptionally(
                        new VibiumTimeoutException(pendingRequest.method, pendingRequest.timeoutMs, msg)
                );
            } else {
                pendingRequest.future.completeExceptionally(new VibiumRemoteErrorException(error, msg));
            }
            return;
        }

        pendingRequest.future.complete(obj);
    }

    private void dispatchEvent(JsonObject obj) {
        if (!obj.has("method")) {
            return;
        }
        String method = obj.get("method").getAsString();
        JsonObject params = obj.has("params") && obj.get("params").isJsonObject()
                ? obj.getAsJsonObject("params")
                : new JsonObject();

        if (eventHandlers.isEmpty()) {
            return;
        }

        BiDiEvent event = new BiDiEvent(method, params);
        for (Consumer<BiDiEvent> h : eventHandlers) {
            try {
                h.accept(event);
            } catch (Exception ignored) {
            }
        }
    }

    public static JsonArray argsString(String... values) {
        JsonArray args = new JsonArray();
        for (String v : values) {
            JsonObject arg = new JsonObject();
            arg.addProperty("type", "string");
            arg.addProperty("value", v);
            args.add(arg);
        }
        return args;
    }

    private static final class PendingRequest {
        private final String method;
        private final long timeoutMs;
        private final CompletableFuture<JsonObject> future;

        private PendingRequest(String method, long timeoutMs, CompletableFuture<JsonObject> future) {
            this.method = method;
            this.timeoutMs = timeoutMs;
            this.future = future;
        }
    }
}
