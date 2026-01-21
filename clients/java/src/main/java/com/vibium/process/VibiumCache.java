package com.vibium.process;

import java.nio.file.Path;

final class VibiumCache {
    private VibiumCache() {}

    static Path rootDir() {
        if (Platform.isMac()) {
            return Path.of(System.getProperty("user.home"), "Library", "Caches", "vibium");
        }
        if (Platform.isWindows()) {
            String localAppData = System.getenv("LOCALAPPDATA");
            if (localAppData != null && !localAppData.isBlank()) {
                return Path.of(localAppData, "vibium");
            }
            return Path.of(System.getProperty("user.home"), "AppData", "Local", "vibium");
        }

        String xdg = System.getenv("XDG_CACHE_HOME");
        if (xdg != null && !xdg.isBlank()) {
            return Path.of(xdg, "vibium");
        }
        return Path.of(System.getProperty("user.home"), ".cache", "vibium");
    }
}
