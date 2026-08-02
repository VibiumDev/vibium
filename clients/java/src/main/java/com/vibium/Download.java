package com.vibium;

import com.google.gson.JsonObject;
import com.vibium.errors.VibiumException;
import com.vibium.internal.BiDiClient;

import java.util.concurrent.CompletableFuture;
import java.util.concurrent.TimeUnit;

/**
 * File download handle.
 */
public class Download {

    private static final long DOWNLOAD_TIMEOUT_MS = 300_000; // 5 minutes

    private final BiDiClient client;
    private final String url;
    private final String suggestedFilename;
    private final CompletableFuture<String> pathFuture = new CompletableFuture<>();

    Download(BiDiClient client, JsonObject params) {
        this.client = client;
        this.url = params.has("url") ? params.get("url").getAsString() : "";
        this.suggestedFilename = params.has("suggestedFilename")
            ? params.get("suggestedFilename").getAsString()
            : (params.has("filename") ? params.get("filename").getAsString() : "");
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

    /** Get the temp file path (waits for download to complete). */
    public String path() {
        try {
            return pathFuture.get(DOWNLOAD_TIMEOUT_MS, TimeUnit.MILLISECONDS);
        } catch (Exception e) {
            return null;
        }
    }

    /**
     * Called internally when the download completes.
     */
    void complete(String status, String filePath) {
        if (filePath != null) {
            pathFuture.complete(filePath);
        } else {
            pathFuture.complete(null);
        }
    }

    @Override
    public String toString() {
        return "Download{url='" + url + "', filename='" + suggestedFilename + "'}";
    }
}
