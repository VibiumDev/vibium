package com.vibium.engine;

import com.vibium.Browser;
import com.vibium.Page;
import com.vibium.Vibium;
import com.vibium.types.StartOptions;
import org.junit.jupiter.api.AfterAll;
import org.junit.jupiter.api.BeforeAll;
import org.junit.jupiter.api.Test;

import static org.junit.jupiter.api.Assertions.assertEquals;

/** Page.setOffline tests. */
@RequiresCapability("core")
class OfflineTest {

    private static final String FETCHED = "fetch('/json').then(() => 'served').catch(() => 'failed')";

    static Browser browser;
    static TestServer server;

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

    @Test
    void takesThePageOfflineAndBackOnline() {
        Page page = browser.page();
        page.go(server.baseUrl());
        page.setOffline(true);
        try {
            assertEquals("failed", page.evaluate(FETCHED));
        } finally {
            page.setOffline(false);
        }
        assertEquals("served", page.evaluate(FETCHED));
    }
}
