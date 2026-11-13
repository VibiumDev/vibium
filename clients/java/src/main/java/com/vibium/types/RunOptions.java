package com.vibium.types;

/** Optional settings for one live Run invocation. */
public final class RunOptions extends ModelOptions<RunOptions> {
    @Override protected RunOptions self() { return this; }
    private String baseURL;
    /** Site under test: opened first unless the current page already shares its
     * origin; relative navigation paths resolve against it. Not the AI provider
     * endpoint; that is aiBaseURL. */
    public RunOptions baseURL(String value) { this.baseURL = value; return this; }
    public String baseURL() { return baseURL; }
    public static RunOptions builder() { return new RunOptions(); }
    public RunOptions build() { return this; }
}
