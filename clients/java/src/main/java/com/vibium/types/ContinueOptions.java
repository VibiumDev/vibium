package com.vibium.types;

import java.util.LinkedHashMap;
import java.util.Map;

/**
 * Options for continuing an intercepted request with overrides.
 */
public class ContinueOptions {
    private String url;
    private String method;
    private Map<String, String> headers;
    private String postData;

    public ContinueOptions url(String url) { this.url = url; return this; }
    public ContinueOptions method(String method) { this.method = method; return this; }
    public ContinueOptions headers(Map<String, String> headers) { this.headers = headers; return this; }
    public ContinueOptions postData(String postData) { this.postData = postData; return this; }

    public Map<String, Object> toParams() {
        Map<String, Object> params = new LinkedHashMap<>();
        if (url != null) params.put("url", url);
        if (method != null) params.put("method", method);
        if (headers != null) params.put("headers", headers);
        // The engine reads the request body from "body", not "postData".
        if (postData != null) params.put("body", postData);
        return params;
    }
}
