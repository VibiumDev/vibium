package com.vibium.process;

import java.io.InputStream;
import java.util.Properties;

final class VibiumVersion {
    private VibiumVersion() {}

    static String currentVersionOrNull() {
        // Maven may include pom.properties in built artifacts.
        // When running from source/classfiles, this can be missing; callers should handle null.
        String resource = "META-INF/maven/com.vibium/vibium-java/pom.properties";
        try (InputStream in = VibiumVersion.class.getClassLoader().getResourceAsStream(resource)) {
            if (in == null) {
                return fromRepoVersionFileOrNull();
            }
            Properties props = new Properties();
            props.load(in);
            String v = props.getProperty("version");
            if (v == null || v.isBlank()) {
                return fromRepoVersionFileOrNull();
            }
            return v.trim();
        } catch (Exception ignored) {
            return fromRepoVersionFileOrNull();
        }
    }

    private static String fromRepoVersionFileOrNull() {
        try {
            java.nio.file.Path dir = java.nio.file.Path.of(System.getProperty("user.dir")).toAbsolutePath();
            for (int i = 0; i < 6 && dir != null; i++) {
                java.nio.file.Path versionFile = dir.resolve("VERSION");
                if (java.nio.file.Files.isRegularFile(versionFile)) {
                    String v = java.nio.file.Files.readString(versionFile).trim();
                    return v.isBlank() ? null : v;
                }
                dir = dir.getParent();
            }
        } catch (Exception ignored) {
        }
        return null;
    }
}
