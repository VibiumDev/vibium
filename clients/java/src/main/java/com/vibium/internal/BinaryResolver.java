package com.vibium.internal;

import com.vibium.errors.VibiumNotFoundException;

import java.io.IOException;
import java.io.InputStream;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.nio.file.StandardCopyOption;
import java.util.concurrent.TimeUnit;

/**
 * Finds or extracts the vibium binary.
 *
 * Resolution order:
 * 1. VIBIUM_BIN_PATH environment variable
 * 2. PATH lookup
 * 3. Extract from JAR resources
 */
public final class BinaryResolver {

    private BinaryResolver() {}

    /**
     * Resolve the path to the vibium binary.
     */
    public static String resolve() {
        // 1. Environment variable
        String envPath = System.getenv("VIBIUM_BIN_PATH");
        if (envPath != null && !envPath.isEmpty()) {
            Path p = Paths.get(envPath);
            if (Files.isExecutable(p)) {
                return p.toAbsolutePath().toString();
            }
        }

        // 2. PATH lookup
        String pathResult = findOnPath();
        if (pathResult != null) {
            warnIfShadowingJar(pathResult);
            return pathResult;
        }

        // 3. Extract from JAR
        String extracted = extractFromJar();
        if (extracted != null) {
            return extracted;
        }

        throw new VibiumNotFoundException(
            "vibium binary not found. Install it via npm (npm install vibium), " +
            "set VIBIUM_BIN_PATH, or ensure it's on your PATH."
        );
    }

    private static volatile boolean shadowChecked = false;

    /**
     * A PATH install wins over the jar's packaged binary by the documented
     * resolution order, but the jar version is the only thing a Maven build
     * controls, and a version mismatch surfaces as protocol errors that do
     * not point at versions at all. When the versions differ, say which
     * binary actually runs (#331). Checked once per JVM.
     */
    private static void warnIfShadowingJar(String pathBinary) {
        if (shadowChecked) {
            return;
        }
        shadowChecked = true;

        String jarVersion = readVersion();
        boolean jarHasBinary = BinaryResolver.class.getClassLoader()
            .getResource("natives/" + PlatformDetector.binaryName()) != null;
        if (!jarHasBinary || "unknown".equals(jarVersion)) {
            return;
        }
        String warning = shadowWarning(pathBinary, binaryVersion(pathBinary), jarVersion);
        if (warning != null) {
            System.err.println(warning);
        }
    }

    /**
     * The warning for a PATH binary shadowing the jar's packaged one, or null
     * when the versions match or the PATH binary's version could not be read.
     * Visible for testing.
     */
    static String shadowWarning(String pathBinary, String pathVersion, String jarVersion) {
        if (pathVersion == null || pathVersion.equals(jarVersion)) {
            return null;
        }
        return "[vibium] Using vibium " + pathVersion + " from PATH (" + pathBinary
            + "), not the " + jarVersion + " binary packaged in this jar."
            + " Set VIBIUM_BIN_PATH to pick one explicitly.";
    }

    /** Version reported by {@code binary --version}, or null when it cannot be read. */
    private static String binaryVersion(String binary) {
        try {
            Process p = new ProcessBuilder(binary, "--version")
                .redirectErrorStream(true)
                .start();
            if (!p.waitFor(5, TimeUnit.SECONDS)) {
                p.destroyForcibly();
                return null;
            }
            return parseVersion(new String(readAllBytes(p.getInputStream())).trim());
        } catch (IOException e) {
            return null;
        } catch (InterruptedException e) {
            Thread.currentThread().interrupt();
            return null;
        }
    }

    /**
     * Extracts the version number from {@code --version} output such as
     * "vibium v26.8.21". Visible for testing.
     */
    static String parseVersion(String output) {
        if (output == null) {
            return null;
        }
        String firstLine = output.split("\\R", 2)[0];
        String version = firstLine.replaceAll("^[^0-9]*", "");
        return version.isEmpty() ? null : version;
    }

    private static String findOnPath() {
        String execName = PlatformDetector.executableName();
        String pathEnv = System.getenv("PATH");
        if (pathEnv == null) return null;

        for (String dir : pathEnv.split(System.getProperty("path.separator"))) {
            Path candidate = Paths.get(dir, execName);
            if (Files.isExecutable(candidate)) {
                return candidate.toAbsolutePath().toString();
            }
        }
        return null;
    }

    private static String extractFromJar() {
        String resourceName = "natives/" + PlatformDetector.binaryName();
        InputStream stream = BinaryResolver.class.getClassLoader().getResourceAsStream(resourceName);
        if (stream == null) {
            return null;
        }

        try (InputStream in = stream) {
            // Read version for cache directory
            String version = readVersion();
            Path extractDir = Paths.get(System.getProperty("java.io.tmpdir"), "vibium-" + version);
            Files.createDirectories(extractDir);

            Path target = extractDir.resolve(PlatformDetector.executableName());
            Path extracted = extractTo(in, extractDir, target, isWindows());
            return extracted.toAbsolutePath().toString();
        } catch (IOException e) {
            return null;
        }
    }

    /**
     * Extracts the binary so the target path only ever names a complete,
     * executable file: the bytes go to a private temp file, the executable
     * bit is set there, and an atomic rename publishes it. The old code
     * guarded on Files.exists(target), which is true from the first byte of
     * another JVM's in-progress copy, so parallel JVMs on a cold cache were
     * handed a half-written, not-yet-executable binary (#329).
     *
     * Visible for testing.
     */
    static Path extractTo(InputStream in, Path extractDir, Path target, boolean windows) throws IOException {
        if (isUsable(target, windows)) {
            return target;
        }

        Path tmp = Files.createTempFile(extractDir, ".extract-", ".tmp");
        try {
            Files.copy(in, tmp, StandardCopyOption.REPLACE_EXISTING);
            if (!windows) {
                tmp.toFile().setExecutable(true);
            }
            // On POSIX the rename atomically replaces a stale or concurrent
            // target. On Windows it can fail if another JVM's binary is in
            // place or running; theirs is complete, so use it.
            Files.move(tmp, target, StandardCopyOption.ATOMIC_MOVE);
        } catch (IOException moveFailed) {
            Files.deleteIfExists(tmp);
            if (isUsable(target, windows)) {
                return target;
            }
            throw moveFailed;
        }
        return target;
    }

    // A completed extraction is executable; existence alone can be a
    // half-written file from a concurrent or crashed extraction.
    private static boolean isUsable(Path target, boolean windows) {
        return Files.exists(target) && (windows || Files.isExecutable(target));
    }

    private static boolean isWindows() {
        return System.getProperty("os.name", "").toLowerCase().contains("windows");
    }

    private static String readVersion() {
        try (InputStream is = BinaryResolver.class.getClassLoader().getResourceAsStream("vibium-version.txt")) {
            if (is != null) {
                return new String(readAllBytes(is)).trim();
            }
        } catch (IOException ignored) {}
        return "unknown";
    }

    private static byte[] readAllBytes(InputStream is) throws IOException {
        byte[] buf = new byte[1024];
        int len;
        java.io.ByteArrayOutputStream out = new java.io.ByteArrayOutputStream();
        while ((len = is.read(buf)) != -1) {
            out.write(buf, 0, len);
        }
        return out.toByteArray();
    }
}
