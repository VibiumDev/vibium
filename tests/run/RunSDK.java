import com.vibium.*;
import com.vibium.types.*;

class RunSDK {
    static void check(boolean condition) { if (!condition) throw new AssertionError("Run SDK acceptance failed"); }
    public static void main(String[] args) {
        Browser bro = Vibium.start(new StartOptions().headless(true));
        try {
            Page page = bro.page();
            page.go(System.getenv("RUN_TEST_URL"));
            page.evaluate("window.name='original';sessionStorage.setItem('builder','preserved')");
            Page other = bro.newPage(); other.go(System.getenv("RUN_TEST_URL") + "/other");
            page.context().recording().start(new RecordingOptions().video(false).path(System.getenv("RUN_TEST_OUTPUT")));
            RunResult result = page.run("change name");
            check(result.status().equals("completed") && result.goal().equals("change name") && !result.evidence().isEmpty());
            check(page.find("#name").value().equals("Updated"));
            check(page.evaluate("window.name").equals("original"));
            check(page.evaluate("sessionStorage.getItem('builder')").equals("preserved"));
            check(other.find("#name").value().equals("Other"));
            check(page.check("the name persisted").status().equals("passed"));
            String provider = System.getenv("VIBIUM_AI_PROVIDER").equals("anthropic") ? "google" : "anthropic";
            String endpoint = System.getenv("VIBIUM_AI_BASE_URL");
            check(page.run("change name", RunOptions.builder().provider(provider).model("run-model").baseURL(endpoint).reasoningEffort("").build()).status().equals("completed"));
            check(page.check("the name persisted", CheckOptions.builder().provider(provider).model("check-model").baseURL(endpoint).reasoningEffort("").build()).status().equals("passed"));
            check(bro.run("not possible").status().equals("not_completed"));
            page.context().recording().stop();
        } finally { bro.stop(); }
    }
}
