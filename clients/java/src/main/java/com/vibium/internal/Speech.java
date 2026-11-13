package com.vibium.internal;

import com.google.gson.Gson;
import com.google.gson.JsonObject;
import com.vibium.errors.SpeechException;
import com.vibium.types.ListenOptions;
import com.vibium.types.ListenResult;
import com.vibium.types.SayOptions;
import com.vibium.types.SayResult;
import com.vibium.types.SpeechOptions;

import java.nio.file.Path;
import java.util.Locale;

/** Say and listen; rendering, playback and capture all run in Go. */
public final class Speech {
    private Speech() {}

    static final long SAY_TIMEOUT_MS = 180_000;
    private static final int LISTEN_TIMEOUT_MS = 10_000;
    private static final int LISTEN_DURATION_MS = 30_000;

    // One engine with no browser serves every Vibium.say and Vibium.listen,
    // so the capture window a say opens is still there for the next listen.
    private static VibiumProcess engineProcess;
    private static BiDiClient engineClient;

    public static SayResult say(BiDiClient client, String text, SayOptions options, String context) {
        return new Gson().fromJson(call(client, "vibium:speech.say", sayParams(text, options, context), SAY_TIMEOUT_MS), SayResult.class);
    }

    public static ListenResult listen(BiDiClient client, ListenOptions options, String context) {
        return new Gson().fromJson(call(client, "vibium:speech.listen", listenParams(options, context), listenTimeoutMs(options)), ListenResult.class);
    }

    /** Say with no browser, on the shared engine. */
    public static SayResult say(String text, SayOptions options) {
        return say(engine(), text, options, null);
    }

    /** Listen with no browser, on the shared engine. */
    public static ListenResult listen(ListenOptions options) {
        return listen(engine(), options, null);
    }

    private static synchronized BiDiClient engine() {
        if (engineProcess == null || !engineProcess.isAlive()) {
            engineProcess = VibiumProcess.startWithoutBrowser(BinaryResolver.resolve());
            engineClient = BiDiClient.fromProcess(engineProcess);
        }
        return engineClient;
    }

    /** A listen may wait its whole timeout, capture its whole duration, and then transcribe. */
    static long listenTimeoutMs(ListenOptions o) {
        int wait = o != null && o.timeout() != null ? o.timeout() : LISTEN_TIMEOUT_MS;
        int capture = o != null && o.duration() != null ? o.duration() : LISTEN_DURATION_MS;
        return (long) wait + capture + 60_000;
    }

    static JsonObject sayParams(String text, SayOptions o, String context) {
        JsonObject params = new JsonObject();
        if (text != null && !text.isEmpty()) params.addProperty("text", text);
        if (o != null) {
            shared(params, o);
            if (o.file() != null) params.addProperty("file", absolute(o.file()));
            if (o.voice() != null) params.addProperty("voice", o.voice());
            if (o.speed() != null) params.addProperty("speed", o.speed());
            if (o.cache() != null) params.addProperty("cache", o.cache());
            if (o.cacheDir() != null) params.addProperty("cacheDir", absolute(o.cacheDir()));
        }
        if (context != null) params.addProperty("context", context);
        return params;
    }

    static JsonObject listenParams(ListenOptions o, String context) {
        JsonObject params = new JsonObject();
        if (o != null) {
            shared(params, o);
            if (o.silence() != null) params.addProperty("silence", o.silence());
            if (o.timeout() != null) params.addProperty("timeout", o.timeout());
            if (o.duration() != null) params.addProperty("duration", o.duration());
            if (o.allowSilence() != null) params.addProperty("allowSilence", o.allowSilence());
            if (o.path() != null) params.addProperty("path", absolute(o.path()));
            if (o.transcribe() != null) params.addProperty("transcribe", o.transcribe());
        }
        if (context != null) params.addProperty("context", context);
        return params;
    }

    private static void shared(JsonObject params, SpeechOptions<?> o) {
        if (o.target() != null) {
            boolean file = o.target().toLowerCase(Locale.ROOT).endsWith(".wav");
            params.addProperty("target", file ? absolute(o.target()) : o.target());
        }
        if (o.provider() != null) params.addProperty("provider", o.provider());
        if (o.model() != null) params.addProperty("model", o.model());
        if (o.language() != null) params.addProperty("language", o.language());
    }

    // The engine may run in another directory.
    private static String absolute(String path) {
        return path.isEmpty() ? path : Path.of(path).toAbsolutePath().normalize().toString();
    }

    private static JsonObject call(BiDiClient client, String method, JsonObject params, long timeoutMs) {
        try {
            return client.send(method, params, timeoutMs);
        } catch (CodedException e) {
            // The engine answers a speech failure with its code.
            throw BiDiClient.attachCallerStack(new SpeechException(e.code, e.getMessage()), e.getStackTrace());
        }
    }
}
