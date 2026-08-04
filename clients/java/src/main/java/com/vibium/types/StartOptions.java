package com.vibium.types;

import java.util.Map;

/**
 * Options for starting a browser.
 */
public class StartOptions {
    private String engine;
    private boolean headless;
    private String executablePath;
    private String connectURL;
    private Map<String, String> connectHeaders;

    /** Browser engine to launch: "chrome" (default) or "firefox". */
    public StartOptions engine(String engine) { this.engine = engine; return this; }
    public StartOptions headless(boolean headless) { this.headless = headless; return this; }
    public StartOptions executablePath(String path) { this.executablePath = path; return this; }
    public StartOptions connectURL(String url) { this.connectURL = url; return this; }
    public StartOptions connectHeaders(Map<String, String> headers) { this.connectHeaders = headers; return this; }

    public String engine() { return engine; }
    public boolean headless() { return headless; }
    public String executablePath() { return executablePath; }
    public String connectURL() { return connectURL; }
    public Map<String, String> connectHeaders() { return connectHeaders; }
}
