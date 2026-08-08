package com.vibium.types;

import java.util.Collections;
import java.util.List;
import java.util.Map;

/**
 * What Recording.stop() / stopChunk() delivered.
 */
public class RecordingResult {
    private final String path;
    private final byte[] bytes;
    private final Integer steps;
    private final Long durationMs;
    private final List<Map<String, Object>> videos;
    private final String videoUnavailable;

    public RecordingResult(String path, byte[] bytes, Integer steps, Long durationMs,
                           List<Map<String, Object>> videos, String videoUnavailable) {
        this.path = path;
        this.bytes = bytes;
        this.steps = steps;
        this.durationMs = durationMs;
        this.videos = videos == null ? Collections.emptyList() : videos;
        this.videoUnavailable = videoUnavailable;
    }

    /** Where the zip landed; null when it was returned inline instead. */
    public String path() { return path; }

    /** The zip itself; present only when no file was written. */
    public byte[] bytes() { return bytes; }

    public Integer steps() { return steps; }

    public Long durationMs() { return durationMs; }

    /** The recorded video tracks (context, durationMs, width, height). */
    public List<Map<String, Object>> videos() { return videos; }

    /** Why no video was recorded (video omitted on an engine without support), or null. */
    public String videoUnavailable() { return videoUnavailable; }
}
