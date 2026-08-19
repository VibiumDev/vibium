package com.vibium;

import org.junit.jupiter.api.Test;

import java.io.InputStream;
import java.nio.file.Files;
import java.nio.file.Paths;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertNotNull;

/**
 * BinaryResolver keys its extraction cache on the vibium-version.txt resource.
 * Without it every jar extracts to vibium-unknown, and the first jar to run on
 * a machine pins its binary for every later version (#330).
 */
class BinaryVersionResourceTest {

    @Test
    void versionResourceMatchesRepoVersion() throws Exception {
        try (InputStream is = getClass().getClassLoader()
                .getResourceAsStream("vibium-version.txt")) {
            assertNotNull(is,
                "vibium-version.txt must be on the classpath so extraction dirs are versioned");
            String resource = new String(is.readAllBytes()).trim();
            String repo = new String(Files.readAllBytes(Paths.get("../../VERSION"))).trim();
            assertEquals(repo, resource);
        }
    }
}
