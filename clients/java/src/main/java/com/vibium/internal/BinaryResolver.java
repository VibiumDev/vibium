package com.vibium.internal;

import com.vibium.errors.VibiumNotFoundException;

import java.io.IOException;
import java.io.InputStream;
import java.net.URL;
import java.net.URLConnection;
import java.nio.channels.FileChannel;
import java.nio.channels.FileLock;
import java.nio.channels.OverlappingFileLockException;
import java.nio.file.AtomicMoveNotSupportedException;
import java.nio.file.DirectoryNotEmptyException;
import java.nio.file.Files;
import java.nio.file.LinkOption;
import java.nio.file.NoSuchFileException;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.nio.file.StandardCopyOption;
import java.nio.file.StandardOpenOption;
import java.nio.file.attribute.BasicFileAttributes;
import java.nio.file.attribute.PosixFilePermissions;
import java.util.concurrent.TimeUnit;

/**
 * Finds or extracts the vibium binary.
 *
 * Resolution order:
 * 1. VIBIUM_BIN_PATH environment variable
 * 2. PATH lookup
 * 3. Extract from JAR resources
 *
 * <p>Extraction has to be safe for several JVMs sharing a cold
 * {@code java.io.tmpdir}, the normal state of a CI agent running a parallel test
 * runner. A lock file serialises publication across processes, and the binary is
 * copied to a staging file, checked, marked executable and only then renamed into
 * place. So a reader sees either no file or a complete, executable one, and an
 * artifact that passed validation is only ever replaced by an atomic rename —
 * never unlinked from under a caller about to exec it.
 */
public final class BinaryResolver {

    private static final String LOCK_FILE = ".vibium-extract.lock";
    private static final long LOCK_TIMEOUT_NANOS = TimeUnit.SECONDS.toNanos(60);

    /** Serialises extraction within this JVM; a FileLock is per-JVM exclusive. */
    private static final Object EXTRACT_LOCK = new Object();

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
        URL resource = BinaryResolver.class.getClassLoader().getResource(resourceName);
        if (resource == null) {
            return null;
        }

        // Read version for cache directory
        String version = readVersion();
        Path extractDir = Paths.get(System.getProperty("java.io.tmpdir"), "vibium-" + version);
        Path target = extractDir.resolve(PlatformDetector.executableName());

        try {
            long expectedSize = resourceSize(resource);
            // A complete artifact from an earlier run needs no lock.
            if (!isUsable(target, expectedSize)) {
                extract(target, expectedSize, resource);
            }
            return target.toAbsolutePath().toString();
        } catch (IOException e) {
            // The binary is bundled but could not be published, so say why rather
            // than telling the user to install what is already in the JAR.
            throw new VibiumNotFoundException(
                "could not extract the bundled vibium binary to " + target + ": " + e.getMessage());
        }
    }

    /**
     * Extract under a lock spanning processes, so exactly one publisher inspects
     * and replaces the cache path at a time. Without it two processes can both see
     * the same stale entry, and the second can delete the artifact the first has
     * already handed to a caller.
     */
    private static void extract(Path target, long expectedSize, URL resource) throws IOException {
        Path dir = target.getParent();
        Files.createDirectories(dir);

        synchronized (EXTRACT_LOCK) {
            try (FileChannel channel = FileChannel.open(dir.resolve(LOCK_FILE),
                     StandardOpenOption.CREATE, StandardOpenOption.WRITE);
                 FileLock lock = acquire(channel)) {
                // Re-check under the lock: someone may have published while we
                // waited, and the rest of this assumes nobody else interferes.
                if (!isUsable(target, expectedSize)) {
                    publish(target, expectedSize, resource, dir);
                }
            }
        }
    }

    /**
     * {@code tryLock} rather than a blocking lock, so a peer that hangs holding it
     * fails this extraction with a clear message instead of stalling every JVM on
     * the box. A peer that is killed needs no timeout — the OS drops its lock.
     */
    private static FileLock acquire(FileChannel channel) throws IOException {
        long deadline = System.nanoTime() + LOCK_TIMEOUT_NANOS;
        while (true) {
            FileLock lock = tryLock(channel);
            if (lock != null) {
                return lock;
            }
            if (System.nanoTime() - deadline > 0) {
                throw new IOException("timed out waiting for another process to extract");
            }
            try {
                Thread.sleep(5);
            } catch (InterruptedException e) {
                Thread.currentThread().interrupt();
                throw new IOException("interrupted waiting to extract", e);
            }
        }
    }

    /**
     * Null when the lock is held rather than acquired. File locks are tracked per
     * JVM, so a second holder inside this one — another class loader with its own
     * copy of this class, and so its own monitor — makes {@code tryLock} throw
     * instead of returning null. That is still just "busy": retry.
     */
    private static FileLock tryLock(FileChannel channel) throws IOException {
        try {
            return channel.tryLock();
        } catch (OverlappingFileLockException e) {
            return null;
        }
    }

    /** Copy to a staging file and swap it in. Called with the lock held. */
    private static void publish(Path target, long expectedSize, URL resource, Path dir)
            throws IOException {
        // Staged in the cache directory so the rename below stays on one
        // filesystem, and under a unique name so a file left behind by a killed
        // JVM is inert rather than mistaken for the binary.
        Path staged = Files.createTempFile(dir, ".vibium-", ".part");
        try {
            long written;
            try (InputStream in = resource.openStream()) {
                written = Files.copy(in, staged, StandardCopyOption.REPLACE_EXISTING);
            }
            if (written == 0 || (expectedSize >= 0 && written != expectedSize)) {
                throw new IOException("incomplete copy of the bundled binary: " + written
                    + " of " + expectedSize + " bytes");
            }
            sync(staged);
            // Executable before it is visible, so it runs the moment it appears.
            makeExecutable(staged);

            // A rename cannot replace a directory, and a symlink is not something
            // to exec through. Clearing one is safe under the lock: whatever is
            // there failed validation, so no caller was given it.
            if (Files.exists(target, LinkOption.NOFOLLOW_LINKS)
                    && !Files.isRegularFile(target, LinkOption.NOFOLLOW_LINKS)) {
                try {
                    Files.delete(target);
                } catch (NoSuchFileException ignored) {
                    // Gone already.
                } catch (DirectoryNotEmptyException e) {
                    throw new IOException(target + " is a non-empty directory, not a vibium "
                        + "binary; remove it or set VIBIUM_BIN_PATH", e);
                }
            }
            try {
                Files.move(staged, target,
                    StandardCopyOption.REPLACE_EXISTING, StandardCopyOption.ATOMIC_MOVE);
            } catch (AtomicMoveNotSupportedException e) {
                // Fail closed: a plain move may copy straight onto the live path,
                // leaving a half-written binary a caller cannot recognise.
                // Unreachable in practice — staging happens in the target's own
                // directory, so this rename never crosses a filesystem.
                throw new IOException("cannot publish " + target + " atomically; "
                    + "set VIBIUM_BIN_PATH instead", e);
            }
        } finally {
            // Ours alone: a successful rename consumed it, and no process may
            // remove another's, which could still be in flight.
            try {
                Files.deleteIfExists(staged);
            } catch (IOException ignored) {
                // Inert either way — never published, never returned.
            }
        }
    }

    /**
     * True when {@code target} is a complete, runnable artifact. Existing is not
     * enough: an extraction cut short by an older client left a truncated file,
     * and the race this replaces handed out files whose exec bit was still coming.
     */
    private static boolean isUsable(Path target, long expectedSize) {
        try {
            BasicFileAttributes attrs =
                Files.readAttributes(target, BasicFileAttributes.class, LinkOption.NOFOLLOW_LINKS);
            if (!attrs.isRegularFile() || attrs.size() == 0) {
                return false;
            }
            if (expectedSize >= 0 && attrs.size() != expectedSize) {
                return false;
            }
            // Windows has no exec bit, and extraction never set one there.
            return isWindows() || Files.isExecutable(target);
        } catch (IOException e) {
            return false;
        }
    }

    private static void sync(Path file) {
        try (FileChannel channel = FileChannel.open(file, StandardOpenOption.WRITE)) {
            // So a crash cannot publish bytes that never reached the disk.
            channel.force(true);
        } catch (IOException ignored) {
            // Durability only; the copy is coherent to other processes regardless.
        }
    }

    private static void makeExecutable(Path file) throws IOException {
        // Set executable permission on Unix
        if (isWindows()) {
            return;
        }
        try {
            // Runnable by everyone, like an installed binary: the cache sits in a
            // shared temp dir, so a second user on the box can reuse it.
            Files.setPosixFilePermissions(file, PosixFilePermissions.fromString("rwxr-xr-x"));
        } catch (UnsupportedOperationException e) {
            file.toFile().setExecutable(true, false);  // not a POSIX filesystem
        }
        if (!Files.isExecutable(file)) {
            throw new IOException("staged vibium binary is not executable: " + file);
        }
    }

    private static boolean isWindows() {
        return System.getProperty("os.name", "").toLowerCase().contains("windows");
    }

    /** Resource length without reading it, or -1 when unknown. */
    private static long resourceSize(URL resource) {
        try {
            URLConnection conn = resource.openConnection();
            conn.setUseCaches(true);
            try (InputStream ignored = conn.getInputStream()) {
                return conn.getContentLengthLong();
            }
        } catch (IOException e) {
            return -1;
        }
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
