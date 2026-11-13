// The speech scenario of surfaces.test.js, through the Java client.
import com.google.gson.Gson;
import com.google.gson.JsonElement;
import com.google.gson.JsonNull;
import com.google.gson.JsonObject;
import com.vibium.Browser;
import com.vibium.Page;
import com.vibium.Vibium;
import com.vibium.errors.SpeechException;
import com.vibium.types.ListenOptions;
import com.vibium.types.SayOptions;
import com.vibium.types.SelectorOptions;
import com.vibium.types.StartOptions;

class SpeechSDK {
    static final Gson GSON = new Gson();
    static final String TEXT = "Come here, Watson.";

    static JsonElement failure(Runnable call) {
        try {
            call.run();
        } catch (SpeechException e) {
            JsonObject o = new JsonObject();
            o.addProperty("type", e.getClass().getSimpleName());
            o.addProperty("code", e.code());
            o.addProperty("message", e.getMessage());
            return o;
        }
        return JsonNull.INSTANCE;
    }

    static JsonObject local() {
        JsonObject out = new JsonObject();
        out.add("said", GSON.toJsonTree(Vibium.say(TEXT, SayOptions.builder().target("said.wav").build())));
        out.add("again", GSON.toJsonTree(Vibium.say(TEXT, SayOptions.builder().target("again.wav").build())));
        out.add("fresh", GSON.toJsonTree(Vibium.say(TEXT, SayOptions.builder().target("fresh.wav").cache(false).build())));
        out.add("heard", GSON.toJsonTree(Vibium.listen(ListenOptions.builder().target("said.wav").silence(0).build())));
        out.add("unheard", GSON.toJsonTree(Vibium.listen(ListenOptions.builder().target("said.wav").transcribe(false).path("copy.wav").build())));
        out.add("quiet", failure(() -> Vibium.listen(ListenOptions.builder().target("quiet.wav").build())));
        out.add("allowed", GSON.toJsonTree(Vibium.listen(ListenOptions.builder().target("quiet.wav").allowSilence(true).build())));
        out.add("noSession", failure(() -> Vibium.say(TEXT, SayOptions.builder().target("session").build())));
        return out;
    }

    static JsonObject page() {
        JsonObject out = new JsonObject();
        String channel = System.getenv("VIBIUM_ENGINE_CHANNEL");
        StartOptions start = new StartOptions().headless(true).engine(System.getenv("SPEECH_TEST_ENGINE"));
        if (channel != null && !channel.isEmpty()) start.channel(channel);
        Browser bro = Vibium.start(start);
        try {
            Page page = bro.page();
            page.go(System.getenv("SPEECH_TEST_URL"));
            page.find(new SelectorOptions().css("#probe")).click();
            JsonObject clock = GSON.toJsonTree(page.evaluate(System.getenv("SPEECH_TEST_CLOCK"))).getAsJsonObject();
            out.add("clock", clock);
            if (!(clock.get("advanced").getAsDouble() >= 300)) return out;
            out.add("early", failure(() -> page.say("", SayOptions.builder().file("tone.wav").build())));
            page.find(new SelectorOptions().css("#talk")).click();
            out.add("said", GSON.toJsonTree(page.say("", SayOptions.builder().file("tone.wav").build())));
            out.add("heard", GSON.toJsonTree(bro.listen(ListenOptions.builder().transcribe(false).timeout(9000).path("heard.wav").build())));
        } finally {
            bro.stop();
        }
        return out;
    }

    public static void main(String[] args) {
        System.out.println(GSON.toJson("page".equals(System.getenv("SPEECH_TEST_MODE")) ? page() : local()));
    }
}
