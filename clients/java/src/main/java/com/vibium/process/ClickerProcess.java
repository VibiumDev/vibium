package com.vibium.process;

import com.vibium.LaunchOptions;
import com.vibium.VibiumException;
import com.vibium.VibiumTimeoutException;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.BufferedReader;
import java.io.File;
import java.io.IOException;
import java.io.InputStreamReader;
import java.net.ServerSocket;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.ArrayList;
import java.util.List;
import java.util.OptionalInt;
import java.util.concurrent.CompletableFuture;
import java.util.concurrent.TimeUnit;
import java.util.regex.Matcher;
import java.util.regex.Pattern;

public final class ClickerProcess {
    private static final Logger log = LoggerFactory.getLogger(ClickerProcess.class);

    private static final Pattern SERVER_LISTENING =
            Pattern.compile("Server listening on ws://localhost:(\\d+)", Pattern.CASE_INSENSITIVE);

    private static final Pattern ROUTER_SESSION_CLOSED =
            Pattern.compile("\\[router\\]\\s+Browser session closed for client\\s+\\d+", Pattern.CASE_INSENSITIVE);

    private final Process process;
    private final int port;
    private final CompletableFuture<Void> sessionClosed;

    private ClickerProcess(Process process, int port, CompletableFuture<Void> sessionClosed) {
        this.process = process;
        this.port = port;
        this.sessionClosed = sessionClosed;
    }

    public static ClickerProcess start(LaunchOptions options) {
        LaunchOptions resolved = options == null ? new LaunchOptions() : options;
        String clickerPath = resolveClickerBinary(resolved.getClickerPath());
        ensureBrowserInstalled(clickerPath);

        int port = resolved.getPort() != null ? resolved.getPort() : findFreePort();

        List<String> cmd = new ArrayList<>();
        cmd.add(clickerPath);
        cmd.add("serve");
        cmd.add("--port");
        cmd.add(String.valueOf(port));
        if (resolved.isHeadless()) {
            cmd.add("--headless");
        }

        ProcessBuilder pb = new ProcessBuilder(cmd);
        pb.redirectErrorStream(true);

        log.debug("Starting clicker: {}", String.join(" ", cmd));

        Process process = null;
        try {
            process = pb.start();
            final Process proc = process;
            StringBuilder output = new StringBuilder();
            CompletableFuture<Integer> portFuture = new CompletableFuture<>();
            CompletableFuture<Void> sessionClosed = new CompletableFuture<>();

            Thread reader = new Thread(() -> readOutput(proc, output, portFuture, sessionClosed), "vibium-clicker-stdout");
            reader.setDaemon(true);
            reader.start();

            int actualPort = portFuture.get(resolved.getTimeoutMs(), TimeUnit.MILLISECONDS);
            return new ClickerProcess(process, actualPort, sessionClosed);
        } catch (java.util.concurrent.TimeoutException e) {
            if (process != null) {
                stopProcess(process);
            }
            throw new VibiumTimeoutException("clicker serve startup", resolved.getTimeoutMs(), e);
        } catch (Exception e) {
            if (process != null) {
                stopProcess(process);
            }
            throw new VibiumException("Failed to start clicker serve", e);
        }
    }

    public int getPort() {
        return port;
    }

    public boolean isAlive() {
        return process.isAlive();
    }

    public boolean waitForSessionClosed(long timeoutMs) {
        if (timeoutMs <= 0) {
            return false;
        }
        if (sessionClosed == null) {
            return false;
        }
        try {
            sessionClosed.get(timeoutMs, TimeUnit.MILLISECONDS);
            return true;
        } catch (Exception ignored) {
            return false;
        }
    }

    public void stop() {
        stopProcess(process);
    }

    private static void stopProcess(Process process) {
        if (process == null) {
            return;
        }
        if (!process.isAlive()) {
            return;
        }

        if (Platform.isWindows()) {
            // Try terminate first. If Clicker doesn't exit, fall back to killing the whole tree.
            try {
                process.destroy();
                process.waitFor(2, TimeUnit.SECONDS);
            } catch (InterruptedException e) {
                Thread.currentThread().interrupt();
            } catch (Exception ignored) {
            }

            if (process.isAlive()) {
                long pid = process.pid();
                try {
                    Process killer = new ProcessBuilder(
                            "taskkill",
                            "/PID", String.valueOf(pid),
                            "/T",
                            "/F"
                    ).redirectErrorStream(true).start();
                    killer.waitFor(10, TimeUnit.SECONDS);
                } catch (Exception ignored) {
                }

                try {
                    process.waitFor(5, TimeUnit.SECONDS);
                } catch (InterruptedException e) {
                    Thread.currentThread().interrupt();
                } catch (Exception ignored) {
                }

                if (process.isAlive()) {
                    process.destroyForcibly();
                }
            }
            return;
        }

        process.destroy();
        try {
            if (!process.waitFor(5, TimeUnit.SECONDS)) {
                process.destroyForcibly();
            }
        } catch (InterruptedException e) {
            Thread.currentThread().interrupt();
            process.destroyForcibly();
        }
    }

    private static void readOutput(
            Process process,
            StringBuilder output,
            CompletableFuture<Integer> portFuture,
            CompletableFuture<Void> sessionClosed
    ) {
        final int limitChars = 16_384;
        try (BufferedReader reader = new BufferedReader(new InputStreamReader(process.getInputStream()))) {
            String line;
            while ((line = reader.readLine()) != null) {
                if (output.length() < limitChars) {
                    output.append(line).append('\n');
                }
                OptionalInt port = parseReadyPortFromLine(line);
                if (port.isPresent() && !portFuture.isDone()) {
                    portFuture.complete(port.getAsInt());
                }
                if (sessionClosed != null && !sessionClosed.isDone() && ROUTER_SESSION_CLOSED.matcher(line).find()) {
                    sessionClosed.complete(null);
                }
            }
        } catch (IOException ignored) {
        } finally {
            if (!portFuture.isDone()) {
                portFuture.completeExceptionally(
                        new VibiumException("Clicker exited before reporting a port" + (output.isEmpty() ? "" : (":\n" + output)))
                );
            }
            if (sessionClosed != null && !sessionClosed.isDone()) {
                sessionClosed.completeExceptionally(new VibiumException("Clicker output ended before session closed"));
            }
        }
    }

    static OptionalInt parsePortFromLine(String line) {
        if (line == null) {
            return OptionalInt.empty();
        }
        Matcher m1 = SERVER_LISTENING.matcher(line);
        if (m1.find()) {
            return OptionalInt.of(Integer.parseInt(m1.group(1)));
        }
        return OptionalInt.empty();
    }

    private static OptionalInt parseReadyPortFromLine(String line) {
        // Only consider the server "ready" once Clicker explicitly prints the
        // line emitted after it successfully bound the socket.
        return parsePortFromLine(line);
    }

    private static String resolveClickerBinary(String customPath) {
        if (customPath != null && !customPath.trim().isEmpty()) {
            Path p = Path.of(customPath);
            if (Files.isRegularFile(p)) {
                return p.toAbsolutePath().toString();
            }
            throw new VibiumException("Clicker binary not found: " + customPath);
        }

        String env = System.getenv("VIBIUM_CLICKER_PATH");
        if (env != null && !env.trim().isEmpty() && Files.isRegularFile(Path.of(env))) {
            return Path.of(env).toAbsolutePath().toString();
        }

        String env2 = System.getenv("CLICKER_PATH");
        if (env2 != null && !env2.trim().isEmpty() && Files.isRegularFile(Path.of(env2))) {
            return Path.of(env2).toAbsolutePath().toString();
        }

        String fromPath = findOnPath(Platform.binaryName());
        if (fromPath != null) {
            return fromPath;
        }

        Path repoLocal = Path.of("..", "..", "clicker", "bin", Platform.binaryName()).normalize();
        if (Files.isRegularFile(repoLocal)) {
            return repoLocal.toAbsolutePath().toString();
        }

        Path cached = findCachedClicker();
        if (cached != null) {
            return cached.toAbsolutePath().toString();
        }

        if (!"1".equals(System.getenv("VIBIUM_SKIP_CLICKER_DOWNLOAD"))) {
            Path downloaded = ClickerDownloader.downloadToCacheIfNeeded();
            return downloaded.toAbsolutePath().toString();
        }

        throw new VibiumException(
                "Could not find clicker binary. Set VIBIUM_CLICKER_PATH (or CLICKER_PATH), " +
                        "or build Clicker at ../../clicker/bin/" + Platform.binaryName() +
                        ". To enable auto-download, unset VIBIUM_SKIP_CLICKER_DOWNLOAD."
        );
    }

    private static Path findCachedClicker() {
        // Prefer a versioned cache if available for this client version.
        String v = VibiumVersion.currentVersionOrNull();
        if (v != null) {
            Path candidate = VibiumCache.rootDir()
                    .resolve("clicker")
                    .resolve(v)
                    .resolve(Platform.binaryName());
            if (Files.isRegularFile(candidate)) {
                return candidate;
            }
        }

        // Fallback: any cached clicker (useful when running from classfiles and version is unknown).
        Path root = VibiumCache.rootDir().resolve("clicker");
        try {
            if (!Files.isDirectory(root)) {
                return null;
            }
            try (java.util.stream.Stream<Path> paths = Files.walk(root, 3)) {
                return paths
                        .filter(p -> p.getFileName() != null && p.getFileName().toString().equalsIgnoreCase(Platform.binaryName()))
                        .filter(Files::isRegularFile)
                        .findFirst()
                        .orElse(null);
            }
        } catch (Exception ignored) {
            return null;
        }
    }

    private static void ensureBrowserInstalled(String clickerPath) {
        boolean skip = "1".equals(System.getenv("VIBIUM_SKIP_BROWSER_DOWNLOAD"));
        if (isBrowserInstalled(clickerPath)) {
            return;
        }
        if (skip) {
            throw new VibiumException("Chrome for Testing not installed. Run '" + clickerPath + " install' or unset VIBIUM_SKIP_BROWSER_DOWNLOAD.");
        }

        log.info("Chrome for Testing not found; running 'clicker install'...");
        Process p = null;
        try {
            ProcessBuilder pb = new ProcessBuilder(clickerPath, "install");
            pb.redirectErrorStream(true);
            p = pb.start();
            final Process proc = p;

            StringBuilder output = new StringBuilder();
            Thread drainer = new Thread(() -> drainOutput(proc, output, 16_384), "vibium-clicker-install-stdout");
            drainer.setDaemon(true);
            drainer.start();

            boolean ok = p.waitFor(10, TimeUnit.MINUTES);
            if (!ok) {
                stopProcess(p);
                throw new VibiumTimeoutException("clicker install", 10 * 60_000L);
            }
            if (p.exitValue() != 0) {
                throw new VibiumException("clicker install failed with exit code " + p.exitValue() + (output.isEmpty() ? "" : (":\n" + output)));
            }

            if (!isBrowserInstalled(clickerPath)) {
                throw new VibiumException("Chrome for Testing still not found after clicker install");
            }
        } catch (VibiumException e) {
            throw e;
        } catch (Exception e) {
            if (p != null) {
                stopProcess(p);
            }
            throw new VibiumException("Failed to run clicker install", e);
        }
    }

    private static void drainOutput(Process p, StringBuilder out, int limitChars) {
        try (BufferedReader r = new BufferedReader(new InputStreamReader(p.getInputStream()))) {
            String line;
            while ((line = r.readLine()) != null) {
                if (out.length() < limitChars) {
                    out.append(line).append('\n');
                }
            }
        } catch (Exception ignored) {
        }
    }

    private static boolean isBrowserInstalled(String clickerPath) {
        Process p = null;
        try {
            ProcessBuilder pb = new ProcessBuilder(clickerPath, "paths");
            pb.redirectErrorStream(true);
            p = pb.start();

            String chromePath = null;
            String chromedriverPath = null;

            try (BufferedReader r = new BufferedReader(new InputStreamReader(p.getInputStream()))) {
                String line;
                while ((line = r.readLine()) != null) {
                    if (line.startsWith("Chrome:")) {
                        chromePath = line.substring("Chrome:".length()).trim();
                    } else if (line.startsWith("Chromedriver:")) {
                        chromedriverPath = line.substring("Chromedriver:".length()).trim();
                    }
                }
            }

            p.waitFor(30, TimeUnit.SECONDS);

            boolean chromeOk = chromePath != null && !chromePath.isBlank() && !"not found".equalsIgnoreCase(chromePath) && Files.isRegularFile(Path.of(chromePath));
            boolean driverOk = chromedriverPath != null && !chromedriverPath.isBlank() && !"not found".equalsIgnoreCase(chromedriverPath) && Files.isRegularFile(Path.of(chromedriverPath));
            return chromeOk && driverOk;
        } catch (Exception ignored) {
            if (p != null) {
                stopProcess(p);
            }
            return false;
        }
    }

    private static String findOnPath(String binaryName) {
        String path = System.getenv("PATH");
        if (path == null) {
            return null;
        }
        for (String dir : path.split(File.pathSeparator)) {
            if (dir == null || dir.isEmpty()) {
                continue;
            }
            Path candidate = Path.of(dir, binaryName);
            if (Files.isRegularFile(candidate)) {
                return candidate.toAbsolutePath().toString();
            }
        }
        return null;
    }

    private static int findFreePort() {
        try (ServerSocket socket = new ServerSocket(0)) {
            socket.setReuseAddress(true);
            return socket.getLocalPort();
        } catch (IOException e) {
            // Fall back to Clicker's default if we can't allocate a port.
            return 9515;
        }
    }
}
