package com.vibium.engine;

import com.vibium.Browser;
import com.vibium.Page;
import com.vibium.Vibium;
import com.vibium.types.StartOptions;
import org.junit.jupiter.api.AfterAll;
import org.junit.jupiter.api.BeforeAll;
import org.junit.jupiter.api.Test;

import java.io.ByteArrayOutputStream;
import java.io.PrintStream;

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
}
