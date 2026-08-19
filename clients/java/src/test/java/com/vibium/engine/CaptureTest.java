package com.vibium.engine;

import com.vibium.Browser;
import com.vibium.ConsoleMessage;
import com.vibium.Dialog;
import com.vibium.Download;
import com.vibium.Page;
import com.vibium.Request;
import com.vibium.Response;
import com.vibium.Vibium;
import com.vibium.errors.VibiumTimeoutException;
import com.vibium.types.StartOptions;
import com.vibium.types.WaitOptions;
import org.junit.jupiter.api.AfterAll;
import org.junit.jupiter.api.BeforeAll;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertInstanceOf;
import static org.junit.jupiter.api.Assertions.assertThrows;
import static org.junit.jupiter.api.Assertions.assertTrue;

/** One-shot capture parity with the JavaScript and Python clients. */
@RequiresCapability("core")
class CaptureTest {

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
        page.go(server.baseUrl() + "/fetch");
    }

    @Test
    @RequiresCapability("network")
    void capturesRequestAndResponse() {
        Request request = page.capture().request("**/api/data",
            () -> page.evaluate("doFetch()"));
        assertTrue(request.url().endsWith("/api/data"));

        Response response = page.capture().response("**/api/data",
            () -> page.evaluate("doFetch()"));
        assertEquals(200, response.status());
    }

    @Test
    @RequiresCapability("navigation-capture")
    void capturesNavigation() {
        String url = page.capture().navigation(
            () -> page.go(server.baseUrl() + "/subpage"));
        assertTrue(url.endsWith("/subpage"));
    }

    @Test
    @RequiresCapability("downloads")
    void capturesDownload() {
        page.go(server.baseUrl() + "/download");
        Download download = page.capture().download(
            () -> page.find("#download-link").click());
        assertEquals("test.txt", download.suggestedFilename());
    }

    @Test
    @RequiresCapability("dialogs")
    void dialogCaptureDoesNotDeadlockWhenActionBlocks() {
        Dialog dialog = page.capture().dialog(
            () -> page.evaluate("alert('captured')"),
            new WaitOptions().timeout(5000));

        assertEquals("captured", dialog.message());
        dialog.accept();
        assertEquals("Fetch", page.title());
    }

    @Test
    @RequiresCapability("console")
    void capturesNamedEvent() {
        Object event = page.capture().event("console",
            () -> page.evaluate("console.log('captured console')"));
        ConsoleMessage message = assertInstanceOf(ConsoleMessage.class, event);
        assertTrue(message.text().contains("captured console"));
    }

    @Test
    void propagatesActionFailureAndCleansListener() {
        IllegalStateException error = assertThrows(IllegalStateException.class,
            () -> page.capture().navigation(() -> {
                throw new IllegalStateException("action failed");
            }));
        assertEquals("action failed", error.getMessage());

        String url = page.capture().navigation(
            () -> page.go(server.baseUrl() + "/subpage"));
        assertTrue(url.endsWith("/subpage"));
    }

    @Test
    void timeoutCleansListener() {
        assertThrows(VibiumTimeoutException.class,
            () -> page.capture().navigation(
                () -> {}, new WaitOptions().timeout(50)));

        String url = page.capture().navigation(
            () -> page.go(server.baseUrl() + "/subpage"));
        assertTrue(url.endsWith("/subpage"));
    }
}
