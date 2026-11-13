package com.vibium.engine;

import com.vibium.Browser;
import com.vibium.Page;
import com.vibium.Vibium;
import com.vibium.types.StartOptions;
import org.junit.jupiter.api.*;

import static org.junit.jupiter.api.Assertions.*;

/**
 * page.expose in both forms (#298): a string defines window[name] from JS
 * source inside the page; a callback exposes a host function whose return
 * value comes back to the page through vibium:expose.call/result.
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

    @Test
    void hostFunctionRunsAndReturnsItsValue() {
        page.setContent("<p>x</p>");
        page.expose("vibiumAdd", (args) ->
            ((Number) args[0]).doubleValue() + ((Number) args[1]).doubleValue());

        Object result = page.evaluate("window.vibiumAdd(2, 3)");
        assertEquals(5.0, ((Number) result).doubleValue());
    }

    @Test
    void hostFunctionErrorRejectsThePagePromise() {
        page.setContent("<p>x</p>");
        page.expose("vibiumExplode", (args) -> {
            throw new IllegalStateException("no fuel");
        });

        Object message = page.evaluate("window.vibiumExplode().catch((e) => e.message)");
        assertEquals("no fuel", message);
    }

    @Test
    void hostFunctionSurvivesNavigationAndReExposureReplaces() {
        page.setContent("<p>x</p>");
        page.expose("vibiumAnswer", (args) -> "first");
        page.expose("vibiumAnswer", (args) -> "second");
        page.reload();

        assertEquals("second", page.evaluate("window.vibiumAnswer()"));
    }
}
