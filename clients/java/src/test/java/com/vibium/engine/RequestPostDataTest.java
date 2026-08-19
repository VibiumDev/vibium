package com.vibium.engine;

import com.vibium.Browser;
import com.vibium.Page;
import com.vibium.Vibium;
import com.vibium.types.StartOptions;
import org.junit.jupiter.api.AfterAll;
import org.junit.jupiter.api.AfterEach;
import org.junit.jupiter.api.BeforeAll;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;

import java.util.concurrent.CountDownLatch;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.atomic.AtomicReference;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertNull;
import static org.junit.jupiter.api.Assertions.assertTrue;

@RequiresCapability("core")
class RequestPostDataTest {

    static Browser browser;
    static TestServer server;
    Page page;

    @BeforeAll
    static void setup() throws Exception {
        server = new TestServer();
        server.start();
        browser = Vibium.start(new StartOptions().headless(true));
    }

    @AfterAll
    static void teardown() {
        if (browser != null) browser.stop();
        if (server != null) server.stop();
    }

    @BeforeEach
    void beforeEach() {
        page = browser.newPage();
        page.go(server.baseUrl());
    }

    @AfterEach
    void afterEach() {
        if (page != null) page.close();
    }

    @Test
    void readsPostDataFromRequestListener() throws Exception {
        AtomicReference<String> body = new AtomicReference<>();
        CountDownLatch received = new CountDownLatch(1);
        page.onRequest(request -> {
            if (request.url().endsWith("/api/echo")) {
                body.set(request.postData());
                received.countDown();
            }
        });

        post("listener-body");

        assertTrue(received.await(5, TimeUnit.SECONDS));
        assertEquals("listener-body", body.get());
    }

    @Test
    void readsPostDataInsideRouteBeforeContinuing() throws Exception {
        AtomicReference<String> body = new AtomicReference<>();
        CountDownLatch received = new CountDownLatch(1);
        page.route("**/api/echo", route -> {
            body.set(route.request().postData());
            route.doContinue();
            received.countDown();
        });

        post("route-body");

        assertTrue(received.await(5, TimeUnit.SECONDS));
        assertEquals("route-body", body.get());
    }

    @Test
    void returnsNullForGetWithoutPostData() throws Exception {
        AtomicReference<String> body = new AtomicReference<>("not-called");
        CountDownLatch received = new CountDownLatch(1);
        page.onRequest(request -> {
            if (request.url().endsWith("/api/data")) {
                body.set(request.postData());
                received.countDown();
            }
        });

        page.evaluate("fetch('/api/data')");

        assertTrue(received.await(5, TimeUnit.SECONDS));
        assertNull(body.get());
    }

    private void post(String body) {
        page.evaluate("fetch('/api/echo', { method: 'POST', body: '" + body + "' })");
    }
}
