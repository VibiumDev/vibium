package com.vibium;

import com.vibium.internal.BiDiClient;
import com.vibium.internal.BinaryResolver;
import com.vibium.internal.VibiumProcess;
import com.vibium.types.StartOptions;

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
    public static com.vibium.types.VerificationResult verify(String claim, com.vibium.types.VerifyOptions options) {
        if (options == null || options.record() == null) throw new IllegalArgumentException("Standalone verification requires record");
        VibiumProcess process = VibiumProcess.startWithoutBrowser(BinaryResolver.resolve());
        BiDiClient client = null;
        try {
            client = BiDiClient.fromProcess(process);
            return com.vibium.internal.Verification.run(client, claim, options, null);
        } finally {
            try { if (client != null) client.close(); } finally { process.stop(); }
        }
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

        VibiumProcess process = VibiumProcess.start(
            binaryPath,
            options.engine(),
            options.channel(),
            options.headless(),
            options.connectURL(),
            options.connectHeaders()
        );

        BiDiClient client = BiDiClient.fromProcess(process);

        return new Browser(client, process);
    }
}
