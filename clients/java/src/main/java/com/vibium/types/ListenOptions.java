package com.vibium.types;

/** Optional settings for one listen. Times are in milliseconds. */
public final class ListenOptions extends SpeechOptions<ListenOptions> {
    @Override protected ListenOptions self() { return this; }
    private Integer silence;
    private Integer timeout;
    private Integer duration;
    private Boolean allowSilence;
    private String path;
    private Boolean transcribe;
    /** Trailing silence that ends the reply. 0 turns endpointing off. Default 1000. */
    public ListenOptions silence(int ms) { this.silence = ms; return this; }
    /** How long to wait for speech to start, from the call. Default 10000. */
    public ListenOptions timeout(int ms) { this.timeout = ms; return this; }
    /** Maximum capture length from the call. Default 30000. */
    public ListenOptions duration(int ms) { this.duration = ms; return this; }
    /** Return an empty transcript instead of throwing when nothing is heard. */
    public ListenOptions allowSilence(boolean value) { this.allowSilence = value; return this; }
    /** Also save the captured audio as a WAV file. */
    public ListenOptions path(String value) { this.path = value; return this; }
    /** false captures without a transcript. */
    public ListenOptions transcribe(boolean value) { this.transcribe = value; return this; }
    public Integer silence() { return silence; }
    public Integer timeout() { return timeout; }
    public Integer duration() { return duration; }
    public Boolean allowSilence() { return allowSilence; }
    public String path() { return path; }
    public Boolean transcribe() { return transcribe; }
    public static ListenOptions builder() { return new ListenOptions(); }
    public ListenOptions build() { return this; }
}
