package com.vibium;

import static org.junit.jupiter.api.Assertions.*;

import com.vibium.types.StartOptions;
import com.vibium.types.WaitOptions;
import java.util.concurrent.atomic.AtomicBoolean;
import org.junit.jupiter.api.AfterAll;
import org.junit.jupiter.api.BeforeAll;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.TestInstance;

/** Verifies the Java-cluster bug fixes (#129, #130, #132, #133, #134, #136, #137). */
@TestInstance(TestInstance.Lifecycle.PER_CLASS)
public class JavaClusterFixTest {

    private Browser browser;

    @BeforeAll
    void setup() {
        browser = Vibium.start(new StartOptions().headless(true));
    }

    @AfterAll
    void teardown() {
        if (browser != null) browser.stop();
    }

    @Test
    void addScriptExecutes() { // #130
        Page page = browser.page();
        page.setContent("<html><body></body></html>");
        page.addScript("window.__test = 'hello';");
        assertEquals("hello", page.evaluate("window.__test"));
    }

    @Test
    void dispatchEventTriggersHandler() { // #132
        Page page = browser.page();
        page.setContent("<html><body><div id='t' onclick=\"this.dataset.fired='1'\">x</div></body></html>");
        page.find("#t").dispatchEvent("click");
        assertEquals("1", page.evaluate("document.querySelector('#t').dataset.fired"));
    }

    @Test
    void clockSetFixedTimeFreezesDate() { // #137
        Page page = browser.page();
        page.setContent("<html><body></body></html>");
        page.clock().install();
        page.clock().setFixedTime("2020-01-01T00:00:00.000Z");
        Object now = page.evaluate("Date.now()");
        assertEquals(1577836800000L, ((Number) now).longValue());
    }

    @Test
    void dragToDoesNotRejectTarget() { // #134
        Page page = browser.page();
        page.setContent(
            "<html><body><div id='src' draggable='true'>A</div><div id='dst'>B</div></body></html>");
        // Must not throw "dragTo requires 'target' parameter".
        assertDoesNotThrow(() -> page.find("#src").dragTo(page.find("#dst")));
    }

    @Test
    void highlightIsImplemented() { // #133
        Page page = browser.page();
        page.setContent("<html><body><a id='a' href='#'>link</a></body></html>");
        // Must not throw "Unknown command 'vibium:element.highlight'".
        assertDoesNotThrow(() -> page.find("#a").highlight());
    }

    @Test
    void waitForURLDoesNotRejectPattern() { // #129
        Page page = browser.page();
        page.setContent("<html><body>hi</body></html>");
        try {
            page.waitForURL("**/never-matches-xyz", new WaitOptions().timeout(300));
            fail("expected a timeout");
        } catch (Exception e) {
            assertFalse(
                e.getMessage() != null && e.getMessage().contains("pattern is required"),
                "should not reject the pattern: " + e.getMessage());
        }
    }

    @Test
    void onErrorReceivesUncaughtError() throws Exception { // #136
        Page page = browser.page();
        page.setContent("<html><body></body></html>");
        AtomicBoolean fired = new AtomicBoolean(false);
        page.onError(e -> fired.set(true));
        page.evaluate("setTimeout(() => { throw new Error('boom'); }, 0)");
        for (int i = 0; i < 50 && !fired.get(); i++) {
            Thread.sleep(100);
        }
        assertTrue(fired.get(), "onError should fire for an uncaught page error");
    }
}
