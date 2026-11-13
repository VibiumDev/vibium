package com.vibium.types;

/** What a say did. A failure throws a SpeechException instead. */
public final class SayResult {
    private String text;
    private String target;
    private String provider;
    private String model;
    private String voice;
    private double durationMs;
    private boolean cached;
    private String sha256;
    private String path;
    private String file;
    public String text() { return text; }
    public String target() { return target; }
    public String provider() { return provider; }
    public String model() { return model; }
    public String voice() { return voice; }
    public double durationMs() { return durationMs; }
    public boolean cached() { return cached; }
    /** SHA-256 of the audio that was played. */
    public String sha256() { return sha256; }
    /** Set for file targets, and null otherwise. */
    public String path() { return path; }
    /** Set when a file was played in place of text, and null otherwise. */
    public String file() { return file; }
}
