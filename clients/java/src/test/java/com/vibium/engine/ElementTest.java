package com.vibium;

import com.vibium.types.BoundingBox;
import com.vibium.types.StartOptions;
import org.junit.jupiter.api.*;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Element interaction and query tests.
 */
@RequiresCapability("core")
class ElementTest {

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
    void testClick() {
        page.go(server.baseUrl() + "/links");
        Element link = page.find(".link");
        link.click();
        page.waitForLoad();
        // Should navigate to subpage
        assertTrue(page.url().contains("/subpage"));
    }

    @Test
    void testFill() {
        page.go(server.baseUrl() + "/inputs");
        Element input = page.find("#text-input");
        input.fill("hello world");
        assertEquals("hello world", input.value());
    }

    @Test
    void testText() {
        page.go(server.baseUrl());
        Element heading = page.find("h1");
        assertTrue(heading.text().contains("Welcome"));
    }

    @Test
    void testAttr() {
        page.go(server.baseUrl());
        Element link = page.find("a[href='/subpage']");
        assertEquals("/subpage", link.attr("href"));
        // Alias
        assertEquals("/subpage", link.getAttribute("href"));
    }

    @Test
    void testIsVisible() {
        page.go(server.baseUrl());
        Element heading = page.find("h1");
        assertTrue(heading.isVisible());
    }

    @Test
    void testBounds() {
        page.go(server.baseUrl());
        Element heading = page.find("h1");
        BoundingBox box = heading.bounds();
        assertNotNull(box);
        assertTrue(box.width() > 0);
        assertTrue(box.height() > 0);
        // Alias
        BoundingBox box2 = heading.boundingBox();
        assertEquals(box.x(), box2.x());
    }

    @Test
    void testCheckUncheck() {
        page.go(server.baseUrl() + "/form");
        Element checkbox = page.find("#agree");
        checkbox.check();
        assertTrue(checkbox.isChecked());
        checkbox.uncheck();
        assertFalse(checkbox.isChecked());
    }

    @Test
    void testSelectOption() {
        page.go(server.baseUrl() + "/form");
        Element select = page.find("#color");
        select.selectOption("blue");
        assertEquals("blue", select.value());
    }

    @Test
    void testScreenshot() {
        page.go(server.baseUrl());
        Element heading = page.find("h1");
        byte[] data = heading.screenshot();
        assertNotNull(data);
        assertTrue(data.length > 0);
    }
}
