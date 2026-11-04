package com.vibium.engine;

import com.vibium.Vibium;
import com.vibium.Page;
import com.vibium.Element;
import com.vibium.Browser;

import com.vibium.types.FindOptions;
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
    void testFindNthAndHasText() {
        page.go(server.baseUrl() + "/filter-rows");
        Element el = page.find(".row", new FindOptions().nth(1));
        assertEquals("Beta one", el.info().text());
        Element last = page.find(".row", new FindOptions().last());
        assertEquals("Gamma", last.info().text());
        Element beta = page.find(".row", new FindOptions().hasText("Beta").nth(1));
        assertEquals("Beta two", beta.info().text());
        // The stored locator re-applies the filters after a re-render (#709).
        page.evaluate("prependRow('Beta zero')");
        assertEquals("Beta one", beta.text());
    }

    @Test
    void testFindAllHasText() {
        page.go(server.baseUrl() + "/filter-rows");
        List<Element> rows = page.findAll(".row", new FindOptions().hasText("Beta"));
        assertEquals(2, rows.size());
        // Index 0 re-selects from the re-filtered list, not the raw one.
        page.evaluate("prependRow('Beta zero')");
        assertEquals("Beta zero", rows.get(0).text());
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
