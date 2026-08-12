package com.vibium.engine;

import com.vibium.Vibium;
import com.vibium.Page;
import com.vibium.Browser;

import com.vibium.types.StartOptions;
import org.junit.jupiter.api.*;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Navigation tests — go, back, forward, reload, url, title, content.
 */
@RequiresCapability("core")
class PageNavigationTest {

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
    }

    @Test
    void testGo() {
        page.go(server.baseUrl());
        String url = page.url();
        assertTrue(url.contains("127.0.0.1"));
    }

    @Test
    void testBackForward() {
        page.go(server.baseUrl());
        page.go(server.baseUrl() + "/subpage");
        assertEquals("Subpage", page.title());
        page.back();
        assertEquals("Test App", page.title());
        page.forward();
        assertEquals("Subpage", page.title());
    }

    @Test
    void testReload() {
        page.go(server.baseUrl());
        page.reload();
        assertEquals("Test App", page.title());
    }

    @Test
    void testUrl() {
        page.go(server.baseUrl() + "/subpage");
        assertTrue(page.url().contains("/subpage"));
    }

    @Test
    void testTitle() {
        page.go(server.baseUrl());
        assertEquals("Test App", page.title());
    }

    @Test
    void testContent() {
        page.go(server.baseUrl());
        String html = page.content();
        assertTrue(html.contains("Welcome to test-app"));
    }
}
