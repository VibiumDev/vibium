package com.vibium.types;

import java.nio.file.Path;

/** Read-only input archive. A null record selects the existing live session. */
public final class VerifyOptions {
    private Path record;
    public VerifyOptions record(Path path) { this.record = path; return this; }
    public Path record() { return record; }
    public static VerifyOptions builder() { return new VerifyOptions(); }
    public VerifyOptions build() { return this; }
}
