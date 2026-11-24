package com.vibium;

import com.google.gson.Gson;
import com.google.gson.JsonObject;
import com.vibium.errors.VibiumConnectionException;
import com.vibium.errors.VibiumException;
import com.vibium.internal.BiDiClient;
import com.vibium.types.ContinueOptions;
import com.vibium.types.FulfillOptions;

import java.util.Map;

/**
 * Intercepted network request route handler.
 */
public class Route {

    private static final Gson GSON = new Gson();

    private final BiDiClient client;
    private final String contextId;
    private final String requestId;
    private final Request request;

    Route(BiDiClient client, String contextId, String requestId, Request request) {
        this.client = client;
        this.contextId = contextId;
        this.requestId = requestId;
        this.request = request;
    }

    /** Get the intercepted request. */
    public Request request() {
        return request;
    }

    /** Fulfill the request with a custom response. */
    public void fulfill() {
        fulfill(null);
    }

    /** Fulfill the request with options. */
    public void fulfill(FulfillOptions options) {
        try {
            JsonObject params = new JsonObject();
            params.addProperty("request", requestId);
            if (options != null) {
                for (Map.Entry<String, Object> entry : options.toParams().entrySet()) {
                    params.add(entry.getKey(), GSON.toJsonTree(entry.getValue()));
                }
            }
            client.send("vibium:network.fulfill", params);
        } catch (VibiumException e) {
            if (!isRaceError(e)) throw e;
        }
    }

    /**
     * Continue the request (optionally with overrides).
     * Named doContinue() because "continue" is a Java reserved word.
     */
    public void doContinue() {
        doContinue(null);
    }

    /**
     * Continue the request with overrides.
     */
    public void doContinue(ContinueOptions options) {
        try {
            JsonObject params = new JsonObject();
            params.addProperty("request", requestId);
            if (options != null) {
                for (Map.Entry<String, Object> entry : options.toParams().entrySet()) {
                    params.add(entry.getKey(), GSON.toJsonTree(entry.getValue()));
                }
            }
            client.send("vibium:network.continue", params);
        } catch (VibiumException e) {
            if (!isRaceError(e)) throw e;
        }
    }

    /** Abort the request. */
    public void abort() {
        try {
            JsonObject params = new JsonObject();
            params.addProperty("request", requestId);
            client.send("vibium:network.abort", params);
        } catch (VibiumException e) {
            if (!isRaceError(e)) throw e;
        }
    }

    /** A request that was already handled, or a connection that is gone, is not a failure. */
    private static boolean isRaceError(VibiumException e) {
        String message = String.valueOf(e.getMessage());
        return e instanceof VibiumConnectionException
            || message.contains("Invalid state")
            || message.contains("No blocked request")
            || message.contains("no such request");
    }
}
