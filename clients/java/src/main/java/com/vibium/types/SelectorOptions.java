package com.vibium.types;

import java.util.LinkedHashMap;
import java.util.Map;

/**
 * Semantic selector options for finding elements by role, text, label, etc.
 */
public class SelectorOptions {
    private String role;
    private String text;
    private String label;
    private String placeholder;
    private String alt;
    private String title;
    private String testid;
    private String xpath;
    private String near;
    private Long timeout;
    private Integer nth;
    private boolean first;
    private boolean last;
    private String hasText;

    public SelectorOptions role(String role) { this.role = role; return this; }
    public SelectorOptions text(String text) { this.text = text; return this; }
    public SelectorOptions label(String label) { this.label = label; return this; }
    public SelectorOptions placeholder(String placeholder) { this.placeholder = placeholder; return this; }
    public SelectorOptions alt(String alt) { this.alt = alt; return this; }
    public SelectorOptions title(String title) { this.title = title; return this; }
    public SelectorOptions testid(String testid) { this.testid = testid; return this; }
    public SelectorOptions xpath(String xpath) { this.xpath = xpath; return this; }
    public SelectorOptions near(String near) { this.near = near; return this; }
    public SelectorOptions timeout(long timeout) { this.timeout = timeout; return this; }
    /** 0-based index into the matches; negative counts from the end (-1 is the last). */
    public SelectorOptions nth(int nth) { this.nth = nth; return this; }
    /** Pick the first match. Sugar for nth(0). */
    public SelectorOptions first() { this.first = true; return this; }
    /** Pick the last match. Sugar for nth(-1). */
    public SelectorOptions last() { this.last = true; return this; }
    /** Keep only matches whose text content contains this substring. */
    public SelectorOptions hasText(String hasText) { this.hasText = hasText; return this; }

    /**
     * Convert to wire params map.
     */
    public Map<String, Object> toParams() {
        Map<String, Object> params = new LinkedHashMap<>();
        if (role != null) params.put("role", role);
        if (text != null) params.put("text", text);
        if (label != null) params.put("label", label);
        if (placeholder != null) params.put("placeholder", placeholder);
        if (alt != null) params.put("alt", alt);
        if (title != null) params.put("title", title);
        if (testid != null) params.put("testid", testid);
        if (xpath != null) params.put("xpath", xpath);
        if (near != null) params.put("near", near);
        if (timeout != null) params.put("timeout", timeout);
        // first/last are client-side sugar; only the resolved nth hits the wire.
        Integer effectiveNth = nth != null ? nth : first ? 0 : last ? -1 : null;
        if (effectiveNth != null) params.put("nth", effectiveNth);
        if (hasText != null) params.put("hasText", hasText);
        return params;
    }
}
