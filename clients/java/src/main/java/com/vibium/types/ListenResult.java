package com.vibium.types;

import java.util.List;

/** What a listen heard. Times are milliseconds from the start of the capture. */
public final class ListenResult {
    private String text;
    private boolean transcribed;
    private boolean speech;
    private List<Segment> segments;
    private List<Word> words;
    private Double onsetMs;
    private Double speechEndMs;
    private Double latencyMs;
    private String stoppedBy;
    private double durationMs;
    private String target;
    private String provider;
    private String model;
    private String path;
    private double lostMs;
    public String text() { return text; }
    /** False when the capture was not sent to a provider. */
    public boolean transcribed() { return transcribed; }
    /** False when no speech was detected, which only allowSilence lets through. */
    public boolean speech() { return speech; }
    public List<Segment> segments() { return segments == null ? List.of() : List.copyOf(segments); }
    /** Empty when the provider returns no word timings. */
    public List<Word> words() { return words == null ? List.of() : List.copyOf(words); }
    /** When speech started in the capture, or null when there was none. */
    public Double onsetMs() { return onsetMs; }
    public Double speechEndMs() { return speechEndMs; }
    /** Speech onset minus the end of the preceding say, or null with no say before it. */
    public Double latencyMs() { return latencyMs; }
    /** "silence", "duration", "eof" or "timeout". */
    public String stoppedBy() { return stoppedBy; }
    /** Length of the captured audio. */
    public double durationMs() { return durationMs; }
    public String target() { return target; }
    public String provider() { return provider; }
    public String model() { return model; }
    /** Where the captured audio was saved, or null. */
    public String path() { return path; }
    /** Audio the target dropped on its way here. The transcript may miss words. */
    public double lostMs() { return lostMs; }
    public static final class Segment {
        private double startMs;
        private double endMs;
        private String text;
        public double startMs() { return startMs; }
        public double endMs() { return endMs; }
        public String text() { return text; }
    }
    public static final class Word {
        private double startMs;
        private double endMs;
        private String word;
        public double startMs() { return startMs; }
        public double endMs() { return endMs; }
        public String word() { return word; }
    }
}
