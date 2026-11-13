package com.vibium.errors;

/**
 * Thrown when say or listen fails. {@link #code()} names the failure, such
 * as {@code no_speech} or {@code not_configured}.
 */
public class SpeechException extends VibiumException {
    private final String code;

    public SpeechException(String code, String message) {
        super(message);
        this.code = code;
    }

    public String code() { return code; }

    @Override
    public String toString() {
        return getClass().getName() + ": " + code + ": " + getMessage();
    }
}
