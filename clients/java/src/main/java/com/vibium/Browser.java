package com.vibium;

import com.vibium.bidi.BiDiClient;
import com.vibium.process.ClickerProcess;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

public final class Browser {
    private static final Logger log = LoggerFactory.getLogger(Browser.class);

    private Browser() {}

    public static Vibe launch() {
        return launch(new LaunchOptions());
    }

    public static Vibe launch(LaunchOptions options) {
        LaunchOptions resolved = options == null ? new LaunchOptions() : options;
        log.info("Launching Vibium (headless={}, port={})", resolved.isHeadless(), resolved.getPort() == null ? "auto" : resolved.getPort());

        ClickerProcess process = ClickerProcess.start(resolved);
        BiDiClient client = null;
        try {
            String wsUrl = String.format("ws://localhost:%d", process.getPort());
            log.debug("Clicker listening at {}", wsUrl);
            client = BiDiClient.connect(wsUrl, resolved.getTimeoutMs());
            return new Vibe(client, process);
        } catch (Exception e) {
            if (client != null) {
                try {
                    client.close();
                } catch (Exception ignored) {
                }
            }
            try {
                process.stop();
            } catch (Exception ignored) {
            }
            throw new VibiumException("Failed to launch Vibium", e);
        }
    }
}
