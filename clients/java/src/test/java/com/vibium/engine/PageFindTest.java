package com.vibium.engine;

import com.vibium.Vibium;
import com.vibium.Page;
import com.vibium.Element;
import com.vibium.Browser;

import com.vibium.types.StartOptions;
import org.junit.jupiter.api.*;

import java.util.List;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Find and waitFor tests.
 */
@RequiresCapability("core")
class PageFindTest {

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
    void testFindBySelector() {
        page.go(server.baseUrl());
        Element heading = page.find("h1");
        assertNotNull(heading);
        assertTrue(heading.text().contains("Welcome"));
    }

    @Test
    void testFindAll() {
        page.go(server.baseUrl() + "/links");
        List<Element> links = page.findAll(".link");
        assertEquals(4, links.size());
    }

    @Test
    void testWaitFor() {
        page.go(server.baseUrl() + "/dynamic-loading");
        Element loaded = page.waitFor("#loaded");
        assertNotNull(loaded);
        assertEquals("Loaded!", loaded.text());
    }

    @Test
    void testWaitForLoad() {
        page.go(server.baseUrl());
        page.waitForLoad();
        assertEquals("Test App", page.title());
    }
}
