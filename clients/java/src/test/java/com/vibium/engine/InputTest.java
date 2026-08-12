package com.vibium.engine;

import com.vibium.Vibium;
import com.vibium.Page;
import com.vibium.Browser;

import com.vibium.types.StartOptions;
import org.junit.jupiter.api.*;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Keyboard, mouse, and touch input tests.
 */
@RequiresCapability("core")
class InputTest {

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
    void testKeyboardType() {
        page.go(server.baseUrl() + "/inputs");
        page.find("#text-input").click();
        page.keyboard().type("hello");
        assertEquals("hello", page.find("#text-input").value());
    }

    @Test
    void testKeyboardPress() {
        page.go(server.baseUrl() + "/inputs");
        page.find("#text-input").fill("abc");
        page.find("#text-input").click();
        page.keyboard().press("Backspace");
        assertEquals("ab", page.find("#text-input").value());
    }

    @Test
    void testEvaluate() {
        page.go(server.baseUrl() + "/eval");
        Object result = page.evaluate("window.testVal");
        assertEquals(42, result);
    }

    @Test
    void testEvaluateExpression() {
        page.go(server.baseUrl() + "/eval");
        Object result = page.evaluate("1 + 1");
        assertEquals(2, result);
    }

    @Test
    void testScreenshot() {
        page.go(server.baseUrl());
        byte[] data = page.screenshot();
        assertNotNull(data);
        assertTrue(data.length > 100); // Should be a real PNG
    }
}
