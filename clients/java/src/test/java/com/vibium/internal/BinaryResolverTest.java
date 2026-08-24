package com.vibium.internal;

import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.io.TempDir;

import java.io.ByteArrayInputStream;
import java.io.IOException;
import java.nio.charset.StandardCharsets;
import java.nio.file.DirectoryStream;
import java.nio.file.Files;
import java.nio.file.Path;

import static org.junit.jupiter.api.Assertions.*;

/**
 * extractTo must never publish a path to a half-written or non-executable
 * binary: bytes land in a temp file, the executable bit is set there, and an
 * atomic rename publishes the finished file (#329).
 */
class BinaryResolverTest {

    private static final boolean WINDOWS =
        System.getProperty("os.name", "").toLowerCase().contains("windows");

    private static ByteArrayInputStream bytes(String s) {
        return new ByteArrayInputStream(s.getBytes(StandardCharsets.UTF_8));
    }

    @Test
    void freshExtractionIsCompleteAndExecutable(@TempDir Path dir) throws IOException {
        Path target = dir.resolve("vibium");
        Path result = BinaryResolver.extractTo(bytes("binary-bytes"), dir, target, WINDOWS);

        assertEquals(target, result);
        assertEquals("binary-bytes", Files.readString(target));
        if (!WINDOWS) {
            assertTrue(Files.isExecutable(target), "published binary must be executable");
        }
    }

    @Test
    void staleNonExecutablePartialIsReplaced(@TempDir Path dir) throws IOException {
        Path target = dir.resolve("vibium");
        Files.writeString(target, "half-writ");
        if (!WINDOWS) {
            assertFalse(Files.isExecutable(target), "precondition: stale file is not executable");
        }

        BinaryResolver.extractTo(bytes("binary-bytes"), dir, target, false);

        assertEquals("binary-bytes", Files.readString(target));
        assertTrue(Files.isExecutable(target), "stale partial must be replaced with a complete binary");
    }

    @Test
    void existingExecutableIsReusedWithoutRewriting(@TempDir Path dir) throws IOException {
        Path target = dir.resolve("vibium");
        Files.writeString(target, "already-there");
        assertTrue(target.toFile().setExecutable(true));

        BinaryResolver.extractTo(bytes("newer-bytes"), dir, target, WINDOWS);

        assertEquals("already-there", Files.readString(target), "a usable binary is kept as is");
    }

    @Test
    void noTempFilesLeftBehind(@TempDir Path dir) throws IOException {
        Path target = dir.resolve("vibium");
        BinaryResolver.extractTo(bytes("binary-bytes"), dir, target, WINDOWS);

        try (DirectoryStream<Path> entries = Files.newDirectoryStream(dir, ".extract-*")) {
            assertFalse(entries.iterator().hasNext(), "temp files must not accumulate");
        }
    }
}
