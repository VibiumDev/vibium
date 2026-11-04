package com.vibium.types;

/**
 * Options for find/waitFor operations.
 */
public class FindOptions {
    private Long timeout;
    private Integer nth;
    private boolean first;
    private boolean last;
    private String hasText;

    public FindOptions timeout(long timeout) { this.timeout = timeout; return this; }
    public Long timeout() { return timeout; }

    /** 0-based index into the matches; negative counts from the end (-1 is the last). */
    public FindOptions nth(int nth) { this.nth = nth; return this; }
    /** Pick the first match. Sugar for nth(0). */
    public FindOptions first() { this.first = true; return this; }
    /** Pick the last match. Sugar for nth(-1). */
    public FindOptions last() { this.last = true; return this; }
    /** Keep only matches whose text content contains this substring. */
    public FindOptions hasText(String hasText) { this.hasText = hasText; return this; }
    public String hasText() { return hasText; }

    /** The nth to send on the wire, with first/last resolved. Null when unfiltered. */
    public Integer effectiveNth() {
        if (nth != null) return nth;
        if (first) return 0;
        if (last) return -1;
        return null;
    }
}
