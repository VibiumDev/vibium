package com.vibium.internal;

import java.io.File;
import java.io.InputStream;
import java.net.URL;
import java.net.URLClassLoader;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.LinkOption;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.util.ArrayList;
import java.util.List;
import java.util.concurrent.CompletableFuture;
import java.util.concurrent.CountDownLatch;
import java.util.concurrent.TimeUnit;

/**
 * Child JVM for {@link BinaryResolverExtractionTest}: resolves the binary the way
 * a consumer does — through the public API, with VIBIUM_BIN_PATH unset and PATH
 * cleared so resolution reaches the extraction branch — then proves the result is
 * complete and runnable, and prints one verdict line.
 *
 * <p>Touches nothing but the public API, so it also runs against the resolver as
 * it was before the fix.
 */
public final class ExtractProbe {

    static final String BARRIER = "vibium.test.barrier";
    static final String CACHE_DIR = "vibium.test.cacheDir";
    static final String SIZE = "vibium.test.size";
    static final String LOADERS = "vibium.test.classLoaders";
    static final String MARKER = "vibium-extract-probe-ok";

    private ExtractProbe() {}

    public static void main(String[] args) {
        String label = args[0];
        try {
            awaitBarrier(label);
            if (Boolean.getBoolean(LOADERS)) {
                raceClassLoaders(label);
                return;
            }
            String path;
            try {
                path = BinaryResolver.resolve();
            } catch (Throwable t) {
                // Refusing a cache it cannot safely repair is a legitimate outcome,
                // and a different one from handing out a bad path.
                print(label, "REFUSED " + t.getMessage());
                return;
            }
            verify(path);
            print(label, "OK " + path);
        } catch (Throwable t) {
            print(label, "FAIL " + t);
        }
    }

    /**
     * Resolve from two class loaders at once. Each gets its own copy of the
     * resolver, so its own intra-JVM monitor, and both reach the file lock — where
     * {@code tryLock} throws rather than returning null.
     */
    private static void raceClassLoaders(String label) throws Exception {
        String[] entries = System.getProperty("java.class.path").split(File.pathSeparator);
        URL[] classpath = new URL[entries.length];
        for (int i = 0; i < entries.length; i++) {
            classpath[i] = Paths.get(entries[i]).toUri().toURL();
        }

        CountDownLatch go = new CountDownLatch(1);
        List<CompletableFuture<Object>> results = new ArrayList<>();
        for (int i = 0; i < 2; i++) {
            CompletableFuture<Object> done = new CompletableFuture<>();
            results.add(done);
            new Thread(() -> {
                try (URLClassLoader loader = new URLClassLoader(classpath, null)) {
                    Class<?> resolver = loader.loadClass(BinaryResolver.class.getName());
                    go.await();
                    done.complete(resolver.getMethod("resolve").invoke(null));
                } catch (Throwable t) {
                    done.complete(t.getCause() != null ? t.getCause() : t);
                }
            }).start();
        }
        go.countDown();

        for (CompletableFuture<Object> result : results) {
            Object value = result.get(120, TimeUnit.SECONDS);
            if (value instanceof Throwable) {
                throw new IllegalStateException("resolve threw", (Throwable) value);
            }
            verify((String) value);
        }
        print(label, "OK two class loaders");
    }

    /** Park until every sibling is up, so the forks contend rather than queue. */
    private static void awaitBarrier(String label) throws Exception {
        String dir = System.getProperty(BARRIER);
        if (dir == null) {
            return;
        }
        Path barrier = Paths.get(dir);
        Files.createFile(barrier.resolve("ready-" + label));
        Path go = barrier.resolve("go");
        long deadline = System.nanoTime() + TimeUnit.SECONDS.toNanos(120);
        while (!Files.exists(go)) {
            if (System.nanoTime() - deadline > 0) {
                throw new IllegalStateException("barrier never opened");
            }
            // Yield, not spin: on a runner with fewer cores than forks, whoever
            // arrives first must not starve the ones still starting up.
            Thread.yield();
        }
    }

    private static void verify(String path) throws Exception {
        Path binary = Paths.get(path).toAbsolutePath().normalize();
        Path cacheDir = Paths.get(System.getProperty(CACHE_DIR)).toAbsolutePath().normalize();
        if (!binary.startsWith(cacheDir)) {
            throw new IllegalStateException("resolved outside the cache: " + binary);
        }
        if (!Files.isRegularFile(binary, LinkOption.NOFOLLOW_LINKS)) {
            throw new IllegalStateException("not a regular file: " + binary);
        }
        long expected = Long.parseLong(System.getProperty(SIZE));
        if (Files.size(binary) != expected) {
            throw new IllegalStateException("truncated: " + Files.size(binary) + " of " + expected);
        }
        if (!Files.isExecutable(binary)) {
            throw new IllegalStateException("not executable: " + binary);
        }

        // Run it, which is what a caller does next. The payload only prints its
        // marker after the whole file has been read, so this catches a short copy
        // as well as a missing exec bit.
        Process process = new ProcessBuilder(binary.toString()).redirectErrorStream(true).start();
        String output;
        try (InputStream in = process.getInputStream()) {
            output = new String(in.readAllBytes(), StandardCharsets.UTF_8);
        }
        if (process.waitFor() != 0 || !output.contains(MARKER)) {
            throw new IllegalStateException("exec failed: " + output.trim());
        }
    }

    private static void print(String label, String verdict) {
        System.out.println("RESULT " + label + " " + verdict.replaceAll("\\s+", " ").trim());
        System.out.flush();
    }
}
