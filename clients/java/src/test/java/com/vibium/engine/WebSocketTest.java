package com.vibium.engine;

import com.vibium.Browser;
import com.vibium.Page;
import com.vibium.Vibium;
import com.vibium.WebSocketInfo;
import com.vibium.types.StartOptions;
import org.junit.jupiter.api.AfterAll;
import org.junit.jupiter.api.AfterEach;
import org.junit.jupiter.api.BeforeAll;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;

import java.util.List;
import java.util.concurrent.CopyOnWriteArrayList;
import java.util.concurrent.CountDownLatch;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.atomic.AtomicInteger;
import java.util.concurrent.atomic.AtomicReference;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertTrue;

@RequiresCapability("core")
class WebSocketTest {

    static Browser browser;
    static TestServer http;
    static WebSocketTestServer sockets;
    Page page;

    @BeforeAll
    static void setup() throws Exception {
        http = new TestServer();
        http.start();
        sockets = new WebSocketTestServer();
        browser = Vibium.start(new StartOptions().headless(true));
    }

    @AfterAll
    static void teardown() throws Exception {
        if (browser != null) browser.stop();
        if (sockets != null) sockets.close();
        if (http != null) http.stop();
    }

    @BeforeEach
    void beforeEach() {
        page = browser.newPage();
        page.go(http.baseUrl());
    }

    @AfterEach
    void afterEach() {
        if (page != null) page.close();
    }

    @Test
    void setupCompletesBeforeImmediateSocketAndDispatchesLifecycle() throws Exception {
        CountDownLatch created = new CountDownLatch(1);
        CountDownLatch sent = new CountDownLatch(1);
        CountDownLatch received = new CountDownLatch(1);
        CountDownLatch closed = new CountDownLatch(1);
        AtomicReference<WebSocketInfo> info = new AtomicReference<>();
        List<String> messages = new CopyOnWriteArrayList<>();

        page.onWebSocket(ws -> {
            info.set(ws);
            ws.onMessage((data, direction) -> {
                messages.add(direction + ":" + data);
                if ("sent".equals(direction)) sent.countDown();
                if ("received".equals(direction)) received.countDown();
            });
            ws.onClose((code, reason) -> closed.countDown());
            created.countDown();
        });

        page.evaluate("(() => { const socket = new WebSocket('" + sockets.url()
            + "'); socket.onopen = () => socket.send('echo-me');"
            + " socket.onmessage = () => socket.close(1000, 'done'); return true; })()");

        assertTrue(created.await(5, TimeUnit.SECONDS));
        assertEquals(sockets.url(), info.get().url());
        assertTrue(sent.await(5, TimeUnit.SECONDS));
        assertTrue(received.await(5, TimeUnit.SECONDS));
        assertTrue(closed.await(5, TimeUnit.SECONDS));
        assertTrue(info.get().isClosed());
        assertTrue(messages.contains("sent:echo-me"));
        assertTrue(messages.contains("received:echo-me"));
    }

    @Test
    void monitoringPersistsAcrossNavigation() throws Exception {
        CountDownLatch created = new CountDownLatch(2);
        page.onWebSocket(ws -> created.countDown());

        page.evaluate("(() => { new WebSocket('" + sockets.url() + "'); return true; })()");
        page.go(http.baseUrl() + "/subpage");
        page.evaluate("(() => { new WebSocket('" + sockets.url() + "'); return true; })()");

        assertTrue(created.await(5, TimeUnit.SECONDS));
    }

    @Test
    void removeAllListenersStopsDelivery() throws Exception {
        AtomicInteger delivered = new AtomicInteger();
        CountDownLatch first = new CountDownLatch(1);
        page.onWebSocket(ws -> {
            delivered.incrementAndGet();
            first.countDown();
        });

        page.evaluate("(() => { new WebSocket('" + sockets.url() + "'); return true; })()");
        assertTrue(first.await(5, TimeUnit.SECONDS));
        page.removeAllListeners("websocket");

        int connections = sockets.connectionCount();
        page.evaluate("(() => { new WebSocket('" + sockets.url() + "'); return true; })()");
        assertTrue(sockets.awaitConnections(connections + 1, 5, TimeUnit.SECONDS));
        page.evaluate("1");
        assertEquals(1, delivered.get());
    }
}
