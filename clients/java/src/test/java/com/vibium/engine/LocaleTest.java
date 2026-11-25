package com.vibium.engine;

import com.vibium.Browser;
import com.vibium.Page;
import com.vibium.Vibium;
import com.vibium.types.StartOptions;
import org.junit.jupiter.api.AfterAll;
import org.junit.jupiter.api.BeforeAll;
import org.junit.jupiter.api.Test;

import static org.junit.jupiter.api.Assertions.assertEquals;

/** Page.setLocale tests. */
@RequiresCapability("core")
class LocaleTest {

    private static final String FORMATTED = "(1234.5).toLocaleString()";

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
    void setsAndRemovesTheLocale() {
        Page page = browser.page();
        page.go(server.baseUrl());
        Object before = page.evaluate(FORMATTED);
        page.setLocale("de-DE");
        try {
            assertEquals("de-DE", page.evaluate("navigator.language"));
            assertEquals("1.234,5", page.evaluate(FORMATTED));
        } finally {
            page.setLocale(null);
        }
        assertEquals(before, page.evaluate(FORMATTED));
    }
}
