package com.vibium.types;

import java.nio.file.Path;

/** Read-only input archive. A null record selects the existing live session. */
public final class CheckOptions extends ModelOptions<CheckOptions> {
    @Override protected CheckOptions self() { return this; }
    private Path record;
    public CheckOptions record(Path path) { this.record = path; return this; }
    public Path record() { return record; }
    public static CheckOptions builder() { return new CheckOptions(); }
    public CheckOptions build() { return this; }
}
