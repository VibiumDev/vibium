package com.vibium;

import com.google.gson.JsonObject;
import com.vibium.internal.BiDiClient;
import com.vibium.types.ClockOptions;

import java.util.Map;

/**
 * Fake timer / Date control.
 */
public class Clock {

    private final BiDiClient client;
    private final String contextId;

    Clock(BiDiClient client, String contextId) {
        this.client = client;
        this.contextId = contextId;
    }

    /** Install fake timers. */
    public void install() {
        install(null);
    }

    /** Install fake timers with options. */
    public void install(ClockOptions options) {
        JsonObject params = params();
        if (options != null) {
            for (Map.Entry<String, Object> entry : options.toParams().entrySet()) {
                if ("time".equals(entry.getKey())) {
                    // The engine expects epoch milliseconds as a number, not a
                    // string — sending a string left the option silently ignored
                    // (issue #137).
                    params.addProperty("time", normalizeTime((String) entry.getValue()));
                } else {
                    params.addProperty(entry.getKey(), (String) entry.getValue());
                }
            }
        }
        client.send("vibium:clock.install", params);
    }

    /** Fast-forward time by milliseconds. */
    public void fastForward(long ticks) {
        JsonObject params = params();
        params.addProperty("ticks", ticks);
        client.send("vibium:clock.fastForward", params);
    }

    /** Run timers for a duration in milliseconds. */
    public void runFor(long ticks) {
        JsonObject params = params();
        params.addProperty("ticks", ticks);
        client.send("vibium:clock.runFor", params);
    }

    /** Pause the clock at a specific time. */
    public void pauseAt(String time) {
        JsonObject params = params();
        params.addProperty("time", normalizeTime(time));
        client.send("vibium:clock.pauseAt", params);
    }

    /** Resume the clock. */
    public void resume() {
        client.send("vibium:clock.resume", params());
    }

    /** Set fixed fake time. */
    public void setFixedTime(String time) {
        JsonObject params = params();
        params.addProperty("time", normalizeTime(time));
        client.send("vibium:clock.setFixedTime", params);
    }

    /** Set system time. */
    public void setSystemTime(String time) {
        JsonObject params = params();
        params.addProperty("time", normalizeTime(time));
        client.send("vibium:clock.setSystemTime", params);
    }

    /**
     * Convert a time string to epoch milliseconds. The engine reads the clock
     * time as a number; sending the raw string made every call fail with
     * "time is required" (issue #137). Accepts epoch-ms strings and ISO-8601
     * instants, date-times, and dates (interpreted as UTC).
     */
    static long normalizeTime(String time) {
        if (time == null) {
            throw new IllegalArgumentException("time is required");
        }
        String t = time.trim();
        try {
            return Long.parseLong(t);
        } catch (NumberFormatException ignored) {
            // not epoch milliseconds
        }
        try {
            return java.time.Instant.parse(t).toEpochMilli();
        } catch (Exception ignored) {
            // not an ISO instant
        }
        try {
            return java.time.LocalDateTime.parse(t).toInstant(java.time.ZoneOffset.UTC).toEpochMilli();
        } catch (Exception ignored) {
            // not an ISO date-time
        }
        try {
            return java.time.LocalDate.parse(t).atStartOfDay(java.time.ZoneOffset.UTC).toInstant().toEpochMilli();
        } catch (Exception ignored) {
            // not an ISO date
        }
        throw new IllegalArgumentException(
            "Unrecognized time format: " + time + " (use epoch milliseconds or an ISO-8601 string)");
    }

    /** Set timezone. */
    public void setTimezone(String timezone) {
        JsonObject params = params();
        params.addProperty("timezone", timezone);
        client.send("vibium:clock.setTimezone", params);
    }

    private JsonObject params() {
        JsonObject p = new JsonObject();
        p.addProperty("context", contextId);
        return p;
    }
}
