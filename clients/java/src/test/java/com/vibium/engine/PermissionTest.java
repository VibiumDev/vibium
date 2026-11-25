package com.vibium.engine;

import com.vibium.Browser;
import com.vibium.Page;
import com.vibium.Vibium;
import com.vibium.types.StartOptions;
import org.junit.jupiter.api.AfterAll;
import org.junit.jupiter.api.BeforeAll;
import org.junit.jupiter.api.Test;

import static org.junit.jupiter.api.Assertions.assertEquals;

/** Page.setPermission tests. */
@RequiresCapability("core")
class PermissionTest {

    private static final String STATE = "navigator.permissions.query({ name: 'geolocation' }).then((s) => s.state)";

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
    void grantsDeniesAndResetsAPermission() {
        Page page = browser.page();
        page.go(server.baseUrl());
        page.setPermission("geolocation", "granted");
        assertEquals("granted", page.evaluate(STATE));
        page.setPermission("geolocation", "denied");
        assertEquals("denied", page.evaluate(STATE));
        page.setPermission("geolocation", "prompt", server.baseUrl());
        assertEquals("prompt", page.evaluate(STATE));
    }
}
