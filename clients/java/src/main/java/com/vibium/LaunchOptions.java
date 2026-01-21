package com.vibium;

public final class LaunchOptions {
    private boolean headless = false;
    private Integer port = null;
    private String clickerPath = null;
    private long timeoutMs = 30_000;

    public LaunchOptions headless(boolean headless) {
        this.headless = headless;
        return this;
    }

    public LaunchOptions port(int port) {
        this.port = port;
        return this;
    }

    public LaunchOptions clickerPath(String clickerPath) {
        this.clickerPath = clickerPath;
        return this;
    }

    public LaunchOptions timeoutMs(long timeoutMs) {
        this.timeoutMs = timeoutMs;
        return this;
    }

    public boolean isHeadless() {
        return headless;
    }

    public Integer getPort() {
        return port;
    }

    public String getClickerPath() {
        return clickerPath;
    }

    public long getTimeoutMs() {
        return timeoutMs;
    }
}

