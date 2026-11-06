package com.vibium.types;

/**
 * Element information captured at find time.
 */
public class ElementInfo {
    private final String tag;
    private final String text;
    private final BoundingBox box;
    private final String ref;
    private final String role;
    private final String name;
    private final String source;
    private final String selector;

    public ElementInfo(String tag, String text, BoundingBox box) {
        this(tag, text, box, null, null, null, null, null);
    }

    public ElementInfo(String tag, String text, BoundingBox box, String ref, String role, String name, String source, String selector) {
        this.tag = tag;
        this.text = text;
        this.box = box;
        this.ref = ref;
        this.role = role;
        this.name = name;
        this.source = source;
        this.selector = selector;
    }

    public String tag() { return tag; }
    public String text() { return text; }
    public BoundingBox box() { return box; }
    /** The @eN ref find registered; the CLI and MCP address the same element by it. Null before a find. */
    public String ref() { return ref; }
    public String role() { return role; }
    /** Accessible name. */
    public String name() { return name; }
    /** What backs the match: dom, tree, image, or vlm. */
    public String source() { return source; }
    /** The DOM selector of a DOM-backed match. Null for a visual region. */
    public String selector() { return selector; }

    @Override
    public String toString() {
        return "ElementInfo{tag='" + tag + "', text='" + text + "', box=" + box + (ref != null ? ", ref=" + ref : "") + "}";
    }
}
