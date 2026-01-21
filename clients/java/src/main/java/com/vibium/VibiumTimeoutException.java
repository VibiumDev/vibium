package com.vibium;

public final class VibiumTimeoutException extends VibiumException {
    public final String operation;
    public final long timeoutMs;

    public VibiumTimeoutException(String operation, long timeoutMs) {
        super("Timeout after " + timeoutMs + "ms waiting for " + operation);
        this.operation = operation;
        this.timeoutMs = timeoutMs;
    }

    public VibiumTimeoutException(String operation, long timeoutMs, String detail) {
        super("Timeout after " + timeoutMs + "ms waiting for " + operation + (detail == null || detail.isBlank() ? "" : (": " + detail)));
        this.operation = operation;
        this.timeoutMs = timeoutMs;
    }

    public VibiumTimeoutException(String operation, long timeoutMs, Throwable cause) {
        super("Timeout after " + timeoutMs + "ms waiting for " + operation, cause);
        this.operation = operation;
        this.timeoutMs = timeoutMs;
    }
}
