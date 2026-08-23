package com.vibium.internal;

import static org.junit.jupiter.api.Assertions.assertArrayEquals;
import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertTrue;
import static org.junit.jupiter.api.Assumptions.assumeFalse;
import static java.util.stream.Collectors.toList;

import java.io.File;
import java.io.IOException;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.LinkOption;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.nio.file.attribute.PosixFilePermissions;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.LinkedHashSet;
import java.util.List;
import java.util.Set;
import java.util.concurrent.TimeUnit;
import java.util.jar.JarEntry;
import java.util.jar.JarOutputStream;
import java.util.stream.Stream;

import org.junit.jupiter.api.BeforeAll;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.io.TempDir;
import org.junit.jupiter.params.ParameterizedTest;
import org.junit.jupiter.params.provider.ValueSource;

/**
 * Extraction under concurrency (#329).
 *
 * <p>Driven through real child JVMs: the failure is cross-process — the cache
 * path became visible from the first byte of the copy and only gained its exec
 * bit afterwards — and the resolver's inputs (VIBIUM_BIN_PATH, PATH,
 * java.io.tmpdir) are per-process, so a child is the only place to control them.
 * Each fork validates the moment it resolves, because a check done afterwards
 * would see the winner's finished artifact and miss the bug entirely.
 *
 * <p>A synthetic payload stands in for the embedded binary, so these run on a
 * checkout where {@code clicker/bin} was never built.
 */
class BinaryResolverExtractionTest {

    private static final int FORKS = 6;
    private static final int PAYLOAD_BYTES = 8 * 1024 * 1024;
    private static final String VERSION = "329-probe";
    private static final String FOREIGN_PART = "another-process.part";

    @TempDir
    static Path work;

    private static byte[] payload;
    private static Path jar;

    @BeforeAll
    static void buildFixture() throws Exception {
        // The payload standing in for the embedded binary is a POSIX shell script,
        // so a fork cannot exec it when the resolver publishes it as vibium.exe.
        // Skipping keeps the fixture from failing for reasons that have nothing to
        // do with the resolver; publication on Windows is not covered here.
        assumeFalse(isWindows(), "fixture payload is a POSIX shell script");

        payload = payload(PAYLOAD_BYTES);
        jar = work.resolve("natives-probe.jar");
        try (JarOutputStream out = new JarOutputStream(Files.newOutputStream(jar))) {
            out.putNextEntry(new JarEntry("natives/" + PlatformDetector.binaryName()));
            out.write(payload);
            out.closeEntry();
            out.putNextEntry(new JarEntry("vibium-version.txt"));
            out.write(VERSION.getBytes(StandardCharsets.US_ASCII));
            out.closeEntry();
        }
    }

    /**
     * The reported bug, plus the ownership invariant behind the fix: several JVMs
     * start together on a cold cache, each doing what one Surefire fork does.
     *
     * <p>Staging-file count is what proves cross-process ownership. A staging file
     * lives for the whole copy, so a fork that copied cannot be missed, and only
     * one fork should copy at all — the rest wait and reuse it. Without a lock
     * spanning processes all six copy and publish, and one can unlink the artifact
     * another has already handed to its caller.
     */
    @Test
    void parallelJvmsOnAColdCacheAllGetARunnableBinary() throws Exception {
        Path cold = coldCache("cold-race");
        Files.createDirectories(cacheDir(cold));
        // Cleanup is each publisher's own staging file, not everything that looks
        // like one, so a foreign one has to survive.
        Path foreign = cacheDir(cold).resolve(FOREIGN_PART);
        Files.write(foreign, "not mine".getBytes(StandardCharsets.UTF_8));

        Path barrier = Files.createDirectories(cold.resolveSibling("barrier"));
        List<Process> forks = new ArrayList<>();
        for (int i = 1; i <= FORKS; i++) {
            forks.add(fork(cold, "fork" + i, barrier));
        }
        release(barrier, forks);

        // Sampled while they contend: a staging file leaves no trace once the race
        // is over.
        Set<String> staged = new LinkedHashSet<>();
        while (forks.stream().anyMatch(Process::isAlive)) {
            stagingFiles(cold).forEach(p -> staged.add(p.getFileName().toString()));
            Thread.sleep(1);  // a whole copy is far longer; polling frees the CPU
        }
        staged.remove(FOREIGN_PART);

        assertAllOk(verdicts(cold, forks));
        assertEquals(1, staged.size(), "every fork copied instead of reusing the winner: " + staged);
        assertEquals(payload.length, Files.size(binary(cold)));
        assertEquals(List.of(foreign), stagingFiles(cold), "leaked its own staging file");
        assertEquals("rwxr-xr-x",
            PosixFilePermissions.toString(Files.getPosixFilePermissions(binary(cold))));
    }

    /**
     * A cache entry that merely exists is not good enough. The first three are what
     * an extraction killed part-way leaves behind; the last two are not files a
     * binary can run from at all.
     */
    @ParameterizedTest(name = "{0}")
    @ValueSource(strings = {"zero-length", "truncated", "non-executable", "symlink", "directory"})
    void staleCacheEntriesAreReplaced(String kind) throws Exception {
        Path cold = coldCache("stale-" + kind);
        Files.createDirectories(cacheDir(cold));
        switch (kind) {
            case "zero-length": seed(cold, new byte[0], true); break;
            case "truncated": seed(cold, Arrays.copyOf(payload, payload.length / 2), true); break;
            case "non-executable": seed(cold, payload, false); break;
            case "symlink": Files.createSymbolicLink(binary(cold), stub()); break;
            default: Files.createDirectories(binary(cold)); break;
        }

        assertAllOk(List.of(runOne(cold)));

        assertTrue(Files.isRegularFile(binary(cold), LinkOption.NOFOLLOW_LINKS));
        assertEquals(payload.length, Files.size(binary(cold)));
        assertEquals(List.of(), stagingFiles(cold), "leaked a staging file");
    }

    /**
     * A JVM killed mid-extraction must leave nothing a later run would inherit —
     * the old guard was {@code !Files.exists(target)}, so a truncated file was
     * trusted forever. Recovery also proves the lock is crash-released: the victim
     * dies holding it, and a lock that outlived it would make the next fork wait
     * out the timeout and refuse.
     */
    @Test
    void anInterruptedExtractionNeverPublishesAPartialBinary() throws Exception {
        // Retry until the kill lands mid-copy rather than fixing a round count: the
        // window is milliseconds wide and a loaded machine can miss it. Every
        // attempt asserts the invariant, so extra rounds only add coverage.
        boolean killedMidCopy = false;
        for (int round = 0; round < 5 && !killedMidCopy; round++) {
            Path cold = coldCache("killed-" + round);

            Process victim = fork(cold, "victim", null);
            awaitBytesInFlight(cold, victim);
            victim.destroyForcibly();
            victim.waitFor();

            if (Files.exists(binary(cold))) {
                assertEquals(payload.length, Files.size(binary(cold)),
                    "round " + round + ": published a truncated binary");
            } else {
                killedMidCopy = true;  // died with bytes on disk, nothing published
            }
            assertAllOk(List.of(runOne(cold)));
        }
        assertTrue(killedMidCopy, "never caught an extraction mid-copy; the window went untested");
    }

    /**
     * When publication cannot happen, extraction fails rather than writing through
     * the cache path: a half-written binary where callers expect a whole one is
     * worse than none. A read-only cache directory is the reachable form — the
     * unreachable one is a filesystem with no atomic rename, which publication
     * refuses outright rather than falling back to a copy.
     */
    @Test
    void extractionFailsClosedWithoutOverwritingAStaleEntry() throws Exception {
        Path cold = coldCache("fail-closed");
        Files.createDirectories(cacheDir(cold));
        seed(cold, Arrays.copyOf(payload, payload.length / 2), true);
        byte[] before = Files.readAllBytes(binary(cold));
        Files.setPosixFilePermissions(cacheDir(cold), PosixFilePermissions.fromString("r-x------"));
        try {
            String verdict = runOne(cold);

            assertTrue(verdict.contains("REFUSED"), "should refuse, got: " + verdict);
            assertTrue(verdict.contains(binary(cold).toString()),
                "refusal should name the path: " + verdict);
            assertArrayEquals(before, Files.readAllBytes(binary(cold)),
                "wrote through the cache path instead of failing closed");
        } finally {
            Files.setPosixFilePermissions(cacheDir(cold), PosixFilePermissions.fromString("rwx------"));
        }
    }

    /**
     * Two class loaders in one JVM each hold their own copy of the resolver, so the
     * intra-JVM monitor does not serialise them and both reach the file lock, where
     * {@code tryLock} throws instead of returning null. Without that exception
     * treated as "busy", one of the two fails on every run.
     */
    @Test
    void concurrentClassLoadersInOneJvmBothResolve() throws Exception {
        Path cold = coldCache("class-loaders");

        assertAllOk(List.of(runOne(cold, "-D" + ExtractProbe.LOADERS + "=true")));

        assertEquals(payload.length, Files.size(binary(cold)));
        assertEquals(List.of(), stagingFiles(cold), "leaked a staging file");
    }

    // --- harness ---------------------------------------------------------------

    /** One fork, start to verdict. */
    private static String runOne(Path cold, String... jvmArgs) throws Exception {
        List<Process> forks = List.of(fork(cold, "fork1", null, jvmArgs));
        return verdicts(cold, forks).get(0);
    }

    private static Process fork(Path cold, String label, Path barrier, String... jvmArgs)
            throws IOException {
        List<String> command = new ArrayList<>(List.of(
            Paths.get(System.getProperty("java.home"), "bin", "java").toString(),
            "-Djava.io.tmpdir=" + cold,
            "-D" + ExtractProbe.CACHE_DIR + "=" + cacheDir(cold),
            "-D" + ExtractProbe.SIZE + "=" + payload.length));
        command.addAll(Arrays.asList(jvmArgs));
        if (barrier != null) {
            command.add("-D" + ExtractProbe.BARRIER + "=" + barrier);
        }
        // The synthetic jar first, so its natives/ entry shadows any real binary a
        // packaged build copied into resources.
        command.addAll(List.of("-cp",
            jar + File.pathSeparator + classes(BinaryResolver.class)
                + File.pathSeparator + classes(ExtractProbe.class),
            ExtractProbe.class.getName(), label));

        ProcessBuilder pb = new ProcessBuilder(command);
        // Reach the extraction branch: no env override, no vibium on PATH.
        pb.environment().remove("VIBIUM_BIN_PATH");
        pb.environment().put("PATH", "/nonexistent-vibium-path");
        pb.redirectErrorStream(true);
        pb.redirectOutput(cold.resolveSibling(label + ".log").toFile());
        return pb.start();
    }

    /** Let every fork reach the barrier, then release them together. */
    private static void release(Path barrier, List<Process> forks) throws Exception {
        long deadline = System.nanoTime() + TimeUnit.SECONDS.toNanos(120);
        while (forks.stream().allMatch(Process::isAlive)
                && count(barrier, "ready-") < forks.size()) {
            assertTrue(System.nanoTime() - deadline < 0, "forks never reached the barrier");
            Thread.sleep(1);
        }
        Files.createFile(barrier.resolve("go"));
    }

    private static long count(Path dir, String prefix) throws IOException {
        try (Stream<Path> entries = Files.list(dir)) {
            return entries.filter(p -> p.getFileName().toString().startsWith(prefix)).count();
        }
    }

    private static List<String> verdicts(Path cold, List<Process> forks) throws Exception {
        List<String> verdicts = new ArrayList<>();
        for (int i = 0; i < forks.size(); i++) {
            String label = "fork" + (i + 1);
            if (!forks.get(i).waitFor(120, TimeUnit.SECONDS)) {
                forks.get(i).destroyForcibly();
                throw new IllegalStateException(label + " never finished");
            }
            Path log = cold.resolveSibling(label + ".log");
            String text = new String(Files.readAllBytes(log), StandardCharsets.UTF_8);
            verdicts.add(Stream.of(text.split("\\R"))
                .filter(line -> line.startsWith("RESULT " + label + " "))
                .findFirst().orElse("NO_VERDICT " + text.trim()));
        }
        return verdicts;
    }

    private static void assertAllOk(List<String> verdicts) {
        List<String> bad = verdicts.stream().filter(v -> !v.contains(" OK ")).collect(toList());
        assertEquals(List.of(), bad, bad.size() + " of " + verdicts.size() + " forks failed");
    }

    /**
     * A shell script that prints its marker only once the whole file has been read:
     * the bulk sits in a here-document, so a truncated copy loses the terminator
     * and never reaches the echo.
     */
    private static byte[] payload(int size) {
        String filler = "# vibium embedded binary filler 0123456789abcdef0123456789abcd\n";
        String tail = "VIBIUM_PAYLOAD_END\necho " + ExtractProbe.MARKER + "\nexit 0\n";
        StringBuilder sb = new StringBuilder(size + 256)
            .append("#!/bin/sh\n: <<'VIBIUM_PAYLOAD_END'\n");
        while (sb.length() + filler.length() + tail.length() <= size) {
            sb.append(filler);
        }
        int shortfall = size - sb.length() - tail.length();
        if (shortfall > 0) {
            sb.append("#".repeat(shortfall - 1)).append('\n');
        }
        return sb.append(tail).toString().getBytes(StandardCharsets.US_ASCII);
    }

    private static Path coldCache(String name) throws IOException {
        return Files.createDirectories(work.resolve(name).resolve("tmp"));
    }

    private static Path cacheDir(Path cold) {
        return cold.resolve("vibium-" + VERSION);
    }

    private static Path binary(Path cold) {
        return cacheDir(cold).resolve(PlatformDetector.executableName());
    }

    private static List<Path> stagingFiles(Path cold) throws IOException {
        if (!Files.isDirectory(cacheDir(cold))) {
            return List.of();
        }
        try (Stream<Path> entries = Files.list(cacheDir(cold))) {
            return entries.filter(p -> p.getFileName().toString().endsWith(".part")).collect(toList());
        }
    }

    /** Identity of a usable artifact, so distinct values count publications. */
    private static void seed(Path cold, byte[] bytes, boolean executable) throws IOException {
        Files.write(binary(cold), bytes);
        Files.setPosixFilePermissions(binary(cold),
            PosixFilePermissions.fromString(executable ? "rwxr-xr-x" : "rw-------"));
    }

    private static Path stub() throws IOException {
        Path path = Files.createTempFile(work, "stub", "");
        Files.write(path, "#!/bin/sh\necho stub\n".getBytes(StandardCharsets.US_ASCII));
        Files.setPosixFilePermissions(path, PosixFilePermissions.fromString("rwxr-xr-x"));
        return path;
    }

    /** Return once the victim has bytes on disk, so destroying it lands mid-copy. */
    private static void awaitBytesInFlight(Path cold, Process victim) throws Exception {
        long deadline = System.nanoTime() + TimeUnit.SECONDS.toNanos(60);
        while (victim.isAlive() && System.nanoTime() - deadline < 0) {
            for (Path staged : stagingFiles(cold)) {
                if (sizeOrZero(staged) > 0) {
                    return;
                }
            }
            if (sizeOrZero(binary(cold)) > 0) {
                return;  // the unfixed resolver writes straight to the cache path
            }
            // Well inside a copy of this size, and polling leaves the CPU to the
            // child that is doing the copying.
            Thread.sleep(1);
        }
    }

    /** Zero for a path that vanished between being listed and being measured. */
    private static long sizeOrZero(Path path) {
        try {
            return Files.size(path);
        } catch (IOException e) {
            return 0;
        }
    }

    private static String classes(Class<?> type) {
        try {
            return Paths.get(type.getProtectionDomain().getCodeSource().getLocation().toURI())
                .toString();
        } catch (Exception e) {
            throw new IllegalStateException("cannot locate classes for " + type, e);
        }
    }

    private static boolean isWindows() {
        return System.getProperty("os.name", "").toLowerCase().contains("windows");
    }
}
