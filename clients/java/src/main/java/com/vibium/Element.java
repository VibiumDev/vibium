package com.vibium;

import com.google.gson.JsonObject;
import com.vibium.bidi.BiDiClient;

import java.util.Objects;

public final class Element {
    private final BiDiClient client;
    private final String contextId;
    private final String selector;
    private final ElementInfo info;

    Element(BiDiClient client, String contextId, String selector, ElementInfo info) {
        this.client = Objects.requireNonNull(client, "client");
        this.contextId = Objects.requireNonNull(contextId, "contextId");
        this.selector = Objects.requireNonNull(selector, "selector");
        this.info = Objects.requireNonNull(info, "info");
    }

    public ElementInfo info() {
        return info;
    }

    public void click() {
        click(null);
    }

    public void click(int timeoutMs) {
        click(ActionOptions.timeoutMs(timeoutMs));
    }

    public void click(ActionOptions options) {
        JsonObject params = new JsonObject();
        params.addProperty("context", contextId);
        params.addProperty("selector", selector);
        long requestTimeoutMs = 0;
        if (options != null && options.timeoutMs != null) {
            params.addProperty("timeout", options.timeoutMs);
            requestTimeoutMs = options.timeoutMs;
        }
        if (requestTimeoutMs > 0) {
            client.send("vibium:click", params, requestTimeoutMs);
        } else {
            client.send("vibium:click", params);
        }
    }

    public void type(String text) {
        type(text, null);
    }

    public void type(String text, int timeoutMs) {
        type(text, ActionOptions.timeoutMs(timeoutMs));
    }

    public void type(String text, ActionOptions options) {
        Objects.requireNonNull(text, "text");
        JsonObject params = new JsonObject();
        params.addProperty("context", contextId);
        params.addProperty("selector", selector);
        params.addProperty("text", text);
        long requestTimeoutMs = 0;
        if (options != null && options.timeoutMs != null) {
            params.addProperty("timeout", options.timeoutMs);
            requestTimeoutMs = options.timeoutMs;
        }
        if (requestTimeoutMs > 0) {
            client.send("vibium:type", params, requestTimeoutMs);
        } else {
            client.send("vibium:type", params);
        }
    }

    public String text() {
        JsonObject params = new JsonObject();
        params.addProperty(
                "functionDeclaration",
                "(selector) => {" +
                        "const el = document.querySelector(selector);" +
                        "return el ? (el.textContent || '').trim() : null;" +
                        "}"
        );
        JsonObject target = new JsonObject();
        target.addProperty("context", contextId);
        params.add("target", target);
        params.add("arguments", BiDiClient.argsString(selector));
        params.addProperty("awaitPromise", false);
        params.addProperty("resultOwnership", "root");

        JsonObject result = client.sendObj("script.callFunction", params);
        JsonObject inner = result.getAsJsonObject("result");
        String type = inner.get("type").getAsString();
        if ("null".equals(type)) {
            throw new VibiumElementNotFoundException(selector);
        }
        return inner.has("value") ? inner.get("value").getAsString() : "";
    }

    public String attribute(String name) {
        return getAttribute(name);
    }

    public String getAttribute(String name) {
        Objects.requireNonNull(name, "name");

        JsonObject params = new JsonObject();
        params.addProperty(
                "functionDeclaration",
                "(selector, attrName) => {" +
                        "const el = document.querySelector(selector);" +
                        "return el ? el.getAttribute(attrName) : null;" +
                        "}"
        );
        JsonObject target = new JsonObject();
        target.addProperty("context", contextId);
        params.add("target", target);
        params.add("arguments", BiDiClient.argsString(selector, name));
        params.addProperty("awaitPromise", false);
        params.addProperty("resultOwnership", "root");

        JsonObject result = client.sendObj("script.callFunction", params);
        JsonObject inner = result.getAsJsonObject("result");
        String type = inner.get("type").getAsString();
        if ("null".equals(type)) {
            return null;
        }
        return inner.has("value") && !inner.get("value").isJsonNull() ? inner.get("value").getAsString() : null;
    }

    public BoundingBox boundingBox() {
        JsonObject params = new JsonObject();
        params.addProperty(
                "functionDeclaration",
                "(selector) => {" +
                        "const el = document.querySelector(selector);" +
                        "if (!el) return null;" +
                        "const rect = el.getBoundingClientRect();" +
                        "return JSON.stringify({x: rect.x, y: rect.y, width: rect.width, height: rect.height});" +
                        "}"
        );
        JsonObject target = new JsonObject();
        target.addProperty("context", contextId);
        params.add("target", target);
        params.add("arguments", BiDiClient.argsString(selector));
        params.addProperty("awaitPromise", false);
        params.addProperty("resultOwnership", "root");

        JsonObject result = client.sendObj("script.callFunction", params);
        JsonObject inner = result.getAsJsonObject("result");
        String type = inner.get("type").getAsString();
        if ("null".equals(type)) {
            throw new VibiumElementNotFoundException(selector);
        }

        if (!inner.has("value") || inner.get("value").isJsonNull()) {
            throw new VibiumException("Failed to get bounding box for: " + selector);
        }

        String json = inner.get("value").getAsString();
        JsonObject box = BiDiClient.GSON.fromJson(json, JsonObject.class);
        return new BoundingBox(
                box.get("x").getAsDouble(),
                box.get("y").getAsDouble(),
                box.get("width").getAsDouble(),
                box.get("height").getAsDouble()
        );
    }

    public static final class ActionOptions {
        private final Integer timeoutMs;

        private ActionOptions(Integer timeoutMs) {
            this.timeoutMs = timeoutMs;
        }

        public static ActionOptions timeoutMs(int timeoutMs) {
            return new ActionOptions(timeoutMs);
        }
    }
}
