package com.vibium.types;

import java.util.List;

/** Completed verification. Operational failures throw instead of returning a verdict. */
public final class VerificationResult {
    private String status;
    private String claim;
    private String summary;
    private List<Evidence> evidence;
    public String status() { return status; }
    public String claim() { return claim; }
    public String summary() { return summary; }
    public List<Evidence> evidence() { return List.copyOf(evidence); }
    public static final class Evidence {
        private String type;
        private String summary;
        public String type() { return type; }
        public String summary() { return summary; }
    }
}
