package com.vibium.engine;

import com.vibium.Vibium;
import com.vibium.Page;
import com.vibium.BrowserContext;
import com.vibium.Browser;

import com.vibium.types.Cookie;
import com.vibium.types.SetCookieParam;
import com.vibium.types.StartOptions;
import org.junit.jupiter.api.*;

import java.util.Collections;
import java.util.List;

import static org.junit.jupiter.api.Assertions.*;

/**
 * BrowserContext tests — cookies, storage.
 */
@RequiresCapability("core")
class ContextTest {

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
    void testCookies() {
        Page page = browser.page();
        page.go(server.baseUrl() + "/set-cookie");

        BrowserContext ctx = page.context();
        List<Cookie> cookies = ctx.cookies();
        assertNotNull(cookies);
        assertTrue(cookies.size() >= 1);
    }

    @Test
    void testSetCookies() {
        BrowserContext ctx = browser.context();
        ctx.setCookies(Collections.singletonList(
            new SetCookieParam("java_test", "works").domain("127.0.0.1").path("/")
        ));

        List<Cookie> cookies = ctx.cookies();
        boolean found = cookies.stream().anyMatch(c -> "java_test".equals(c.name()));
        assertTrue(found, "Should find the cookie we set");
    }

    @Test
    void testClearCookies() {
        BrowserContext ctx = browser.context();
        ctx.clearCookies();
        List<Cookie> cookies = ctx.cookies();
        assertTrue(cookies.isEmpty() || cookies.size() == 0);
    }
}
