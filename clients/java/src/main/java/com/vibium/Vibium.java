package com.vibium;

import com.vibium.internal.BiDiClient;
import com.vibium.internal.BinaryResolver;
import com.vibium.internal.VibiumProcess;
import com.vibium.types.StartOptions;

import java.util.LinkedHashMap;
import java.util.Map;

/**
 * Entry point for the Vibium browser automation library.
 *
 * <pre>{@code
 * Browser bro = Vibium.start();
 * Page vibe = bro.page();
 * vibe.go("https://example.com");
 * System.out.println(vibe.title());
 * bro.stop();
 * }</pre>
 */
public final class Vibium {

    private Vibium() {}



    /** Inspect a saved recording without installing or starting a browser. */
    public static com.vibium.types.CheckResult check(String claim, com.vibium.types.CheckOptions options) {
        if (options == null || options.record() == null) throw new IllegalArgumentException("Standalone verification requires record");
        VibiumProcess process = VibiumProcess.startWithoutBrowser(BinaryResolver.resolve());
        BiDiClient client = null;
        try {
            client = BiDiClient.fromProcess(process);
            return com.vibium.internal.Check.run(client, claim, options, null);
        } finally {
            try { if (client != null) client.close(); } finally { process.stop(); }
        }
    }


    /** Speak with no browser: into this machine's speakers, or a .wav file. */
    public static com.vibium.types.SayResult say(String text) {
        return say(text, new com.vibium.types.SayOptions());
    }

    public static com.vibium.types.SayResult say(String text, com.vibium.types.SayOptions options) {
        return com.vibium.internal.Speech.say(text, options);
    }

    /** Listen with no browser: to this machine's microphone, or a .wav file. */
    public static com.vibium.types.ListenResult listen() {
        return listen(new com.vibium.types.ListenOptions());
    }

    public static com.vibium.types.ListenResult listen(com.vibium.types.ListenOptions options) {
        return com.vibium.internal.Speech.listen(options);
    }

    /**
     * Start a visible browser.
     */
    public static Browser start() {
        return start(new StartOptions());
    }

    /**
     * Start a browser with options.
     */
    public static Browser start(StartOptions options) {
        String binaryPath;
        if (options.executablePath() != null) {
            binaryPath = options.executablePath();
        } else {
            binaryPath = BinaryResolver.resolve();
        }

        // Env fallbacks, matching the JS and Python clients: the connect URL
        // from VIBIUM_CONNECT_URL, then — only when connecting remotely — a
        // Bearer header from VIBIUM_CONNECT_API_KEY and capabilities from
        // VIBIUM_CONNECT_CAPS. Explicit options always win.
        String connectURL = options.connectURL();
        if (connectURL == null || connectURL.isEmpty()) {
            connectURL = System.getenv("VIBIUM_CONNECT_URL");
        }
        Map<String, String> connectHeaders = options.connectHeaders();
        String connectCaps = options.connectCaps();
        if (connectURL != null && !connectURL.isEmpty()) {
            String apiKey = System.getenv("VIBIUM_CONNECT_API_KEY");
            if (apiKey != null && !apiKey.isEmpty()) {
                Map<String, String> merged = new LinkedHashMap<>();
                merged.put("Authorization", "Bearer " + apiKey);
                if (connectHeaders != null) {
                    merged.putAll(connectHeaders);
                }
                connectHeaders = merged;
            }
            if (connectCaps == null || connectCaps.isEmpty()) {
                connectCaps = System.getenv("VIBIUM_CONNECT_CAPS");
            }
        }

        VibiumProcess process = VibiumProcess.start(
            binaryPath,
            options.engine(),
            options.channel(),
            options.headless(),
            options.userDataDir(),
            connectURL,
            connectHeaders,
            connectCaps
        );

        BiDiClient client = BiDiClient.fromProcess(process);

        return new Browser(client, process);
    }
}
