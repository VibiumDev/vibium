package com.vibium.internal;

import com.google.gson.JsonObject;
import com.vibium.errors.SpeechException;
import com.vibium.errors.VibiumException;
import com.vibium.types.ListenOptions;
import com.vibium.types.SayOptions;
import org.junit.jupiter.api.Test;

import java.nio.file.Path;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertFalse;
import static org.junit.jupiter.api.Assertions.assertTrue;

/** What the Java client sends for say and listen. No engine or audio is needed. */
class SpeechTest {

    private static String abs(String path) {
        return Path.of(path).toAbsolutePath().normalize().toString();
    }

    @Test
    void sayGoesOutWithItsOptionsAndThePage() {
        JsonObject params = Speech.sayParams("What time is it?",
            SayOptions.builder().voice("cedar").speed(1.5).cache(false).build(), "page-7");
        JsonObject want = new JsonObject();
        want.addProperty("text", "What time is it?");
        want.addProperty("voice", "cedar");
        want.addProperty("speed", 1.5);
        want.addProperty("cache", false);
        want.addProperty("context", "page-7");
        assertEquals(want, params);
    }

    @Test
    void aBrowserNamesNoPage() {
        JsonObject want = new JsonObject();
        want.addProperty("text", "Hello");
        assertEquals(want, Speech.sayParams("Hello", new SayOptions(), null));
        assertEquals(new JsonObject(), Speech.listenParams(new ListenOptions(), null));
        assertEquals(new JsonObject(), Speech.listenParams(null, null));
    }

    @Test
    void listenGoesOutWithItsOptions() {
        JsonObject params = Speech.listenParams(
            ListenOptions.builder().silence(0).allowSilence(true).transcribe(false).build(), "page-7");
        JsonObject want = new JsonObject();
        want.addProperty("silence", 0);
        want.addProperty("allowSilence", true);
        want.addProperty("transcribe", false);
        want.addProperty("context", "page-7");
        assertEquals(want, params);
    }

    @Test
    void pathsGoToTheEngineAbsoluteAndNamedTargetsAsTheyAre() {
        JsonObject say = Speech.sayParams("",
            SayOptions.builder().file("clip.wav").target("out/said.WAV").cacheDir("cache").build(), null);
        assertFalse(say.has("text"));
        assertEquals(abs("clip.wav"), say.get("file").getAsString());
        assertEquals(abs("out/said.WAV"), say.get("target").getAsString());
        assertEquals(abs("cache"), say.get("cacheDir").getAsString());

        JsonObject listen = Speech.listenParams(ListenOptions.builder().target("reply.wav").path("heard.wav").build(), null);
        assertEquals(abs("reply.wav"), listen.get("target").getAsString());
        assertEquals(abs("heard.wav"), listen.get("path").getAsString());

        for (String target : new String[] {"session", "local", "speakers"}) {
            assertEquals(target, Speech.listenParams(ListenOptions.builder().target(target).build(), null).get("target").getAsString());
        }
    }

    @Test
    void theClientOutwaitsAListen() {
        assertEquals(100_000, Speech.listenTimeoutMs(new ListenOptions()));
        assertEquals(100_000, Speech.listenTimeoutMs(null));
        assertEquals(85_000, Speech.listenTimeoutMs(ListenOptions.builder().timeout(5000).duration(20000).build()));
        assertEquals(180_000, Speech.SAY_TIMEOUT_MS);
    }

    @Test
    void aFailureWithACodeOfItsOwnKeepsTheCode() {
        VibiumException coded = BiDiClient.mapError("no_speech", "listened for 10.0 s and heard no speech");
        assertEquals(CodedException.class, coded.getClass());
        assertEquals("no_speech", ((CodedException) coded).code);
        assertEquals("listened for 10.0 s and heard no speech", coded.getMessage());

        assertEquals(VibiumException.class, BiDiClient.mapError("error", "unsupported argument pitch").getClass());
    }

    @Test
    void aSpeechExceptionShowsItsCodeWhenPrinted() {
        SpeechException e = new SpeechException("no_speech", "listened for 10.0 s and heard no speech");
        assertEquals("no_speech", e.code());
        assertEquals("listened for 10.0 s and heard no speech", e.getMessage());
        assertTrue(e.toString().endsWith("SpeechException: no_speech: listened for 10.0 s and heard no speech"));
        assertTrue(e instanceof VibiumException);
    }
}
