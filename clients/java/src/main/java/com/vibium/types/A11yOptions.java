package com.vibium.types;

import java.util.LinkedHashMap;
import java.util.Map;

/**
 * Options for getting the accessibility tree.
 */
public class A11yOptions {
    private Boolean everything;
    private String root;

    /** Include every node, not just the interesting ones. Defaults to false. */
    public A11yOptions everything(boolean everything) { this.everything = everything; return this; }
    public A11yOptions root(String root) { this.root = root; return this; }

    public Map<String, Object> toParams() {
        Map<String, Object> params = new LinkedHashMap<>();
        if (everything != null) params.put("everything", everything);
        if (root != null) params.put("root", root);
        return params;
    }
}
