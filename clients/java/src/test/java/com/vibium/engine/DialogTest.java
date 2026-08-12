package com.vibium.engine;

import com.vibium.Vibium;
import com.vibium.Page;
import com.vibium.Browser;

import com.vibium.types.StartOptions;
import org.junit.jupiter.api.*;

import java.util.concurrent.CountDownLatch;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.atomic.AtomicReference;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Dialog handling tests.
 */
@RequiresCapability("core")
class DialogTest {

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
    @RequiresCapability("dialogs")
    void testDialogAccept() throws Exception {
        page.go(server.baseUrl() + "/dialog");

        CountDownLatch latch = new CountDownLatch(1);
        AtomicReference<String> dialogMessage = new AtomicReference<>();

        page.onDialog(dialog -> {
            dialogMessage.set(dialog.message());
            dialog.accept();
            latch.countDown();
        });

        page.find("#alert-btn").click();
        assertTrue(latch.await(5, TimeUnit.SECONDS));
        assertEquals("hello", dialogMessage.get());
    }
}
