package com.vibium.engine;

import com.vibium.Browser;
import com.vibium.Page;
import com.vibium.Vibium;
import com.vibium.types.StartOptions;
import org.junit.jupiter.api.*;

import static org.junit.jupiter.api.Assertions.*;

/**
 * page.expose defines a named JS function in the page, matching the JS and
 * Python clients. The previous Java signature was non-functional (#298).
 */
@RequiresCapability("core")
class ExposeTest {

    static Browser browser;
    Page page;

    @BeforeAll
    static void setup() {
        browser = Vibium.start(new StartOptions().headless(true));
        // Firefox keeps the startup tab in the parent process until a
        // navigation, where script-backed commands are refused.
        browser.page().go("about:blank");
    }

    @AfterAll
    static void teardown() {
        if (browser != null) browser.stop();
    }

    @BeforeEach
    void beforeEach() {
        page = browser.page();
    }

    @Test
    void exposeDefinesAFunctionThePageCanCall() {
        page.setContent("<p>x</p>");
        page.expose("vibiumDouble", "(n) => n * 2");

        Object result = page.evaluate("window.vibiumDouble(4)");
        assertEquals(8.0, ((Number) result).doubleValue());
    }

    @Test
    void exposeSurvivesAReload() {
        page.setContent("<p>x</p>");
        page.expose("vibiumMark", "() => 'marked'");
        page.reload();

        assertEquals("marked", page.evaluate("window.vibiumMark()"));
    }
}
