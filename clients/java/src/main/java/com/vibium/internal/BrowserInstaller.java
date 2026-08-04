package com.vibium.internal;

import com.vibium.errors.VibiumConnectionException;

import java.util.ArrayList;
import java.util.List;
import java.util.concurrent.TimeUnit;

/**
 * Ensures the selected browser is installed before launching it.
 *
 * Mirrors the Python client's {@code ensure_browser_installed()} behaviour:
 * runs {@code vibium is-installed} for the selected engine, and if missing
 * runs {@code vibium install} to download it automatically.
 */
public final class BrowserInstaller {

    private BrowserInstaller() {}

    /**
     * Ensure Chrome for Testing is available on this machine.
     *
     * @param binaryPath path to the vibium binary
     * @throws VibiumConnectionException if installation fails
     */
    public static void ensureInstalled(String binaryPath) {
        ensureInstalled(binaryPath, null, null);
    }

    /** Ensure the selected browser engine and channel are available. */
    public static void ensureInstalled(String binaryPath, String engine, String channel) {
        // Respect VIBIUM_SKIP_BROWSER_DOWNLOAD=1
        String skip = System.getenv("VIBIUM_SKIP_BROWSER_DOWNLOAD");
        if ("1".equals(skip) || "true".equalsIgnoreCase(skip)) {
            return;
        }

        List<String> engineArgs = engineArgs(engine, channel);
        String browserName = engine == null || engine.isEmpty() ? "Chrome for Testing" : engine;

        if (isInstalled(binaryPath, engineArgs)) {
            return;
        }

        System.out.println("Downloading " + browserName + "...");
        System.out.flush();

        try {
            List<String> command = new ArrayList<>();
            command.add(binaryPath);
            command.add("install");
            command.addAll(engineArgs);
            ProcessBuilder pb = new ProcessBuilder(command);
            pb.inheritIO();
            Process process = pb.start();

            boolean finished = process.waitFor(5, TimeUnit.MINUTES);
            if (!finished) {
                process.destroyForcibly();
                throw new VibiumConnectionException(browserName + " installation timed out");
            }

            int exitCode = process.exitValue();
            if (exitCode != 0) {
                throw new VibiumConnectionException(
                    "Failed to install " + browserName + " (exit code " + exitCode + ")"
                );
            }

            System.out.println(browserName + " installed successfully.");
            System.out.flush();
        } catch (VibiumConnectionException e) {
            throw e;
        } catch (Exception e) {
            throw new VibiumConnectionException("Failed to install " + browserName + ": " + e.getMessage(), e);
        }
    }

    private static List<String> engineArgs(String engine, String channel) {
        List<String> args = new ArrayList<>();
        if (engine != null && !engine.isEmpty()) {
            args.add("--engine");
            args.add(engine);
        }
        if (channel != null && !channel.isEmpty()) {
            args.add("--firefox-channel");
            args.add(channel);
        }
        return args;
    }

    private static boolean isInstalled(String binaryPath, List<String> engineArgs) {
        try {
            List<String> command = new ArrayList<>();
            command.add(binaryPath);
            command.add("is-installed");
            command.addAll(engineArgs);
            ProcessBuilder pb = new ProcessBuilder(command);
            pb.redirectErrorStream(true);
            Process process = pb.start();

            boolean finished = process.waitFor(10, TimeUnit.SECONDS);
            if (!finished) {
                process.destroyForcibly();
                return false;
            }

            return process.exitValue() == 0;
        } catch (Exception e) {
            return false;
        }
    }
}
