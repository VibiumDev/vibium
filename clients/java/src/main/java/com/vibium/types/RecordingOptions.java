package com.vibium.types;

import java.util.LinkedHashMap;
import java.util.Map;

/**
 * Options for starting a recording.
 */
public class RecordingOptions {
    private String name;
    private Boolean screenshots;
    private Boolean snapshots;
    private Boolean sources;
    private String title;
    private Boolean bidi;
    private String format;
    private Double quality;
    private Boolean video;
    private Integer videoWidth;
    private Integer videoHeight;
    private Integer videoFrameRate;
    private String path;

    public RecordingOptions name(String name) { this.name = name; return this; }
    public RecordingOptions screenshots(boolean screenshots) { this.screenshots = screenshots; return this; }
    public RecordingOptions snapshots(boolean snapshots) { this.snapshots = snapshots; return this; }
    public RecordingOptions sources(boolean sources) { this.sources = sources; return this; }
    public RecordingOptions title(String title) { this.title = title; return this; }
    public RecordingOptions bidi(boolean bidi) { this.bidi = bidi; return this; }
    public RecordingOptions format(String format) { this.format = format; return this; }
    public RecordingOptions quality(double quality) { this.quality = quality; return this; }

    /**
     * Video track (Firefox 154+, local browsers). Unset: record video if the
     * engine supports it. true (or any video dimension set): start fails if
     * the engine can't deliver. false: off.
     */
    public RecordingOptions video(boolean video) { this.video = video; return this; }
    /** Video dimensions in pixels (default: viewport). Implies video on. */
    public RecordingOptions videoSize(int width, int height) {
        this.videoWidth = width;
        this.videoHeight = height;
        return this;
    }
    /** Video frame rate (engine default if unset). Implies video on. */
    public RecordingOptions videoFrameRate(int frameRate) { this.videoFrameRate = frameRate; return this; }
    /** Where the recording zip lands at stop (default: record.zip). */
    public RecordingOptions path(String path) { this.path = path; return this; }

    public Map<String, Object> toParams() {
        Map<String, Object> params = new LinkedHashMap<>();
        if (name != null) params.put("name", name);
        if (screenshots != null) params.put("screenshots", screenshots);
        if (snapshots != null) params.put("snapshots", snapshots);
        if (sources != null) params.put("sources", sources);
        if (title != null) params.put("title", title);
        if (bidi != null) params.put("bidi", bidi);
        if (format != null) params.put("format", format);
        if (quality != null) params.put("quality", quality);
        // The flat video setters map onto the wire's nested video param.
        Map<String, Object> videoParams = new LinkedHashMap<>();
        if (videoWidth != null) videoParams.put("width", videoWidth);
        if (videoHeight != null) videoParams.put("height", videoHeight);
        if (videoFrameRate != null) videoParams.put("frameRate", videoFrameRate);
        if (video != null && !video) {
            params.put("video", false);
        } else if (!videoParams.isEmpty()) {
            params.put("video", videoParams);
        } else if (video != null) {
            params.put("video", true);
        }
        if (path != null) params.put("path", path);
        return params;
    }
}
