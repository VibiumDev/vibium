package com.vibium;

import org.junit.jupiter.api.Assumptions;
import org.junit.jupiter.api.BeforeAll;
import org.junit.jupiter.api.Test;

import java.io.BufferedReader;
import java.io.File;
import java.io.InputStreamReader;
import java.net.Socket;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.ArrayList;
import java.util.List;
import java.util.concurrent.CompletableFuture;
import java.util.concurrent.TimeUnit;

import static org.junit.jupiter.api.Assertions.*;

public class VibiumIntegrationIT {
    @BeforeAll
    public static void maybeSkipIfDownloadsDisabled() throws Exception {
        boolean skipClickerDownload = "1".equals(System.getenv("VIBIUM_SKIP_CLICKER_DOWNLOAD"));
        boolean skipBrowserDownload = "1".equals(System.getenv("VIBIUM_SKIP_BROWSER_DOWNLOAD"));

        if (!skipClickerDownload && !skipBrowserDownload) {
            return;
        }

        String clickerPath = resolveClickerBinaryNoDownload();
        Assumptions.assumeTrue(clickerPath != null, "clicker binary not found and downloads disabled");

        if (skipBrowserDownload) {
            Assumptions.assumeTrue(isBrowserInstalled(clickerPath), "Chrome/Chromedriver not installed and VIBIUM_SKIP_BROWSER_DOWNLOAD=1");
        }
    }

    @Test
    public void canNavigateAndScreenshot() throws Exception {
        LaunchOptions opts = new LaunchOptions()
                .headless(true)
                .timeoutMs(120_000);

        try (Vibe vibe = Browser.launch(opts)) {
            assertTrue(vibe.port() > 0);
            vibe.go("https://the-internet.herokuapp.com/");
            byte[] screenshot = vibe.screenshot();
            assertNotNull(screenshot);
            assertTrue(screenshot.length > 1000, "screenshot too small");
            assertEquals((byte) 0x89, screenshot[0]);
            assertEquals((byte) 0x50, screenshot[1]);
            assertEquals((byte) 0x4E, screenshot[2]);
            assertEquals((byte) 0x47, screenshot[3]);
        }
    }

    @Test
    public void canFindClickAndNavigate() {
        LaunchOptions opts = new LaunchOptions()
                .headless(true)
                .timeoutMs(120_000);

        try (Vibe vibe = Browser.launch(opts)) {
            vibe.go("https://the-internet.herokuapp.com/");
            Element link = vibe.find("a[href=\"/add_remove_elements/\"]", Vibe.FindOptions.timeoutMs(30_000));
            link.click(Element.ActionOptions.timeoutMs(30_000));

            Element heading = vibe.find("h3", Vibe.FindOptions.timeoutMs(30_000));
            assertTrue(heading.info().text.contains("Add/Remove Elements"), "did not navigate: " + heading.info().text);
        }
    }

    @Test
    public void canTypeIntoInput() {
        LaunchOptions opts = new LaunchOptions()
                .headless(true)
                .timeoutMs(120_000);

        try (Vibe vibe = Browser.launch(opts)) {
            vibe.go("https://the-internet.herokuapp.com/inputs");
            Element input = vibe.find("input", Vibe.FindOptions.timeoutMs(30_000));
            input.type("12345", Element.ActionOptions.timeoutMs(30_000));

            String value = vibe.evaluate("return document.querySelector('input').value");
            assertEquals("12345", value);
        }
    }

    @Test
    public void autoWaitFindsDynamicContent() {
        LaunchOptions opts = new LaunchOptions()
                .headless(true)
                .timeoutMs(120_000);

        try (Vibe vibe = Browser.launch(opts)) {
            vibe.go("https://the-internet.herokuapp.com/dynamic_loading/1");
            Element start = vibe.find("#start button", 10_000);
            start.click(10_000);

            Element finish = vibe.find("#finish h4", 20_000);
            assertEquals("Hello World!", finish.info().text);
        }
    }

    @Test
    public void findTimesOutForMissingSelector() {
        LaunchOptions opts = new LaunchOptions()
                .headless(true)
                .timeoutMs(120_000);

        try (Vibe vibe = Browser.launch(opts)) {
            vibe.go("https://the-internet.herokuapp.com/");
            VibiumTimeoutException e = assertThrows(VibiumTimeoutException.class, () -> vibe.find("#does-not-exist", 1000));
            assertEquals(1000, e.timeoutMs);
            assertEquals("vibium:find", e.operation);
        }
    }

    @Test
    public void clickTimesOutWhenElementDisappears() {
        LaunchOptions opts = new LaunchOptions()
                .headless(true)
                .timeoutMs(120_000);

        try (Vibe vibe = Browser.launch(opts)) {
            vibe.go("https://the-internet.herokuapp.com/add_remove_elements/");

            Element add = vibe.find("button[onclick=\"addElement()\"]", 10_000);
            add.click(10_000);

            Element delete = vibe.find(".added-manually", 10_000);
            vibe.evaluate("document.querySelector('.added-manually')?.remove()");

            VibiumTimeoutException e = assertThrows(VibiumTimeoutException.class, () -> delete.click(1000));
            assertEquals(1000, e.timeoutMs);
            assertEquals("vibium:click", e.operation);
        }
    }

    @Test
    public void typeTimesOutWhenElementDisappears() {
        LaunchOptions opts = new LaunchOptions()
                .headless(true)
                .timeoutMs(120_000);

        try (Vibe vibe = Browser.launch(opts)) {
            vibe.go("https://the-internet.herokuapp.com/inputs");

            Element input = vibe.find("input", 10_000);
            vibe.evaluate("document.querySelector('input')?.remove()");

            VibiumTimeoutException e = assertThrows(VibiumTimeoutException.class, () -> input.type("x", 1000));
            assertEquals(1000, e.timeoutMs);
            assertEquals("vibium:type", e.operation);
        }
    }

    @Test
    public void elementTextThrowsWhenElementNoLongerExists() {
        LaunchOptions opts = new LaunchOptions()
                .headless(true)
                .timeoutMs(120_000);

        try (Vibe vibe = Browser.launch(opts)) {
            vibe.go("https://the-internet.herokuapp.com/inputs");

            Element input = vibe.find("input", 10_000);
            vibe.evaluate("document.querySelector('input')?.remove()");

            assertThrows(VibiumElementNotFoundException.class, input::text);
        }
    }

    @Test
    public void scriptErrorsMapToRemoteError() {
        LaunchOptions opts = new LaunchOptions()
                .headless(true)
                .timeoutMs(120_000);

        try (Vibe vibe = Browser.launch(opts)) {
            vibe.go("https://the-internet.herokuapp.com/");

            VibiumRemoteErrorException e = assertThrows(
                    VibiumRemoteErrorException.class,
                    () -> vibe.evaluate("throw new Error('boom')")
            );
            assertTrue(e.getMessage().toLowerCase().contains("boom"), "expected remote error message to include 'boom': " + e.getMessage());
        }
    }

    @Test
    public void quittingClosesClickerPort() throws Exception {
        LaunchOptions opts = new LaunchOptions()
                .headless(true)
                .timeoutMs(120_000);

        int port;
        try (Vibe vibe = Browser.launch(opts)) {
            port = vibe.port();
            assertTrue(port > 0);
            vibe.go("https://the-internet.herokuapp.com/");
        }

        // Give the process a moment to exit and release the port.
        Thread.sleep(750);

        try {
            new Socket("127.0.0.1", port).close();
            fail("Port should be closed after quit()");
        } catch (Exception expected) {
            // expected
        }
    }

    @Test
    public void cleanupRunsOnException() throws Exception {
        LaunchOptions opts = new LaunchOptions()
                .headless(true)
                .timeoutMs(120_000);

        int port = -1;
        try {
            try (Vibe vibe = Browser.launch(opts)) {
                port = vibe.port();
                assertTrue(port > 0);
                vibe.go("https://the-internet.herokuapp.com/");
                throw new RuntimeException("boom");
            }
        } catch (RuntimeException e) {
            assertEquals("boom", e.getMessage());
        }

        assertTrue(port > 0, "port should have been assigned");

        Thread.sleep(750);

        try {
            new Socket("127.0.0.1", port).close();
            fail("Port should be closed after exception unwinds try-with-resources");
        } catch (Exception expected) {
            // expected
        }
    }

    @Test
    public void asyncApiCanNavigateAndScreenshot() throws Exception {
        LaunchOptions opts = new LaunchOptions()
                .headless(true)
                .timeoutMs(120_000);

        VibeAsync vibe = BrowserAsync.launch(opts).get(120, TimeUnit.SECONDS);
        try {
            vibe.go("https://the-internet.herokuapp.com/").get(60, TimeUnit.SECONDS);
            byte[] screenshot = vibe.screenshot().get(60, TimeUnit.SECONDS);
            assertNotNull(screenshot);
            assertTrue(screenshot.length > 1000, "screenshot too small");
        } finally {
            vibe.quit().get(30, TimeUnit.SECONDS);
        }
    }

    @Test
    public void canSubscribeAndWaitForLoadEvent() throws Exception {
        LaunchOptions opts = new LaunchOptions()
                .headless(true)
                .timeoutMs(120_000);

        try (Vibe vibe = Browser.launch(opts)) {
            vibe.subscribe("browsingContext.load");

            CompletableFuture<?> waiter = vibe.waitForEventAsync("browsingContext.load").orTimeout(30, TimeUnit.SECONDS);
            vibe.go("https://the-internet.herokuapp.com/");
            waiter.get(45, TimeUnit.SECONDS);
        }
    }

    private static boolean isBrowserInstalled(String clicker) throws Exception {
        ProcessBuilder pb = new ProcessBuilder(clicker, "paths");
        pb.redirectErrorStream(true);
        Process p = pb.start();
        List<String> lines = readAllLines(p);
        p.waitFor(30, TimeUnit.SECONDS);

        String chromePath = null;
        String driverPath = null;
        for (String line : lines) {
            if (line.startsWith("Chrome:")) {
                chromePath = line.substring("Chrome:".length()).trim();
            } else if (line.startsWith("Chromedriver:")) {
                driverPath = line.substring("Chromedriver:".length()).trim();
            }
        }

        boolean chromeOk = chromePath != null && !chromePath.isBlank() && !chromePath.equalsIgnoreCase("not found") && Files.isRegularFile(Path.of(chromePath));
        boolean driverOk = driverPath != null && !driverPath.isBlank() && !driverPath.equalsIgnoreCase("not found") && Files.isRegularFile(Path.of(driverPath));
        return chromeOk && driverOk;
    }

    private static List<String> readAllLines(Process p) throws Exception {
        List<String> out = new ArrayList<>();
        try (BufferedReader r = new BufferedReader(new InputStreamReader(p.getInputStream()))) {
            String line;
            while ((line = r.readLine()) != null) {
                out.add(line);
            }
        }
        return out;
    }

    private static String resolveClickerBinaryNoDownload() {
        String env = System.getenv("VIBIUM_CLICKER_PATH");
        if (env != null && !env.isBlank() && Files.isRegularFile(Path.of(env))) {
            return Path.of(env).toAbsolutePath().toString();
        }

        String env2 = System.getenv("CLICKER_PATH");
        if (env2 != null && !env2.isBlank() && Files.isRegularFile(Path.of(env2))) {
            return Path.of(env2).toAbsolutePath().toString();
        }

        Path repoLocal = Path.of("..", "..", "clicker", "bin", isWindows() ? "clicker.exe" : "clicker").normalize();
        if (Files.isRegularFile(repoLocal)) {
            return repoLocal.toAbsolutePath().toString();
        }

        return findOnPath(isWindows() ? "clicker.exe" : "clicker");
    }

    private static String findOnPath(String binary) {
        String path = System.getenv("PATH");
        if (path == null) return null;
        for (String dir : path.split(File.pathSeparator)) {
            if (dir == null || dir.isBlank()) continue;
            Path candidate = Path.of(dir, binary);
            if (Files.isRegularFile(candidate)) {
                return candidate.toAbsolutePath().toString();
            }
        }
        return null;
    }

    private static boolean isWindows() {
        String os = System.getProperty("os.name").toLowerCase();
        return os.contains("win");
    }
}
