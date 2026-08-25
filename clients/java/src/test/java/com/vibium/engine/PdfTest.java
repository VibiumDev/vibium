package com.vibium.engine;

import com.vibium.Browser;
import com.vibium.Page;
import com.vibium.Vibium;
import com.vibium.types.PdfOptions;
import com.vibium.types.StartOptions;
import org.junit.jupiter.api.*;

import java.nio.charset.StandardCharsets;
import java.util.regex.Matcher;
import java.util.regex.Pattern;

import static org.junit.jupiter.api.Assertions.*;

/**
 * PDF print options (#72).
 */
@RequiresCapability("pdf")
class PdfTest {

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

    // The page dimensions land uncompressed in "/MediaBox [0 0 <w> <h>]".
    private static double[] mediaBox(byte[] pdf) {
        Matcher m = Pattern.compile("/MediaBox \\[0 0 ([\\d.]+) ([\\d.]+)\\]")
            .matcher(new String(pdf, StandardCharsets.ISO_8859_1));
        assertTrue(m.find(), "PDF should contain a parseable /MediaBox");
        return new double[]{Double.parseDouble(m.group(1)), Double.parseDouble(m.group(2))};
    }

    @Test
    void landscapeOptionSwapsOrientation() {
        page.setContent("<p>hello</p>");

        double[] portrait = mediaBox(page.pdf());
        double[] landscape = mediaBox(page.pdf(new PdfOptions().landscape(true)));

        assertTrue(portrait[1] > portrait[0], "portrait should be taller than wide");
        assertTrue(landscape[0] > landscape[1], "landscape should be wider than tall");
    }
}
