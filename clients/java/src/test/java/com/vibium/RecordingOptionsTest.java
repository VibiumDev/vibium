package com.vibium;

import com.vibium.types.RecordingOptions;
import org.junit.jupiter.api.Test;

import java.util.Map;

import static org.junit.jupiter.api.Assertions.*;

/**
 * A single video dimension must reach the wire alone so the engine derives
 * the other from the viewport aspect ratio, as the JS and Python clients
 * already allow (#409).
 */
class RecordingOptionsTest {

    @SuppressWarnings("unchecked")
    private Map<String, Object> videoParams(RecordingOptions options) {
        Object video = options.toParams().get("video");
        assertInstanceOf(Map.class, video, "video should be a params map");
        return (Map<String, Object>) video;
    }

    @Test
    void videoHeightAloneOmitsWidth() {
        Map<String, Object> video = videoParams(new RecordingOptions().videoHeight(480));
        assertEquals(480, video.get("height"));
        assertFalse(video.containsKey("width"), "width should stay unset");
    }

    @Test
    void videoWidthAloneOmitsHeight() {
        Map<String, Object> video = videoParams(new RecordingOptions().videoWidth(640));
        assertEquals(640, video.get("width"));
        assertFalse(video.containsKey("height"), "height should stay unset");
    }

    @Test
    void videoSizeZeroMeansUnset() {
        Map<String, Object> video = videoParams(new RecordingOptions().videoSize(0, 480));
        assertEquals(480, video.get("height"));
        assertFalse(video.containsKey("width"), "width 0 should stay unset");
    }

    @Test
    void videoSizeBothDimensionsStillWork() {
        Map<String, Object> video = videoParams(new RecordingOptions().videoSize(640, 480));
        assertEquals(640, video.get("width"));
        assertEquals(480, video.get("height"));
    }
}
