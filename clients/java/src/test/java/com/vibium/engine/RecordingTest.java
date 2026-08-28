package com.vibium.engine;

import com.vibium.Browser;
import com.vibium.BrowserContext;
import com.vibium.Page;
import com.vibium.Vibium;
import com.vibium.errors.VibiumException;
import com.vibium.types.RecordingOptions;
import com.vibium.types.RecordingResult;
import com.vibium.types.StartOptions;
import org.junit.jupiter.api.AfterAll;
import org.junit.jupiter.api.AfterEach;
import org.junit.jupiter.api.BeforeAll;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.io.TempDir;

import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.util.Map;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertFalse;
import static org.junit.jupiter.api.Assertions.assertNotNull;
import static org.junit.jupiter.api.Assertions.assertNull;
import static org.junit.jupiter.api.Assertions.assertThrows;
import static org.junit.jupiter.api.Assertions.assertTrue;
import static org.junit.jupiter.api.Assumptions.assumeFalse;
import static org.junit.jupiter.api.Assumptions.assumeTrue;

/**
 * Recording tests: start/stop, path handling, and the video track semantics
 * from RecordingOptions — explicit video fails at start on an engine without
 * support, best-effort video reports videoUnavailable instead. Video has no
 * capability manifest entry, so the video tests branch on VIBIUM_ENGINE the
 * way the JS chrome-video and firefox suites split the same coverage.
 */
@RequiresCapability("core")
class RecordingTest {

    static Browser browser;
    static TestServer server;
    static final String ENGINE =
        System.getenv("VIBIUM_ENGINE") == null ? "chrome" : System.getenv("VIBIUM_ENGINE");

    BrowserContext ctx;
    Page page;

    @TempDir
    Path tmp;

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
        ctx = browser.newContext();
        page = ctx.newPage();
        page.go(server.baseUrl());
    }

    @AfterEach
    void afterEach() {
        if (ctx != null) ctx.close();
    }

    @Test
    void startStopWritesZip() throws Exception {
        String path = tmp.resolve("basic.zip").toString();
        ctx.recording().start(new RecordingOptions().video(false).path(path));
        page.go(server.baseUrl());

        RecordingResult result = ctx.recording().stop();

        assertEquals(path, result.path());
        assertNull(result.bytes(), "bytes come back only when no file was written");
        assertTrue(Files.exists(Paths.get(path)));
        byte[] head = new byte[2];
        try (var in = Files.newInputStream(Paths.get(path))) {
            assertEquals(2, in.read(head));
        }
        assertEquals('P', head[0]);
        assertEquals('K', head[1]);
        assertNotNull(result.durationMs());
        assertTrue(result.durationMs() >= 0);
    }

    @Test
    void stopPathOverridesStartPath() {
        String startPath = tmp.resolve("declared.zip").toString();
        String stopPath = tmp.resolve("overridden.zip").toString();
        ctx.recording().start(new RecordingOptions().video(false).path(startPath));
        page.go(server.baseUrl());

        RecordingResult result = ctx.recording().stop(stopPath);

        assertEquals(stopPath, result.path());
        assertTrue(Files.exists(Paths.get(stopPath)));
        assertFalse(Files.exists(Paths.get(startPath)));
    }

    @Test
    void requiredVideoFailsAtStartWhereUnsupported() {
        assumeFalse("firefox".equals(ENGINE));

        String path = tmp.resolve("never-written.zip").toString();
        VibiumException err = assertThrows(VibiumException.class, () ->
            ctx.recording().start(new RecordingOptions().video(true).path(path)));

        assertTrue(err.getMessage().contains("not supported by this browser"),
            "expected the actionable video-unsupported error, got: " + err.getMessage());
        assertFalse(Files.exists(Paths.get(path)));
    }

    @Test
    void bestEffortVideoReportsUnavailableWhereUnsupported() {
        assumeFalse("firefox".equals(ENGINE));

        String path = tmp.resolve("trace-only.zip").toString();
        ctx.recording().start(new RecordingOptions().path(path));
        page.go(server.baseUrl());

        RecordingResult result = ctx.recording().stop();

        assertTrue(result.videos().isEmpty());
        assertNotNull(result.videoUnavailable(),
            "an engine without video support must say why the track is missing");
        assertTrue(Files.exists(Paths.get(path)), "the trace still lands without a video track");
    }

    @Test
    void videoTrackCarriesDimensions() {
        assumeTrue("firefox".equals(ENGINE));

        String path = tmp.resolve("video.zip").toString();
        ctx.recording().start(new RecordingOptions().video(true).path(path));
        page.go(server.baseUrl());
        page.sleep(500);

        RecordingResult result = ctx.recording().stop();

        assertNull(result.videoUnavailable());
        assertEquals(1, result.videos().size(), "one page recorded, one video track");
        Map<String, Object> video = result.videos().get(0);
        assertTrue(((Number) video.get("width")).intValue() > 0);
        assertTrue(((Number) video.get("height")).intValue() > 0);
        assertTrue(((Number) video.get("durationMs")).longValue() > 0);
        assertTrue(Files.exists(Paths.get(path)));
    }
}
