package com.vibium;

import com.google.gson.JsonArray;
import com.google.gson.JsonElement;
import com.google.gson.JsonObject;
import com.google.gson.JsonParser;
import com.vibium.internal.BiDiClient;

import java.util.Base64;
import java.util.LinkedHashMap;
import java.util.Map;

/**
 * Network response info.
 */
public class Response {

    private final BiDiClient client;
    private final String url;
    private final int status;
    private final Map<String, String> headers;
    private final String requestId;

    Response(BiDiClient client, JsonObject params) {
        this.client = client;

        JsonObject resp = params.has("response") && params.get("response").isJsonObject()
            ? params.getAsJsonObject("response")
            : params;

        this.url = resp.has("url") ? resp.get("url").getAsString() : "";
        this.status = resp.has("status") ? resp.get("status").getAsInt()
            : (resp.has("statusCode") ? resp.get("statusCode").getAsInt() : 0);
        JsonObject req = params.has("request") && params.get("request").isJsonObject()
            ? params.getAsJsonObject("request")
            : params;
        this.requestId = req.has("request") ? req.get("request").getAsString()
            : (params.has("requestId") ? params.get("requestId").getAsString() : "");
        this.headers = parseHeaders(resp);
    }

    /** Get the response URL. */
    public String url() { return url; }

    /** Get the HTTP status code. */
    public int status() { return status; }

    /** Get the response headers. */
    public Map<String, String> headers() { return headers; }

    /** Get the request ID. */
    public String requestId() { return requestId; }

    /** Get the response body as a string. */
    public String body() {
        try {
            JsonObject params = new JsonObject();
            params.addProperty("dataType", "response");
            params.addProperty("request", requestId);
            JsonObject result = client.send("network.getData", params);
            if (result.has("bytes") && result.get("bytes").isJsonObject()) {
                JsonObject bytes = result.getAsJsonObject("bytes");
                if (!bytes.has("value")) return null;
                String data = bytes.get("value").getAsString();
                if (bytes.has("type") && "base64".equals(bytes.get("type").getAsString())) {
                    return new String(Base64.getDecoder().decode(data), "UTF-8");
                }
                return data;
            }
        } catch (Exception ignored) {}
        return null;
    }

    /** Parse the response body as JSON. */
    public Object json() {
        String bodyStr = body();
        if (bodyStr != null) {
            JsonElement el = JsonParser.parseString(bodyStr);
            return Page.jsonToJava(el);
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
        return "Response{status=" + status + ", url='" + url + "'}";
    }
}
