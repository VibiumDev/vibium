package com.vibium.types;

import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.Base64;
import java.util.LinkedHashMap;
import java.util.Map;

/**
 * Locator options for find: a natural-language description, a reference
 * image, a CSS selector, a tag, or semantic fields such as role and text.
 * Description and image stand alone; the others compose.
 */
public class SelectorOptions {
    private String description;
    private String image;
    private String css;
    private String tag;
    private String name;
    private String role;
    private String text;
    private String label;
    private String placeholder;
    private String alt;
    private String title;
    private String testid;
    private String xpath;
    private String near;
    private String strategy;
    private Boolean details;
    private String provider;
    private String model;
    private String aiBaseURL;
    private String reasoningEffort;
    private Long timeout;

    /** Natural-language description, e.g. "the Add to Cart button". Stands alone. */
    public SelectorOptions description(String description) { this.description = description; return this; }
    /** Reference image bytes (PNG or JPEG) to locate on screen. Stands alone. */
    public SelectorOptions image(byte[] png) { this.image = Base64.getEncoder().encodeToString(png); return this; }
    /** Reference image file (PNG or JPEG) to locate on screen. Stands alone. */
    public SelectorOptions image(Path file) {
        try {
            return image(Files.readAllBytes(file));
        } catch (IOException e) {
            throw new IllegalArgumentException("cannot read image " + file, e);
        }
    }
    /** CSS selector. Exact; never sent to a model. */
    public SelectorOptions css(String css) { this.css = css; return this; }
    /** HTML tag name. */
    public SelectorOptions tag(String tag) { this.tag = tag; return this; }
    /** Accessible-name filter: the text, label, alt, or aria-label a person reads the element by. */
    public SelectorOptions name(String name) { this.name = name; return this; }
    public SelectorOptions role(String role) { this.role = role; return this; }
    public SelectorOptions text(String text) { this.text = text; return this; }
    public SelectorOptions label(String label) { this.label = label; return this; }
    public SelectorOptions placeholder(String placeholder) { this.placeholder = placeholder; return this; }
    public SelectorOptions alt(String alt) { this.alt = alt; return this; }
    public SelectorOptions title(String title) { this.title = title; return this; }
    public SelectorOptions testid(String testid) { this.testid = testid; return this; }
    public SelectorOptions xpath(String xpath) { this.xpath = xpath; return this; }
    public SelectorOptions near(String near) { this.near = near; return this; }
    /**
     * auto (default): page evidence first, a vision model only when needed
     * and configured. cv: local image matching only. vlm: vision model only.
     * tree: the page's DOM/accessibility tree only.
     */
    public SelectorOptions strategy(String strategy) { this.strategy = strategy; return this; }
    /** Include diagnostics in the element's info. */
    public SelectorOptions details(boolean details) { this.details = details; return this; }
    /** Per-call AI overrides, as for check and run. */
    public SelectorOptions provider(String provider) { this.provider = provider; return this; }
    public SelectorOptions model(String model) { this.model = model; return this; }
    public SelectorOptions aiBaseURL(String aiBaseURL) { this.aiBaseURL = aiBaseURL; return this; }
    public SelectorOptions reasoningEffort(String reasoningEffort) { this.reasoningEffort = reasoningEffort; return this; }
    public SelectorOptions timeout(long timeout) { this.timeout = timeout; return this; }

    /** The css selector, or null. It stays the element's own handle so scoped finds see it. */
    public String css() { return css; }

    /** Whether later commands should address the match by what find resolved rather than by this locator. */
    public boolean resolvedHandle() { return description != null || image != null || tag != null || name != null; }

    /**
     * Convert to wire params map. css travels as "selector", the name element
     * commands read.
     */
    public Map<String, Object> toParams() {
        Map<String, Object> params = new LinkedHashMap<>();
        if (description != null) params.put("description", description);
        if (image != null) params.put("image", image);
        if (css != null) params.put("selector", css);
        if (tag != null) params.put("tag", tag);
        if (name != null) params.put("name", name);
        if (role != null) params.put("role", role);
        if (text != null) params.put("text", text);
        if (label != null) params.put("label", label);
        if (placeholder != null) params.put("placeholder", placeholder);
        if (alt != null) params.put("alt", alt);
        if (title != null) params.put("title", title);
        if (testid != null) params.put("testid", testid);
        if (xpath != null) params.put("xpath", xpath);
        if (near != null) params.put("near", near);
        if (strategy != null) params.put("strategy", strategy);
        if (details != null) params.put("details", details);
        if (provider != null) params.put("provider", provider);
        if (model != null) params.put("model", model);
        if (aiBaseURL != null) params.put("aiBaseURL", aiBaseURL);
        if (reasoningEffort != null) params.put("reasoningEffort", reasoningEffort);
        if (timeout != null) params.put("timeout", timeout);
        return params;
    }
}
