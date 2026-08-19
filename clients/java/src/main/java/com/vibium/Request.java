package com.vibium;

import com.google.gson.JsonArray;
import com.google.gson.JsonElement;
import com.google.gson.JsonObject;
import com.vibium.internal.BiDiClient;

import java.util.LinkedHashMap;
import java.util.Map;

/**
 * Network request info.
 */
public class Request {

    private static final long POST_DATA_TIMEOUT_MS = 500;

    private final BiDiClient client;
    private final String url;
    private final String method;
    private final Map<String, String> headers;
    private final String requestId;

    Request(BiDiClient client, JsonObject params) {
        this.client = client;

        // Extract from either top-level or nested request object
        JsonObject req = params.has("request") && params.get("request").isJsonObject()
            ? params.getAsJsonObject("request")
            : params;

        this.url = req.has("url") ? req.get("url").getAsString() : "";
        this.method = req.has("method") ? req.get("method").getAsString() : "";
        this.requestId = req.has("request") ? req.get("request").getAsString()
            : (req.has("requestId") ? req.get("requestId").getAsString()
            : (params.has("requestId") ? params.get("requestId").getAsString() : ""));
        this.headers = parseHeaders(req);
    }

    /** Get the request URL. */
    public String url() { return url; }

    /** Get the HTTP method. */
    public String method() { return method; }

    /** Get the request headers. */
    public Map<String, String> headers() { return headers; }

    /** Get the request ID. */
    public String requestId() { return requestId; }

    /**
     * Get the request body, or null when it is unavailable.
     * Intercepted requests may not expose their body until they are continued,
     * so this best-effort lookup is bounded to avoid blocking a route handler.
     */
    public String postData() {
        if (requestId.isEmpty()) return null;
        try {
            JsonObject params = new JsonObject();
            params.addProperty("dataType", "request");
            params.addProperty("request", requestId);
            JsonObject result = client.send("network.getData", params, POST_DATA_TIMEOUT_MS);
            if (result.has("bytes") && result.get("bytes").isJsonObject()) {
                JsonObject bytes = result.getAsJsonObject("bytes");
                return bytes.has("value") ? bytes.get("value").getAsString() : null;
            }
        } catch (Exception ignored) {
            // Request bodies are best-effort: browsers may not retain every body.
        }
        return null;
    }

    private static Map<String, String> parseHeaders(JsonObject obj) {
        Map<String, String> map = new LinkedHashMap<>();
        if (obj.has("headers") && obj.get("headers").isJsonArray()) {
            JsonArray arr = obj.getAsJsonArray("headers");
            for (JsonElement el : arr) {
                JsonObject header = el.getAsJsonObject();
                String name = header.get("name").getAsString();
                JsonObject value = header.getAsJsonObject("value");
                map.put(name, value.get("value").getAsString());
            }
        }
        return map;
    }

    @Override
    public String toString() {
        return "Request{method='" + method + "', url='" + url + "'}";
    }
}
