package com.vibium.engine;

import com.vibium.Browser;
import com.vibium.Page;
import com.vibium.Vibium;
import com.vibium.types.StartOptions;
import org.junit.jupiter.api.AfterAll;
import org.junit.jupiter.api.BeforeAll;
import org.junit.jupiter.api.Test;

import static org.junit.jupiter.api.Assertions.assertEquals;

/** A route handler must decide the request on any page, not only the default one. */
@RequiresCapability("core")
class RouteOnNewPageTest {

    private static final String FETCH = "Promise.race([fetch('/api/data').then(r => 'served').catch(e => 'aborted'),"
        + " new Promise(resolve => setTimeout(() => resolve('still pending'), 3000))])";

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

    private String fetchWithAbortRoute() {
        Page page = browser.newPage();
        try {
            page.go(server.baseUrl());
            page.route("**/api/data", route -> route.abort());
            return String.valueOf(page.evaluate(FETCH));
        } finally {
            page.close();
        }
    }

    @Test
    void abortTakesEffectOnANewPage() {
        assertEquals("aborted", fetchWithAbortRoute());
    }

    @Test
    void abortTakesEffectWhileAPageListenerHoldsASecondPageObject() {
        browser.onPage(page -> { });
        try {
            assertEquals("aborted", fetchWithAbortRoute());
        } finally {
            browser.removeAllListeners("page");
        }
    }
}
