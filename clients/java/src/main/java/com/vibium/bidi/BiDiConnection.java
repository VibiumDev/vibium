package com.vibium.bidi;

import com.vibium.VibiumConnectionException;
import org.java_websocket.client.WebSocketClient;
import org.java_websocket.handshake.ServerHandshake;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.net.URI;
import java.util.Objects;
import java.util.concurrent.TimeUnit;
import java.util.function.Consumer;

final class BiDiConnection extends WebSocketClient {
    private static final Logger log = LoggerFactory.getLogger(BiDiConnection.class);

    private final long timeoutMs;
    private volatile Consumer<String> messageHandler;
    private volatile Exception error;

    private BiDiConnection(URI serverUri, long timeoutMs) {
        super(serverUri);
        this.timeoutMs = timeoutMs;
    }

    static BiDiConnection connect(String wsUrl, long timeoutMs) {
        Objects.requireNonNull(wsUrl, "wsUrl");

        try {
            BiDiConnection conn = new BiDiConnection(URI.create(wsUrl), timeoutMs);
            boolean opened = conn.connectBlocking(timeoutMs, TimeUnit.MILLISECONDS);
            if (!opened) {
                throw new VibiumConnectionException(wsUrl, "Timeout connecting to " + wsUrl);
            }
            if (conn.error != null) {
                throw new VibiumConnectionException(wsUrl, "Failed to connect to " + wsUrl, conn.error);
            }
            return conn;
        } catch (InterruptedException e) {
            Thread.currentThread().interrupt();
            throw new VibiumConnectionException(wsUrl, "Interrupted connecting to " + wsUrl, e);
        }
    }

    long getTimeoutMs() {
        return timeoutMs;
    }

    void onMessage(Consumer<String> handler) {
        this.messageHandler = handler;
    }

    @Override
    public void onOpen(ServerHandshake handshake) {
        log.debug("WebSocket opened: {}", getURI());
    }

    @Override
    public void onMessage(String message) {
        Consumer<String> handler = messageHandler;
        if (handler != null) {
            handler.accept(message);
        }
    }

    @Override
    public void onClose(int code, String reason, boolean remote) {
        log.debug("WebSocket closed: code={}, reason={}, remote={}", code, reason, remote);
    }

    @Override
    public void onError(Exception ex) {
        this.error = ex;
        log.debug("WebSocket error", ex);
    }

    void sendText(String message) {
        super.send(message);
    }

    void closeConnection() {
        try {
            closeBlocking();
        } catch (InterruptedException e) {
            Thread.currentThread().interrupt();
        }
    }
}
