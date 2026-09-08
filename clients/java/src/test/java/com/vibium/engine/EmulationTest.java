package com.vibium.engine;

import com.vibium.Vibium;
import com.vibium.Page;
import com.vibium.Browser;

import com.vibium.types.*;
import org.junit.jupiter.api.*;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Viewport, media emulation, and geolocation tests.
 */
@RequiresCapability("core")
class EmulationTest {

    static Browser browser;
    static TestServer server;
    Page page;

    @BeforeAll
    static void setup() throws Exception {
        server = new TestServer();
        server.start();
        browser = Vibium.start(new StartOptions().headless(true));
        // Firefox keeps the startup tab in the parent process until a
        // navigation, where script-backed commands are refused.
        browser.page().go("about:blank");
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
    void testSetViewport() {
        page.setViewport(new ViewportSize(800, 600));
        ViewportSize vp = page.viewport();
        assertEquals(800, vp.width());
        assertEquals(600, vp.height());
    }

    @Test
    void testEmulateMedia() {
        page.go(server.baseUrl());
        page.emulateMedia(new MediaOptions().colorScheme("dark"));
        // Verify via evaluate
        Object scheme = page.evaluate("window.matchMedia('(prefers-color-scheme: dark)').matches");
        assertEquals(true, scheme);
    }

    @Test
    void testSetContent() {
        page.setContent("<html><body><h1>Custom</h1></body></html>");
        assertEquals("Custom", page.find("h1").text());
    }

    @Test
    void testWindow() {
        WindowInfo info = page.window();
        assertNotNull(info);
        assertTrue(info.width() > 0);
        assertTrue(info.height() > 0);
    }
}
