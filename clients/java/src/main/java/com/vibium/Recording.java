package com.vibium;

import com.google.gson.Gson;
import com.google.gson.JsonObject;
import com.google.gson.reflect.TypeToken;
import com.vibium.internal.BiDiClient;
import com.vibium.types.ChunkOptions;
import com.vibium.types.RecordingOptions;
import com.vibium.types.RecordingResult;

import java.nio.file.Files;
import java.nio.file.Paths;
import java.text.SimpleDateFormat;
import java.util.Base64;
import java.util.Date;
import java.util.List;
import java.util.Map;

/**
 * Trace recording control.
 */
public class Recording {

    private static final Gson GSON = new Gson();
    private final BiDiClient client;
    private final String userContextId;

    Recording(BiDiClient client, String userContextId) {
        this.client = client;
        this.userContextId = userContextId;
    }

    /** Start recording. */
    public void start() {
        start(null);
    }

    /** Start recording with options. */
    public void start(RecordingOptions options) {
        JsonObject params = params();
        if (options != null) {
            for (Map.Entry<String, Object> entry : options.toParams().entrySet()) {
                params.add(entry.getKey(), GSON.toJsonTree(entry.getValue()));
            }
        }
        // The binary's working directory is not necessarily ours, so relative
        // paths resolve here before going over the wire. The default is
        // timestamped so a rerun never clobbers the previous run.
        String path = params.has("path")
            ? params.get("path").getAsString()
            : defaultRecordPath(params.has("name") ? params.get("name").getAsString() : null);
        params.addProperty("path", Paths.get(path).toAbsolutePath().toString());
        client.send("vibium:recording.start", params);
    }

    /** Stop recording and deliver the zip to the declared path. */
    public RecordingResult stop() {
        return stop(null);
    }

    /** Stop recording; path overrides the path declared at start. */
    public RecordingResult stop(String path) {
        JsonObject params = params();
        if (path != null) {
            params.addProperty("path", Paths.get(path).toAbsolutePath().toString());
        }
        return parseResult(client.send("vibium:recording.stop", params));
    }

    /** Start a recording chunk. */
    public void startChunk() {
        startChunk(null);
    }

    /** Start a recording chunk with options. */
    public void startChunk(ChunkOptions options) {
        JsonObject params = params();
        if (options != null) {
            for (Map.Entry<String, Object> entry : options.toParams().entrySet()) {
                params.add(entry.getKey(), GSON.toJsonTree(entry.getValue()));
            }
        }
        client.send("vibium:recording.startChunk", params);
    }

    /** Stop a recording chunk; without a path its bytes come back inline. */
    public RecordingResult stopChunk() {
        return stopChunk(null);
    }

    /** Stop a recording chunk and save to path. */
    public RecordingResult stopChunk(String path) {
        JsonObject params = params();
        if (path != null) {
            params.addProperty("path", Paths.get(path).toAbsolutePath().toString());
        }
        return parseResult(client.send("vibium:recording.stopChunk", params));
    }

    private static RecordingResult parseResult(JsonObject result) {
        List<Map<String, Object>> videos = null;
        if (result.has("videos")) {
            videos = GSON.fromJson(result.get("videos"),
                new TypeToken<List<Map<String, Object>>>() {}.getType());
        }
        return new RecordingResult(
            result.has("path") ? result.get("path").getAsString() : null,
            result.has("data") ? Base64.getDecoder().decode(result.get("data").getAsString()) : null,
            result.has("steps") ? result.get("steps").getAsInt() : null,
            result.has("durationMs") ? result.get("durationMs").getAsLong() : null,
            videos,
            result.has("videoUnavailable") ? result.get("videoUnavailable").getAsString() : null);
    }

    /**
     * Timestamped default destination so a rerun never clobbers the
     * previous artifact. The recording's name, sanitized, seeds the stem:
     * name "login" yields login-20260808-094123.zip.
     */
    private static String defaultRecordPath(String name) {
        String stem = name == null ? "" : name.replaceAll("[^A-Za-z0-9._-]", "-");
        stem = stem.replaceAll("^[-.]+|[-.]+$", "");
        if (stem.isEmpty()) {
            stem = "record";
        }
        String stamp = new SimpleDateFormat("yyyyMMdd-HHmmss").format(new Date());
        String path = stem + "-" + stamp + ".zip";
        for (int n = 2; Files.exists(Paths.get(path)); n++) {
            path = stem + "-" + stamp + "-" + n + ".zip";
        }
        return path;
    }

    /** Start a logical group. */
    public void startGroup(String name) {
        startGroup(name, null);
    }

    /** Start a logical group with location. */
    public void startGroup(String name, String location) {
        JsonObject params = params();
        params.addProperty("name", name);
        if (location != null) {
            params.addProperty("location", location);
        }
        client.send("vibium:recording.startGroup", params);
    }

    /** Stop the current logical group. */
    public void stopGroup() {
        client.send("vibium:recording.stopGroup", params());
    }

    private JsonObject params() {
        JsonObject p = new JsonObject();
        p.addProperty("userContext", userContextId);
        return p;
    }
}
