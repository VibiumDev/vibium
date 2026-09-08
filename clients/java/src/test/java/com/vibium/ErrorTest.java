package com.vibium;

import com.vibium.engine.TestServer;
import com.vibium.errors.*;
import com.vibium.types.FindOptions;
import com.vibium.types.StartOptions;
import com.vibium.types.WaitOptions;
import org.junit.jupiter.api.*;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Error handling tests.
 */
class ErrorTest {

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
        page = browser.page();
        page.go(server.baseUrl());
    }

    @Test
    void testElementNotFound() {
        assertThrows(ElementNotFoundException.class, () -> {
            page.find("#nonexistent", new FindOptions().timeout(1000));
        });
    }

    @Test
    void testTimeout() {
        assertThrows(VibiumTimeoutException.class, () -> {
            page.waitForFunction("() => false", new WaitOptions().timeout(1000));
        });
    }

    @Test
    void testVibiumConnectionError() {
        // Starting with a nonexistent binary path should throw a connection exception
        assertThrows(VibiumConnectionException.class, () -> {
            Vibium.start(new StartOptions().executablePath("/nonexistent/vibium"));
        });
    }
}
