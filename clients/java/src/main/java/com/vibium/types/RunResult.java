package com.vibium.types;

import java.util.List;

/** Run outcome. Operational failures throw instead of returning a result. */
public final class RunResult {
    private String status;
    private String goal;
    private String summary;
    private List<Evidence> evidence;
    public String status() { return status; }
    public String goal() { return goal; }
    public String summary() { return summary; }
    public List<Evidence> evidence() { return List.copyOf(evidence); }
    public static final class Evidence {
        private String type;
        private String summary;
        public String type() { return type; }
        public String summary() { return summary; }
    }
}
