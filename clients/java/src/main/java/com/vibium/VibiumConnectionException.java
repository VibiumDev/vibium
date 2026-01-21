package com.vibium;

public final class VibiumConnectionException extends VibiumException {
    public final String url;

    public VibiumConnectionException(String url, String message) {
        super(message);
        this.url = url;
    }

    public VibiumConnectionException(String url, String message, Throwable cause) {
        super(message, cause);
        this.url = url;
    }
}

