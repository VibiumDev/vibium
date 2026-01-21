package com.vibium;

public class VibiumException extends RuntimeException {
    public VibiumException(String message) {
        super(message);
    }

    public VibiumException(String message, Throwable cause) {
        super(message, cause);
    }
}

