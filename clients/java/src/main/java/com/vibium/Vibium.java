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
            options.headless(),
            options.connectURL(),
            options.connectHeaders()
        );

        BiDiClient client = BiDiClient.fromProcess(process);

        return new Browser(client, process);
    }
}
