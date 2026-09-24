package com.vibium.types;

/** Per-call model settings. Credentials are read by the runtime from its environment. */
public abstract class ModelOptions<T extends ModelOptions<T>> {
    private String provider;
    private String model;
    private String aiBaseURL;
    private String reasoningEffort;
    protected abstract T self();
    public T provider(String value) { provider = value; return self(); }
    public T model(String value) { model = value; return self(); }
    /** The AI provider's API base URL (the model endpoint, not the site under test). */
    public T aiBaseURL(String value) { aiBaseURL = value; return self(); }
    public T reasoningEffort(String value) { reasoningEffort = value; return self(); }
    public String provider() { return provider; }
    public String model() { return model; }
    public String aiBaseURL() { return aiBaseURL; }
    public String reasoningEffort() { return reasoningEffort; }
}
