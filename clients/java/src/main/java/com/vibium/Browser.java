package com.vibium;

import com.google.gson.Gson;
import com.google.gson.JsonArray;
import com.google.gson.JsonElement;
import com.google.gson.JsonObject;
import com.vibium.internal.BiDiClient;
import com.vibium.internal.Check;
import com.vibium.internal.Run;
import com.vibium.types.RunResult;
import com.vibium.types.RunOptions;
import com.vibium.types.CheckOptions;
import com.vibium.types.CheckResult;
import com.vibium.internal.VibiumProcess;

import java.util.ArrayList;
import java.util.List;
import java.util.concurrent.CopyOnWriteArrayList;
import java.util.function.Consumer;

/**
 * Manages the browser lifecycle.
 */
public class Browser {

    /** Accomplish a goal through the existing runtime in this live session. */
    public RunResult run(String goal) { return run(goal, new RunOptions()); }
    public RunResult run(String goal, RunOptions options) { return Run.run(client, goal, options, null); }

    /** Verify the existing session with a fresh verifier inference context. */
    public CheckResult check(String claim) { return check(claim, new CheckOptions()); }
    /** Inspect an archive when record is supplied; otherwise use this live session. */
    public CheckResult check(String claim, CheckOptions options) {
        return Check.run(client, claim, options, null);
    }


    private final BiDiClient client;
    private final VibiumProcess process;
    private BrowserContext defaultContext;
    private final CopyOnWriteArrayList<Consumer<Page>> pageListeners = new CopyOnWriteArrayList<>();
    private final CopyOnWriteArrayList<Consumer<Page>> popupListeners = new CopyOnWriteArrayList<>();

    Browser(BiDiClient client, VibiumProcess process) {
        this.client = client;
        this.process = process;

        // Listen for new browsing context events
        client.onEvent(this::handleEvent);
    }

    /**
     * Get the default page.
     */
    public Page page() {
        JsonObject result = client.send("vibium:browser.page", null);
        String contextId = result.get("context").getAsString();

        if (defaultContext == null) {
            String userContext = result.has("userContext") ? result.get("userContext").getAsString() : "default";
            defaultContext = new BrowserContext(client, userContext);
        }

        return new Page(client, contextId, defaultContext);
    }

    /**
     * Create a new page (tab).
     */
    public Page newPage() {
        JsonObject result = client.send("vibium:browser.newPage", null);
        String contextId = result.get("context").getAsString();

        if (defaultContext == null) {
            String userContext = result.has("userContext") ? result.get("userContext").getAsString() : "default";
            defaultContext = new BrowserContext(client, userContext);
        }

        return new Page(client, contextId, defaultContext);
    }

    /**
     * Create a new browser context (isolated cookies/storage).
     */
    public BrowserContext newContext() {
        JsonObject result = client.send("vibium:browser.newContext", null);
        String userContext = result.get("userContext").getAsString();
        return new BrowserContext(client, userContext);
    }

    /**
     * Get the default browser context.
     */
    public BrowserContext context() {
        if (defaultContext == null) {
            // Trigger page() to discover the default context
            page();
        }
        return defaultContext;
    }

    /**
     * List all open pages.
     */
    public List<Page> pages() {
        JsonObject result = client.send("vibium:browser.pages", null);
        JsonArray pagesArr = result.getAsJsonArray("pages");
        List<Page> pages = new ArrayList<>();

        if (defaultContext == null) {
            defaultContext = new BrowserContext(client, "default");
        }

        for (JsonElement el : pagesArr) {
            String contextId = el.getAsJsonObject().get("context").getAsString();
            pages.add(new Page(client, contextId, defaultContext));
        }
        return pages;
    }

    /**
     * Listen for new page events.
     */
    public void onPage(Consumer<Page> callback) {
        pageListeners.add(callback);
    }

    /**
     * Listen for popup events.
     */
    public void onPopup(Consumer<Page> callback) {
        popupListeners.add(callback);
    }

    /**
     * Remove all event listeners, optionally filtered by event name.
     */
    public void removeAllListeners(String event) {
        if (event == null) {
            pageListeners.clear();
            popupListeners.clear();
        } else if ("page".equals(event)) {
            pageListeners.clear();
        } else if ("popup".equals(event)) {
            popupListeners.clear();
        }
    }

    /**
     * Remove all event listeners.
     */
    public void removeAllListeners() {
        removeAllListeners(null);
    }

    /**
     * Stop the browser and clean up.
     */
    public void stop() {
        try {
            client.send("vibium:browser.stop", null);
        } catch (Exception ignored) {
            // Best-effort
        }
        client.close();
        process.stop();
    }

    BiDiClient getClient() {
        return client;
    }

    private void handleEvent(JsonObject event) {
        String method = event.has("method") ? event.get("method").getAsString() : "";
        if ("browsingContext.contextCreated".equals(method)) {
            JsonObject params = event.getAsJsonObject("params");
            if (params != null && params.has("context")) {
                String contextId = params.get("context").getAsString();

                if (defaultContext == null) {
                    defaultContext = new BrowserContext(client, "default");
                }

                Page newPage = new Page(client, contextId, defaultContext);

                for (Consumer<Page> listener : pageListeners) {
                    try {
                        listener.accept(newPage);
                    } catch (Exception ignored) {}
                }

                // Popups have an opener
                if (params.has("opener") && !params.get("opener").isJsonNull()) {
                    for (Consumer<Page> listener : popupListeners) {
                        try {
                            listener.accept(newPage);
                        } catch (Exception ignored) {}
                    }
                }
            }
        }
    }
}
