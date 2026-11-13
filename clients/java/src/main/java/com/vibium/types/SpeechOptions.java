package com.vibium.types;

/** What say and listen share. An unset value takes the runtime's default. */
public abstract class SpeechOptions<T extends SpeechOptions<T>> {
    private String target;
    private String provider;
    private String model;
    private String language;
    protected abstract T self();
    /** "session", "local", or a path ending in .wav. */
    public T target(String value) { target = value; return self(); }
    public T provider(String value) { provider = value; return self(); }
    public T model(String value) { model = value; return self(); }
    public T language(String value) { language = value; return self(); }
    public String target() { return target; }
    public String provider() { return provider; }
    public String model() { return model; }
    public String language() { return language; }
}
