package com.vibium.types;

import java.util.Map;

/**
 * Options for starting a browser.
 */
public class StartOptions {
    private String engine;
    private String channel;
    private boolean headless;
    private String executablePath;
    private String connectURL;
    private Map<String, String> connectHeaders;
    private String connectCaps;

    /** Browser engine to launch: "chrome" (default) or "firefox". */
    public StartOptions engine(String engine) { this.engine = engine; return this; }
    /** Browser release channel, such as "beta". Currently honored by Firefox only. */
    public StartOptions channel(String channel) { this.channel = channel; return this; }
    public StartOptions headless(boolean headless) { this.headless = headless; return this; }
    public StartOptions executablePath(String path) { this.executablePath = path; return this; }
    public StartOptions connectURL(String url) { this.connectURL = url; return this; }
    public StartOptions connectHeaders(Map<String, String> headers) { this.connectHeaders = headers; return this; }
    /** JSON object of extra alwaysMatch capabilities for classic WebDriver endpoints
     *  (cloud grids take their config this way, via vendor-prefixed capability keys). */
    public StartOptions connectCaps(String capsJson) { this.connectCaps = capsJson; return this; }

    public String engine() { return engine; }
    public String channel() { return channel; }
    public boolean headless() { return headless; }
    public String executablePath() { return executablePath; }
    public String connectURL() { return connectURL; }
    public Map<String, String> connectHeaders() { return connectHeaders; }
    public String connectCaps() { return connectCaps; }
}
