package com.vibium.engine;

import com.vibium.Vibium;
import com.vibium.Page;
import com.vibium.Browser;

import com.vibium.types.A11yNode;
import com.vibium.types.StartOptions;
import org.junit.jupiter.api.*;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Accessibility tree tests.
 */
@RequiresCapability("core")
class A11yTest {

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
    void testA11yTree() {
        page.go(server.baseUrl() + "/a11y");
        A11yNode tree = page.a11yTree();
        assertNotNull(tree);
        assertNotNull(tree.role());
    }
}
