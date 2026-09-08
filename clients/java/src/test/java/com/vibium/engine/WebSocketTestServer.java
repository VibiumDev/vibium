package com.vibium.engine;

import java.io.BufferedInputStream;
import java.io.BufferedOutputStream;
import java.io.ByteArrayOutputStream;
import java.io.IOException;
import java.net.InetAddress;
import java.net.ServerSocket;
import java.net.Socket;
import java.nio.charset.StandardCharsets;
import java.security.MessageDigest;
import java.util.Base64;
import java.util.Locale;
import java.util.concurrent.CountDownLatch;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.atomic.AtomicInteger;

/** Minimal local RFC 6455 echo server for the Java client tests. */
final class WebSocketTestServer implements AutoCloseable {

    private static final String MAGIC = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11";

    private final ServerSocket server;
    private final ExecutorService workers = Executors.newCachedThreadPool(r -> {
        Thread thread = new Thread(r, "vibium-test-websocket");
        thread.setDaemon(true);
        return thread;
    });
    private final AtomicInteger connections = new AtomicInteger();
    private volatile CountDownLatch connectionChanged = new CountDownLatch(1);
    private volatile boolean closed;

    WebSocketTestServer() throws IOException {
        server = new ServerSocket(0, 50, InetAddress.getLoopbackAddress());
        workers.submit(this::acceptLoop);
    }

    String url() {
        return "ws://127.0.0.1:" + server.getLocalPort();
    }

    int connectionCount() {
        return connections.get();
    }

    boolean awaitConnections(int count, long timeout, TimeUnit unit) throws InterruptedException {
        long deadline = System.nanoTime() + unit.toNanos(timeout);
        while (connections.get() < count) {
            long remaining = deadline - System.nanoTime();
            if (remaining <= 0 || !connectionChanged.await(remaining, TimeUnit.NANOSECONDS)) {
                return connections.get() >= count;
            }
        }
        return true;
    }

    private void acceptLoop() {
        while (!closed) {
            try {
                Socket socket = server.accept();
                connections.incrementAndGet();
                CountDownLatch changed = connectionChanged;
                connectionChanged = new CountDownLatch(1);
                changed.countDown();
                workers.submit(() -> handle(socket));
            } catch (IOException error) {
                if (!closed) throw new RuntimeException(error);
            }
        }
    }

    private void handle(Socket socket) {
        try (Socket ignored = socket;
             BufferedInputStream input = new BufferedInputStream(socket.getInputStream());
             BufferedOutputStream output = new BufferedOutputStream(socket.getOutputStream())) {
            String headers = readHeaders(input);
            String key = null;
            for (String line : headers.split("\\r\\n")) {
                if (line.toLowerCase(Locale.ROOT).startsWith("sec-websocket-key:")) {
                    key = line.substring(line.indexOf(':') + 1).trim();
                }
            }
            if (key == null) return;

            String accept = Base64.getEncoder().encodeToString(
                MessageDigest.getInstance("SHA-1")
                    .digest((key + MAGIC).getBytes(StandardCharsets.US_ASCII)));
            output.write(("HTTP/1.1 101 Switching Protocols\r\n"
                + "Upgrade: websocket\r\n"
                + "Connection: Upgrade\r\n"
                + "Sec-WebSocket-Accept: " + accept + "\r\n\r\n")
                .getBytes(StandardCharsets.US_ASCII));
            output.flush();

            while (!socket.isClosed()) {
                Frame frame = readFrame(input);
                if (frame == null) return;
                if (frame.opcode == 1) {
                    writeFrame(output, 1, frame.payload);
                } else if (frame.opcode == 8) {
                    writeFrame(output, 8, frame.payload);
                    return;
                } else if (frame.opcode == 9) {
                    writeFrame(output, 10, frame.payload);
                }
            }
        } catch (Exception ignored) {
            // A browser closing a test page can drop the socket mid-frame.
        }
    }

    private static String readHeaders(BufferedInputStream input) throws IOException {
        ByteArrayOutputStream bytes = new ByteArrayOutputStream();
        int state = 0;
        while (bytes.size() < 16 * 1024) {
            int value = input.read();
            if (value < 0) break;
            bytes.write(value);
            state = value == "\r\n\r\n".charAt(state) ? state + 1 : (value == '\r' ? 1 : 0);
            if (state == 4) break;
        }
        return bytes.toString(StandardCharsets.US_ASCII);
    }

    private static Frame readFrame(BufferedInputStream input) throws IOException {
        int first = input.read();
        int second = input.read();
        if (first < 0 || second < 0) return null;

        long length = second & 0x7f;
        if (length == 126) {
            length = (readRequired(input) << 8) | readRequired(input);
        } else if (length == 127) {
            length = 0;
            for (int i = 0; i < 8; i++) length = (length << 8) | readRequired(input);
        }
        if (length > 1024 * 1024) throw new IOException("test WebSocket frame too large");

        byte[] mask = null;
        if ((second & 0x80) != 0) {
            mask = input.readNBytes(4);
            if (mask.length != 4) throw new IOException("truncated WebSocket mask");
        }
        byte[] payload = input.readNBytes((int) length);
        if (payload.length != length) throw new IOException("truncated WebSocket payload");
        if (mask != null) {
            for (int i = 0; i < payload.length; i++) payload[i] ^= mask[i % 4];
        }
        return new Frame(first & 0x0f, payload);
    }

    private static int readRequired(BufferedInputStream input) throws IOException {
        int value = input.read();
        if (value < 0) throw new IOException("truncated WebSocket frame");
        return value;
    }

    private static void writeFrame(BufferedOutputStream output, int opcode, byte[] payload)
            throws IOException {
        output.write(0x80 | opcode);
        if (payload.length < 126) {
            output.write(payload.length);
        } else {
            output.write(126);
            output.write((payload.length >>> 8) & 0xff);
            output.write(payload.length & 0xff);
        }
        output.write(payload);
        output.flush();
    }

    @Override
    public void close() throws IOException {
        closed = true;
        server.close();
        workers.shutdownNow();
    }

    private static final class Frame {
        final int opcode;
        final byte[] payload;

        Frame(int opcode, byte[] payload) {
            this.opcode = opcode;
            this.payload = payload;
        }
    }
}
