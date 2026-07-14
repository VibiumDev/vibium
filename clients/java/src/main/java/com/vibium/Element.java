package com.vibium;

import com.google.gson.Gson;
import com.google.gson.JsonArray;
import com.google.gson.JsonObject;
import com.vibium.internal.BiDiClient;
import com.vibium.types.*;

import java.util.*;

/**
 * Represents a resolved DOM element.
 */
public class Element {

    private static final Gson GSON = new Gson();

    private final BiDiClient client;
    private final String contextId;
    private final String selector;
    private final int index;
    private final ElementInfo info;

    Element(BiDiClient client, String contextId, String selector, int index, ElementInfo info) {
        this.client = client;
        this.contextId = contextId;
        this.selector = selector;
        this.index = index;
        this.info = info;
    }

    /** Get info about this element (tag, text, bounding box). */
    public ElementInfo info() { return info; }

    // ── Interaction ─────────────────────────────────────────────

    /** Click the element. */
    public void click() { sendAction("vibium:element.click"); }

    /** Double-click the element. */
    public void dblclick() { sendAction("vibium:element.dblclick"); }

    /** Fill an input field (clears existing content first). */
    public void fill(String value) {
        JsonObject params = elementParams();
        params.addProperty("value", value);
        client.send("vibium:element.fill", params);
    }

    /** Type text character by character (appends). */
    public void type(String text) {
        JsonObject params = elementParams();
        params.addProperty("text", text);
        client.send("vibium:element.type", params);
    }

    /** Press a key or key combination. */
    public void press(String key) {
        JsonObject params = elementParams();
        params.addProperty("key", key);
        client.send("vibium:element.press", params);
    }

    /** Clear the input field. */
    public void clear() { sendAction("vibium:element.clear"); }

    /** Check a checkbox. */
    public void check() { sendAction("vibium:element.check"); }

    /** Uncheck a checkbox. */
    public void uncheck() { sendAction("vibium:element.uncheck"); }

    /** Select a dropdown option by value. */
    public void selectOption(String value) {
        JsonObject params = elementParams();
        params.addProperty("value", value);
        client.send("vibium:element.selectOption", params);
    }

    /** Hover over the element. */
    public void hover() { sendAction("vibium:element.hover"); }

    /** Focus the element. */
    public void focus() { sendAction("vibium:element.focus"); }

    /** Drag this element to a target element. */
    public void dragTo(Element target) {
        JsonObject params = elementParams();
        // The engine reads the target as a nested element-params object under the
        // "target" key, not flat "targetSelector"/"targetIndex" (issue #134).
        JsonObject targetParams = new JsonObject();
        targetParams.addProperty("selector", target.selector);
        if (target.index > 0) {
            targetParams.addProperty("index", target.index);
        }
        params.add("target", targetParams);
        client.send("vibium:element.dragTo", params);
    }

    /** Tap the element (touch). */
    public void tap() { sendAction("vibium:element.tap"); }

    /** Scroll the element into view. */
    public void scrollIntoView() { sendAction("vibium:element.scrollIntoView"); }

    /** Dispatch a DOM event. */
    public void dispatchEvent(String eventType) {
        dispatchEvent(eventType, null);
    }

    /** Dispatch a DOM event with init options. */
    public void dispatchEvent(String eventType, Map<String, Object> eventInit) {
        JsonObject params = elementParams();
        // The engine reads the event name from "eventType", not "type" (issue #132).
        params.addProperty("eventType", eventType);
        if (eventInit != null) {
            params.add("eventInit", GSON.toJsonTree(eventInit));
        }
        client.send("vibium:element.dispatchEvent", params);
    }

    /** Set files on a file input. */
    public void setFiles(List<String> files) {
        JsonObject params = elementParams();
        params.add("files", GSON.toJsonTree(files));
        client.send("vibium:element.setFiles", params);
    }

    /** Highlight the element visually. */
    public void highlight() { sendAction("vibium:element.highlight"); }

    // ── State Queries ───────────────────────────────────────────

    /** Get the text content (trimmed). */
    public String text() {
        JsonObject result = client.send("vibium:element.text", elementParams());
        return result.get("text").getAsString();
    }

    /** Get the inner text (rendered). */
    public String innerText() {
        JsonObject result = client.send("vibium:element.innerText", elementParams());
        return result.get("text").getAsString();
    }

    /** Get the outer HTML. */
    public String html() {
        JsonObject result = client.send("vibium:element.html", elementParams());
        return result.get("html").getAsString();
    }

    /** Get the input value. */
    public String value() {
        JsonObject result = client.send("vibium:element.value", elementParams());
        return result.get("value").getAsString();
    }

    /** Get an attribute value. */
    public String attr(String name) {
        JsonObject params = elementParams();
        params.addProperty("name", name);
        JsonObject result = client.send("vibium:element.attr", params);
        if (result.has("value") && !result.get("value").isJsonNull()) {
            return result.get("value").getAsString();
        }
        return null;
    }

    /** Alias for attr(). Playwright compat. */
    public String getAttribute(String name) {
        return attr(name);
    }

    /** Get the bounding box. */
    public BoundingBox bounds() {
        JsonObject result = client.send("vibium:element.bounds", elementParams());
        return new BoundingBox(
            result.get("x").getAsDouble(),
            result.get("y").getAsDouble(),
            result.get("width").getAsDouble(),
            result.get("height").getAsDouble()
        );
    }

    /** Alias for bounds(). Playwright compat. */
    public BoundingBox boundingBox() {
        return bounds();
    }

    /** Check if the element is visible. */
    public boolean isVisible() {
        JsonObject result = client.send("vibium:element.isVisible", elementParams());
        return result.get("visible").getAsBoolean();
    }

    /** Check if the element is hidden. */
    public boolean isHidden() {
        JsonObject result = client.send("vibium:element.isHidden", elementParams());
        return result.get("hidden").getAsBoolean();
    }

    /** Check if the element is enabled. */
    public boolean isEnabled() {
        JsonObject result = client.send("vibium:element.isEnabled", elementParams());
        return result.get("enabled").getAsBoolean();
    }

    /** Check if the element is checked. */
    public boolean isChecked() {
        JsonObject result = client.send("vibium:element.isChecked", elementParams());
        return result.get("checked").getAsBoolean();
    }

    /** Check if the element is editable. */
    public boolean isEditable() {
        JsonObject result = client.send("vibium:element.isEditable", elementParams());
        return result.get("editable").getAsBoolean();
    }

    /** Get the ARIA role. */
    public String role() {
        JsonObject result = client.send("vibium:element.role", elementParams());
        return result.get("role").getAsString();
    }

    /** Get the accessible label. */
    public String label() {
        JsonObject result = client.send("vibium:element.label", elementParams());
        return result.get("label").getAsString();
    }

    /** Take a screenshot of the element, returns PNG bytes. */
    public byte[] screenshot() {
        JsonObject result = client.send("vibium:element.screenshot", elementParams());
        String data = result.get("data").getAsString();
        return Base64.getDecoder().decode(data);
    }

    // ── Waiting ─────────────────────────────────────────────────

    /** Wait for the element to reach a state. Default: "visible". */
    public void waitUntil() {
        waitUntil("visible", null);
    }

    /** Wait for the element to reach a specified state. */
    public void waitUntil(String state) {
        waitUntil(state, null);
    }

    /** Wait for the element to reach a specified state with options. */
    public void waitUntil(String state, FindOptions options) {
        JsonObject params = elementParams();
        if (state != null) params.addProperty("state", state);
        if (options != null && options.timeout() != null) {
            params.addProperty("timeout", options.timeout());
        }
        client.send("vibium:element.waitFor", params);
    }

    // ── Scoped Finding ──────────────────────────────────────────

    /** Find a child element by CSS selector. */
    public Element find(String childSelector) {
        return find(childSelector, (FindOptions) null);
    }

    /** Find a child element by CSS selector with options. */
    public Element find(String childSelector, FindOptions options) {
        JsonObject params = elementParams();
        params.addProperty("childSelector", childSelector);
        if (options != null && options.timeout() != null) {
            params.addProperty("timeout", options.timeout());
        }
        JsonObject result = client.send("vibium:element.find", params);
        return elementFromResult(result);
    }

    /** Find a child element by semantic selector. */
    public Element find(SelectorOptions childOptions) {
        JsonObject params = elementParams();
        for (Map.Entry<String, Object> entry : childOptions.toParams().entrySet()) {
            params.add(entry.getKey(), GSON.toJsonTree(entry.getValue()));
        }
        JsonObject result = client.send("vibium:element.find", params);
        return elementFromResult(result);
    }

    /** Find all child elements by CSS selector. */
    public List<Element> findAll(String childSelector) {
        return findAll(childSelector, (FindOptions) null);
    }

    /** Find all child elements by CSS selector with options. */
    public List<Element> findAll(String childSelector, FindOptions options) {
        JsonObject params = elementParams();
        params.addProperty("childSelector", childSelector);
        if (options != null && options.timeout() != null) {
            params.addProperty("timeout", options.timeout());
        }
        JsonObject result = client.send("vibium:element.findAll", params);
        return elementsFromResult(result);
    }

    /** Find all child elements by semantic selector. */
    public List<Element> findAll(SelectorOptions childOptions) {
        JsonObject params = elementParams();
        for (Map.Entry<String, Object> entry : childOptions.toParams().entrySet()) {
            params.add(entry.getKey(), GSON.toJsonTree(entry.getValue()));
        }
        JsonObject result = client.send("vibium:element.findAll", params);
        return elementsFromResult(result);
    }

    // ── Internal ────────────────────────────────────────────────

    private JsonObject elementParams() {
        JsonObject params = new JsonObject();
        params.addProperty("context", contextId);
        params.addProperty("selector", selector);
        if (index > 0) {
            params.addProperty("index", index);
        }
        return params;
    }

    private void sendAction(String method) {
        client.send(method, elementParams());
    }

    private Element elementFromResult(JsonObject result) {
        String sel = result.has("selector") ? result.get("selector").getAsString() : "";
        int idx = result.has("index") ? result.get("index").getAsInt() : 0;
        ElementInfo eInfo = parseElementInfo(result);
        return new Element(client, contextId, sel, idx, eInfo);
    }

    private List<Element> elementsFromResult(JsonObject result) {
        List<Element> elements = new ArrayList<>();
        com.google.gson.JsonArray arr = result.has("elements") ? result.getAsJsonArray("elements") : new com.google.gson.JsonArray();
        for (int i = 0; i < arr.size(); i++) {
            JsonObject el = arr.get(i).getAsJsonObject();
            String sel = el.has("selector") ? el.get("selector").getAsString() : "";
            ElementInfo eInfo = parseElementInfo(el);
            elements.add(new Element(client, contextId, sel, i, eInfo));
        }
        return elements;
    }

    private ElementInfo parseElementInfo(JsonObject obj) {
        String tag = obj.has("tag") ? obj.get("tag").getAsString() : "";
        String txt = obj.has("text") ? obj.get("text").getAsString() : "";
        BoundingBox box = null;
        if (obj.has("box") && obj.get("box").isJsonObject()) {
            JsonObject b = obj.getAsJsonObject("box");
            box = new BoundingBox(
                b.get("x").getAsDouble(),
                b.get("y").getAsDouble(),
                b.get("width").getAsDouble(),
                b.get("height").getAsDouble()
            );
        }
        return new ElementInfo(tag, txt, box);
    }

    @Override
    public String toString() {
        return "Element{selector='" + selector + "', info=" + info + "}";
    }
}
