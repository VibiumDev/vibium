package com.vibium;

import com.google.gson.JsonObject;
import com.vibium.errors.VibiumException;
import com.vibium.internal.BiDiClient;

/**
 * File download handle.
 */
public class Download {

    private static final long DOWNLOAD_TIMEOUT_MS = 300_000; // 5 minutes

    private final BiDiClient client;
    private final String url;
    private final String suggestedFilename;
    private final String navigation;

    Download(BiDiClient client, JsonObject params) {
        this.client = client;
        this.url = params.has("url") ? params.get("url").getAsString() : "";
        this.suggestedFilename = params.has("suggestedFilename")
            ? params.get("suggestedFilename").getAsString()
            : (params.has("filename") ? params.get("filename").getAsString() : "");
        this.navigation = params.has("navigation") ? params.get("navigation").getAsString() : "";
    }

    /** Get the download URL. */
    public String url() { return url; }

    /** Get the suggested filename. */
    public String suggestedFilename() { return suggestedFilename; }

    /** Save the download to a path. */
    public void saveAs(String path) {
        String sourcePath = path();
        if (sourcePath == null) {
            throw new VibiumException("download did not complete; cannot save");
        }
        JsonObject params = new JsonObject();
        params.addProperty("sourcePath", sourcePath);
        params.addProperty("destPath", path);
        client.send("vibium:download.saveAs", params);
    }

    /**
     * Get the temp file path (waits for download to complete).
     *
     * Completion is awaited in the engine (vibium:download.await), which
     * tracks every download by its navigation id, so this client keeps no
     * pending-downloads map. Finished downloads answer immediately, so
     * asking repeatedly is fine.
     */
    public String path() {
        JsonObject params = new JsonObject();
        params.addProperty("navigation", navigation);
        params.addProperty("timeout", DOWNLOAD_TIMEOUT_MS);
        try {
            JsonObject result = client.send("vibium:download.await", params, DOWNLOAD_TIMEOUT_MS + 10_000);
            if (!result.has("status") || !"complete".equals(result.get("status").getAsString())) {
                return null;
            }
            String filepath = result.has("filepath") ? result.get("filepath").getAsString() : "";
            return filepath.isEmpty() ? null : filepath;
        } catch (Exception e) {
            return null;
        }
    }

    @Override
    public String toString() {
        return "Download{url='" + url + "', filename='" + suggestedFilename + "'}";
    }
}
