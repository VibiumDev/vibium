package com.vibium;

import java.util.concurrent.CompletableFuture;

/**
 * Async browser launcher (CompletableFuture-based), mirroring JS/Python async clients.
 */
public final class BrowserAsync {
    private BrowserAsync() {}

    public static CompletableFuture<VibeAsync> launch() {
        return launch(new LaunchOptions());
    }

    public static CompletableFuture<VibeAsync> launch(LaunchOptions options) {
        LaunchOptions resolved = options == null ? new LaunchOptions() : options;
        return CompletableFuture.supplyAsync(() -> Browser.launch(resolved), AsyncSupport.executor())
                .thenApply(VibeAsync::new);
    }
}
