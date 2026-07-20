package com.vibium.internal;

import com.vibium.errors.BrowserCrashedException;
import com.vibium.errors.VibiumConnectionException;

import java.io.*;
import java.util.ArrayList;
import java.util.List;
import java.util.Map;
import java.util.concurrent.ExecutionException;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;
import java.util.concurrent.Future;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.TimeoutException;

/**
 * Manages the vibium subprocess lifecycle.
 *
 * Spawns {@code vibium pipe [--headless]} and waits for the ready signal.
 */
public class VibiumProcess {

    /** Bound on how long to wait for the ready signal per launch attempt. */
    private static final long READY_TIMEOUT_SECONDS = 60;
    /** Number of launch attempts before giving up (retries transient failures). */
    private static final int MAX_START_ATTEMPTS = 2;
    /** Backoff between launch attempts. */
    private static final long START_RETRY_BACKOFF_MS = 500;

    private final Process process;
    private final BufferedWriter stdin;
    private final BufferedReader stdout;
    private final List<String> preReadyLines;

    private VibiumProcess(Process process, BufferedWriter stdin, BufferedReader stdout, List<String> preReadyLines) {
        this.process = process;
        this.stdin = stdin;
        this.stdout = stdout;
        this.preReadyLines = preReadyLines;
    }

    /**
     * Start a vibium pipe subprocess.
     */
    public static VibiumProcess start(String binaryPath, boolean headless, String connectURL, Map<String, String> connectHeaders) {
        List<String> cmd = new ArrayList<>();
        cmd.add(binaryPath);
        cmd.add("pipe");

        if (headless) {
            cmd.add("--headless");
        }

        if (connectURL != null && !connectURL.isEmpty()) {
            cmd.add("--connect");
            cmd.add(connectURL);
        }

        if (connectHeaders != null) {
            for (Map.Entry<String, String> entry : connectHeaders.entrySet()) {
                cmd.add("--connect-header");
                cmd.add(entry.getKey() + "=" + entry.getValue());
            }
        }

        // Auto-install Chrome if needed (skip for remote connections)
        if (connectURL == null || connectURL.isEmpty()) {
            BrowserInstaller.ensureInstalled(binaryPath);
        }

        // Startup is slow (~16s cold) and slower when many browsers launch at
        // once (test suites, CI), where a cold launch can blow the ready timeout
        // or crash under resource pressure. Retry a timed-out or crashed launch a
        // couple of times with a short backoff so a single unlucky launch doesn't
        // fail hard.
        VibiumConnectionException lastError = null;
        for (int attempt = 1; attempt <= MAX_START_ATTEMPTS; attempt++) {
            try {
                return startAttempt(cmd);
            } catch (VibiumConnectionException e) {
                lastError = e;
                if (attempt < MAX_START_ATTEMPTS) {
                    try {
                        Thread.sleep(START_RETRY_BACKOFF_MS);
                    } catch (InterruptedException ie) {
                        Thread.currentThread().interrupt();
                        throw e;
                    }
                    continue;
                }
                throw e;
            }
        }
        // Unreachable: the loop returns on success or throws on the final attempt.
        throw lastError;
    }

    private static VibiumProcess startAttempt(List<String> cmd) {
        try {
            ProcessBuilder pb = new ProcessBuilder(cmd);
            pb.redirectErrorStream(false);
            Process process = pb.start();

            BufferedWriter stdin = new BufferedWriter(new OutputStreamWriter(process.getOutputStream(), "UTF-8"));
            BufferedReader stdout = new BufferedReader(new InputStreamReader(process.getInputStream(), "UTF-8"));

            // Read lines until we get the ready signal, bounded by a timeout so a
            // hung launch (process alive but never emitting ready) can't block
            // forever — historically this readLine() loop could hang the suite.
            List<String> preReadyLines = new ArrayList<>();
            boolean ready;
            ExecutorService readerPool = Executors.newSingleThreadExecutor(r -> {
                Thread t = new Thread(r, "vibium-ready-wait");
                t.setDaemon(true);
                return t;
            });
            try {
                Future<Boolean> readerTask = readerPool.submit(() -> {
                    String line;
                    while ((line = stdout.readLine()) != null) {
                        if (line.contains("vibium:lifecycle.ready")) {
                            return true;
                        }
                        preReadyLines.add(line);
                    }
                    return false; // EOF before ready — process exited
                });
                try {
                    ready = readerTask.get(READY_TIMEOUT_SECONDS, TimeUnit.SECONDS);
                } catch (TimeoutException te) {
                    readerTask.cancel(true);
                    process.destroyForcibly();
                    throw new VibiumConnectionException(
                        "Vibium failed to start: timed out waiting for ready signal");
                } catch (ExecutionException ee) {
                    process.destroyForcibly();
                    Throwable cause = ee.getCause() != null ? ee.getCause() : ee;
                    throw new VibiumConnectionException(
                        "Vibium failed to start: " + cause.getMessage(), cause);
                } catch (InterruptedException ie) {
                    Thread.currentThread().interrupt();
                    process.destroyForcibly();
                    throw new VibiumConnectionException("Interrupted while starting vibium", ie);
                }
            } finally {
                readerPool.shutdownNow();
            }

            if (!ready) {
                // Process exited (EOF) before emitting ready.
                String stderr = readStream(process.getErrorStream());
                int exitCode = -1;
                try {
                    if (process.waitFor(5, TimeUnit.SECONDS)) {
                        exitCode = process.exitValue();
                    }
                } catch (InterruptedException ignored) {
                    Thread.currentThread().interrupt();
                }
                process.destroyForcibly();
                throw new VibiumConnectionException(
                    rewriteError(stderr, exitCode)
                );
            }

            VibiumProcess vp = new VibiumProcess(process, stdin, stdout, preReadyLines);

            // Register shutdown hook for cleanup
            Runtime.getRuntime().addShutdownHook(new Thread(() -> {
                try {
                    vp.stop();
                } catch (Exception ignored) {}
            }));

            return vp;
        } catch (VibiumConnectionException e) {
            throw e;
        } catch (IOException e) {
            throw new VibiumConnectionException("Failed to start vibium process: " + e.getMessage(), e);
        }
    }

    public Process getProcess() { return process; }
    public BufferedWriter getStdin() { return stdin; }
    public BufferedReader getStdout() { return stdout; }
    public List<String> getPreReadyLines() { return preReadyLines; }

    /**
     * Stop the vibium process gracefully.
     */
    public void stop() {
        if (!process.isAlive()) return;

        try {
            // Try to close stdin to signal the process
            try {
                stdin.close();
            } catch (IOException ignored) {}

            // Wait for graceful exit
            if (!process.waitFor(3, TimeUnit.SECONDS)) {
                process.destroy();
                if (!process.waitFor(2, TimeUnit.SECONDS)) {
                    process.destroyForcibly();
                }
            }
        } catch (InterruptedException e) {
            process.destroyForcibly();
            Thread.currentThread().interrupt();
        }
    }

    /**
     * Check if the process is still running.
     */
    public boolean isAlive() {
        return process.isAlive();
    }

    /**
     * Rewrite Go-level error messages into Java-specific guidance.
     */
    private static String rewriteError(String stderr, int exitCode) {
        String lower = stderr.toLowerCase();

        if (lower.contains("chromedriver not found") || lower.contains("chrome not found")) {
            return "Chrome for Testing is not installed.\n\n" +
                "The automatic download did not succeed. To install manually, run:\n\n" +
                "  java -jar vibium.jar install\n\n" +
                "Or, if vibium is on your PATH:\n\n" +
                "  vibium install\n\n" +
                "To skip automatic downloads, set VIBIUM_SKIP_BROWSER_DOWNLOAD=1.\n" +
                (stderr.isEmpty() ? "" : "\nOriginal error: " + stderr);
        }

        // Default: pass through the original message
        return "vibium process did not send ready signal (exit code: " + exitCode + ")" +
            (stderr.isEmpty() ? "" : "\nstderr: " + stderr);
    }

    private static String readStream(InputStream is) {
        try {
            byte[] buf = new byte[4096];
            int avail = is.available();
            if (avail <= 0) return "";
            int len = is.read(buf, 0, Math.min(avail, buf.length));
            if (len <= 0) return "";
            return new String(buf, 0, len, "UTF-8");
        } catch (IOException e) {
            return "";
        }
    }
}
