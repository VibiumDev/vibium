package com.vibium.internal;

import com.vibium.errors.VibiumException;

/** A failure the engine gave a code of its own. The caller decides what the code means. */
final class CodedException extends VibiumException {
    final String code;

    CodedException(String code, String message) {
        super(message);
        this.code = code;
    }
}
