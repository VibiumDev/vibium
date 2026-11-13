package com.vibium.types;

/** Optional settings for one say. */
public final class SayOptions extends SpeechOptions<SayOptions> {
    @Override protected SayOptions self() { return this; }
    private String file;
    private String voice;
    private Double speed;
    private Boolean cache;
    private String cacheDir;
    /** A WAV file to play as it is. Pass an empty text with it. */
    public SayOptions file(String path) { this.file = path; return this; }
    public SayOptions voice(String value) { this.voice = value; return this; }
    public SayOptions speed(double value) { this.speed = value; return this; }
    /** false renders again and replaces the cached audio. */
    public SayOptions cache(boolean value) { this.cache = value; return this; }
    public SayOptions cacheDir(String path) { this.cacheDir = path; return this; }
    public String file() { return file; }
    public String voice() { return voice; }
    public Double speed() { return speed; }
    public Boolean cache() { return cache; }
    public String cacheDir() { return cacheDir; }
    public static SayOptions builder() { return new SayOptions(); }
    public SayOptions build() { return this; }
}
