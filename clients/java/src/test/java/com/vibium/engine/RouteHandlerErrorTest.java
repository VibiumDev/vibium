package com.vibium.engine;

import com.vibium.Browser;
import com.vibium.Download;
import com.vibium.Page;
import com.vibium.Route;
import com.vibium.Vibium;
import com.vibium.errors.VibiumConnectionException;
import com.vibium.types.FulfillOptions;
import com.vibium.types.SelectorOptions;
import com.vibium.types.StartOptions;
import org.junit.jupiter.api.AfterAll;
import org.junit.jupiter.api.BeforeAll;
import org.junit.jupiter.api.Test;

import java.io.ByteArrayOutputStream;
import java.io.PrintStream;
import java.util.Map;
import java.util.function.Consumer;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertThrows;
import static org.junit.jupiter.api.Assertions.assertTrue;

@RequiresCapability("core")
class RouteHandlerErrorTest {

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
    void reportsTheErrorAndContinuesTheRequest() {
        Page page = browser.newPage();
        page.go(server.baseUrl());
        PrintStream original = System.err;
        ByteArrayOutputStream captured = new ByteArrayOutputStream();
        System.setErr(new PrintStream(captured, true));
        try {
            page.route("**/api/data", route -> {
                throw new IllegalStateException("boom");
            });
            Object result = page.evaluate(
                "Promise.race([fetch('/api/data').then(r => r.text()),"
                    + " new Promise(resolve => setTimeout(() => resolve('still pending'), 3000))])");
            assertTrue(!"still pending".equals(String.valueOf(result)), "request stayed pending");
        } finally {
            System.setErr(original);
            page.close();
        }
        assertTrue(captured.toString().contains("vibium: error in route handler: boom"), captured.toString());
    }

    private static final String FETCH = "Promise.race([fetch('/api/data').then(r => r.text()),"
        + " new Promise(resolve => setTimeout(() => resolve('still pending'), 3000))])";

    /** Runs a fetch through a handler on the default page and returns the result and stderr. */
    private static String[] fetchThrough(Consumer<Route> handler) {
        Page page = browser.page();
        page.go(server.baseUrl());
        PrintStream original = System.err;
        ByteArrayOutputStream captured = new ByteArrayOutputStream();
        System.setErr(new PrintStream(captured, true));
        try {
            page.route("**/api/data", handler);
            return new String[] {String.valueOf(page.evaluate(FETCH)), captured.toString()};
        } finally {
            System.setErr(original);
            page.unroute("**/api/data");
        }
    }

    /** A fulfill the browser rejects reaches the caller, which reports it and continues (#857). */
    @Test
    void aFailedFulfillIsReportedAndTheRequestContinues() {
        String[] seen = fetchThrough(route ->
            route.fulfill(new FulfillOptions().headers(Map.of("Bad Header", "x")).body("mock")));
        assertTrue(seen[0].contains("real data"), "request was not continued: " + seen[0]);
        assertTrue(seen[1].contains("vibium: error in route handler:") && seen[1].contains("invalid argument"), seen[1]);
    }

    /** Acting on a request that is already handled stays silent (#857). */
    @Test
    void actingOnAHandledRequestIsIgnored() {
        String[] seen = fetchThrough(route -> {
            route.doContinue();
            route.doContinue();
            route.fulfill(new FulfillOptions().body("late"));
            route.abort();
        });
        assertTrue(seen[0].contains("real data"), seen[0]);
        assertEquals("", seen[1]);
    }

    /** A dead connection during Download.path() is an error, not a null path (#857). */
    @Test
    @RequiresCapability("downloads")
    void downloadPathReportsADeadConnection() {
        Browser own = Vibium.start(new StartOptions().headless(true));
        Download download;
        try {
            Page page = own.page();
            page.go(server.baseUrl() + "/download");
            download = page.capture().download(
                () -> page.find(new SelectorOptions().css("#download-link")).click());
        } finally {
            own.stop();
        }
        assertThrows(VibiumConnectionException.class, download::path);
    }
}
