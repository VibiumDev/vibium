package com.vibium.process;

import com.vibium.VibiumException;
import org.apache.commons.compress.archivers.tar.TarArchiveEntry;
import org.apache.commons.compress.archivers.tar.TarArchiveInputStream;
import org.apache.commons.compress.compressors.gzip.GzipCompressorInputStream;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.BufferedInputStream;
import java.io.BufferedOutputStream;
import java.io.IOException;
import java.io.InputStream;
import java.io.OutputStream;
import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.StandardCopyOption;
import java.nio.file.attribute.PosixFilePermission;
import java.time.Duration;
import java.util.EnumSet;
import java.util.Set;

final class ClickerDownloader {
    private static final Logger log = LoggerFactory.getLogger(ClickerDownloader.class);

    private ClickerDownloader() {}

    static Path downloadToCacheIfNeeded() {
        String pkg = "@vibium/" + Platform.npmPlatform() + "-" + Platform.npmArch();
        String version = VibiumVersion.currentVersionOrNull();
        if (version == null) {
            version = "latest";
        }

        Path dest = cachePath(version);
        if (Files.isRegularFile(dest)) {
            return dest;
        }

        try {
            Files.createDirectories(dest.getParent());
        } catch (IOException e) {
            throw new VibiumException("Failed to create cache directory for clicker: " + dest.getParent(), e);
        }

        // Double-check after mkdirs (race with another process).
        if (Files.isRegularFile(dest)) {
            return dest;
        }

        String tarballUrl = NpmRegistry.resolveTarballUrl(pkg, version);
        log.info("Downloading Clicker from npm: {} ({})", pkg, version);

        Path tmp;
        try {
            tmp = Files.createTempFile(dest.getParent(), "clicker-", ".tmp");
        } catch (IOException e) {
            throw new VibiumException("Failed to create temp file for Clicker download", e);
        }

        try {
            downloadFile(tarballUrl, tmp);
            extractBinaryFromTarGz(tmp, dest, Platform.binaryName());
            makeExecutable(dest);
            return dest;
        } finally {
            try {
                Files.deleteIfExists(tmp);
            } catch (IOException ignored) {
            }
        }
    }

    private static Path cachePath(String version) {
        // Versioned to avoid accidental mixing across releases.
        return VibiumCache.rootDir()
                .resolve("clicker")
                .resolve(version)
                .resolve(Platform.binaryName());
    }

    private static void downloadFile(String url, Path destTmp) {
        try {
            HttpClient client = HttpClient.newBuilder()
                    .connectTimeout(Duration.ofSeconds(30))
                    .followRedirects(HttpClient.Redirect.NORMAL)
                    .build();

            HttpRequest req = HttpRequest.newBuilder()
                    .uri(URI.create(url))
                    .timeout(Duration.ofMinutes(2))
                    .header("User-Agent", "vibium-java")
                    .GET()
                    .build();

            HttpResponse<Path> resp = client.send(req, HttpResponse.BodyHandlers.ofFile(destTmp));
            if (resp.statusCode() != 200) {
                throw new VibiumException("Failed to download Clicker tarball: " + resp.statusCode() + " for " + url);
            }
        } catch (VibiumException e) {
            throw e;
        } catch (Exception e) {
            throw new VibiumException("Failed to download Clicker tarball: " + url, e);
        }
    }

    private static void extractBinaryFromTarGz(Path tarGz, Path destBinary, String binaryName) {
        String expectedSuffix = "/bin/" + binaryName;

        Path extractedTmp;
        try {
            extractedTmp = Files.createTempFile(destBinary.getParent(), binaryName, ".extracted");
        } catch (IOException e) {
            throw new VibiumException("Failed to create temp file for Clicker extraction", e);
        }

        boolean found = false;
        try (InputStream in = new BufferedInputStream(Files.newInputStream(tarGz));
             GzipCompressorInputStream gzIn = new GzipCompressorInputStream(in);
             TarArchiveInputStream tarIn = new TarArchiveInputStream(gzIn, StandardCharsets.UTF_8.name())) {

            TarArchiveEntry entry;
            while ((entry = (TarArchiveEntry) tarIn.getNextEntry()) != null) {
                if (entry.isDirectory()) {
                    continue;
                }
                String name = entry.getName();
                // NPM tarballs usually prefix paths with "package/".
                if (name == null) continue;
                name = name.replace('\\', '/');
                if (!name.endsWith(expectedSuffix)) {
                    continue;
                }

                try (OutputStream out = new BufferedOutputStream(Files.newOutputStream(extractedTmp))) {
                    tarIn.transferTo(out);
                }

                found = true;
                break;
            }
        } catch (IOException e) {
            throw new VibiumException("Failed to extract Clicker from npm tarball", e);
        }

        if (!found) {
            try {
                Files.deleteIfExists(extractedTmp);
            } catch (IOException ignored) {
            }
            throw new VibiumException("Downloaded npm package did not contain " + expectedSuffix);
        }

        try {
            Files.move(extractedTmp, destBinary, StandardCopyOption.REPLACE_EXISTING, StandardCopyOption.ATOMIC_MOVE);
        } catch (IOException e) {
            // Atomic move can fail across filesystems; fallback to non-atomic.
            try {
                Files.move(extractedTmp, destBinary, StandardCopyOption.REPLACE_EXISTING);
            } catch (IOException e2) {
                throw new VibiumException("Failed to place Clicker binary in cache: " + destBinary, e2);
            }
        }
    }

    private static void makeExecutable(Path path) {
        if (Platform.isWindows()) {
            return;
        }
        try {
            Set<PosixFilePermission> perms = EnumSet.of(
                    PosixFilePermission.OWNER_READ,
                    PosixFilePermission.OWNER_WRITE,
                    PosixFilePermission.OWNER_EXECUTE,
                    PosixFilePermission.GROUP_READ,
                    PosixFilePermission.GROUP_EXECUTE,
                    PosixFilePermission.OTHERS_READ,
                    PosixFilePermission.OTHERS_EXECUTE
            );
            Files.setPosixFilePermissions(path, perms);
        } catch (Exception ignored) {
        }
    }
}
