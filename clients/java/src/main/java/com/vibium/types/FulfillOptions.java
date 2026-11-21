package com.vibium.types;

import java.util.LinkedHashMap;
import java.util.Map;

/**
 * Options for fulfilling an intercepted request.
 */
public class FulfillOptions {
    private Integer status;
    private Map<String, String> headers;
    private String contentType;
    private String body;

    public FulfillOptions status(int status) { this.status = status; return this; }
    public FulfillOptions headers(Map<String, String> headers) { this.headers = headers; return this; }
    public FulfillOptions contentType(String contentType) { this.contentType = contentType; return this; }
    public FulfillOptions body(String body) { this.body = body; return this; }

    public Map<String, Object> toParams() {
        Map<String, Object> params = new LinkedHashMap<>();
        // The engine reads the status under "statusCode", as the JS and
        // Python clients send it.
        if (status != null) params.put("statusCode", status);
        if (headers != null) params.put("headers", headers);
        if (contentType != null) params.put("contentType", contentType);
        if (body != null) params.put("body", body);
        return params;
    }
}
