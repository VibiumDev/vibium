package com.vibium.types;

import java.nio.file.Path;

/** Read-only input archive. A null record selects the existing live session. */
public final class CheckOptions extends ModelOptions<CheckOptions> {
    @Override protected CheckOptions self() { return this; }
    private Path record;
    private String baseURL;
    public CheckOptions record(Path path) { this.record = path; return this; }
    public Path record() { return record; }
    /** Site under test: opened first unless the current page already shares its
     * origin; relative navigation paths resolve against it. Not the AI provider
     * endpoint; that is aiBaseURL. */
    public CheckOptions baseURL(String value) { this.baseURL = value; return this; }
    public String baseURL() { return baseURL; }
    public static CheckOptions builder() { return new CheckOptions(); }
    public CheckOptions build() { return this; }
}
