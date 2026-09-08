import com.vibium.*;
import com.vibium.types.*;
import java.nio.file.Path;

class CheckSDK {
    static void check(boolean condition) { if (!condition) throw new AssertionError("Check SDK acceptance failed"); }
    public static void main(String[] args) {
        CheckOptions archive = CheckOptions.builder().provider("local").model("archive-override").baseURL(System.getenv("VIBIUM_AI_BASE_URL")).reasoningEffort("").record(Path.of(System.getenv("CHECK_TEST_INPUT"))).build();
        if ("1".equals(System.getenv("CHECK_TEST_ARCHIVE_ONLY"))) {
            check(Vibium.check("archive evidence", archive).status().equals("inconclusive"));
            return;
        }
        Browser bro = Vibium.start(new StartOptions().headless(true));
        try {
            Page page = bro.page();
            page.go(System.getenv("CHECK_TEST_URL"));
            page.evaluate("window.name='original'; sessionStorage.setItem('builder','preserved')");
            Page other = bro.newPage();
            other.go(System.getenv("CHECK_TEST_URL") + "/other");
            page.context().recording().start(new RecordingOptions().video(false).path(System.getenv("CHECK_TEST_OUTPUT")));
            CheckResult result = page.check("changing my display name persists after refresh");
            check(result.status().equals("passed"));
            check(!result.evidence().isEmpty());
            check(page.find("#name").value().equals("Updated"));
            check(page.evaluate("window.name").equals("original"));
            check(page.evaluate("sessionStorage.getItem('builder')").equals("preserved"));
            check(other.find("#name").value().equals("Other"));
            page.context().recording().stop();
            check(bro.check("archive evidence", archive).status().equals("inconclusive"));
        } finally { bro.stop(); }
    }
}
