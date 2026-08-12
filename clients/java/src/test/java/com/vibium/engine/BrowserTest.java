package com.vibium.engine;

import com.vibium.Vibium;
import com.vibium.Page;
import com.vibium.BrowserContext;
import com.vibium.Browser;

import com.vibium.types.StartOptions;
import org.junit.jupiter.api.*;

import java.util.List;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Browser lifecycle tests.
 */
@TestMethodOrder(MethodOrderer.OrderAnnotation.class)
@RequiresCapability("core")
class BrowserTest {

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
    @Order(1)
    void testPage() {
        Page page = browser.page();
        assertNotNull(page);
        assertNotNull(page.id());
    }

    @Test
    @Order(2)
    void testNewPage() {
        Page page = browser.newPage();
        assertNotNull(page);
    }

    @Test
    @Order(3)
    void testPages() {
        List<Page> pages = browser.pages();
        assertNotNull(pages);
        assertTrue(pages.size() >= 1);
    }

    @Test
    @Order(4)
    void testContext() {
        BrowserContext ctx = browser.context();
        assertNotNull(ctx);
    }
}
