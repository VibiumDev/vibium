package com.vibium.types;

/** Optional settings for one live Run invocation. */
public final class RunOptions extends ModelOptions<RunOptions> {
    @Override protected RunOptions self() { return this; }
    public static RunOptions builder() { return new RunOptions(); }
    public RunOptions build() { return this; }
}
