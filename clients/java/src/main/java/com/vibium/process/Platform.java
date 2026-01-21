package com.vibium.process;

public final class Platform {
    private static final String OS = System.getProperty("os.name").toLowerCase();
    private static final String ARCH = System.getProperty("os.arch").toLowerCase();

    private Platform() {}

    public static boolean isWindows() {
        return OS.contains("win");
    }

    public static boolean isMac() {
        return OS.contains("mac") || OS.contains("darwin");
    }

    public static boolean isLinux() {
        return OS.contains("nux") || OS.contains("nix") || OS.contains("linux");
    }

    public static String binaryName() {
        return isWindows() ? "clicker.exe" : "clicker";
    }

    public static String npmPlatform() {
        if (isWindows()) return "win32";
        if (isMac()) return "darwin";
        return "linux";
    }

    public static String npmArch() {
        if (ARCH.contains("aarch64") || ARCH.contains("arm64")) {
            return "arm64";
        }
        // Treat everything else as x64 by default.
        return "x64";
    }
}
