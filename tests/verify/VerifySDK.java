import com.vibium.*;
import com.vibium.types.*;
import java.nio.file.Path;

class VerifySDK {
    static void check(boolean condition) { if (!condition) throw new AssertionError("Verify SDK acceptance failed"); }
    public static void main(String[] args) {
        VerifyOptions archive = VerifyOptions.builder().record(Path.of(System.getenv("VERIFY_INPUT"))).build();
        if ("1".equals(System.getenv("VERIFY_ARCHIVE_ONLY"))) {
            check(Vibium.verify("archive evidence", archive).status().equals("inconclusive"));
            return;
        }
        Browser bro = Vibium.start(new StartOptions().headless(true));
        try {
            Page page = bro.page();
            page.go(System.getenv("VERIFY_URL"));
            page.evaluate("window.name='original'; sessionStorage.setItem('builder','preserved')");
            Page other = bro.newPage();
            other.go(System.getenv("VERIFY_URL") + "/other");
            page.context().recording().start(new RecordingOptions().video(false).path(System.getenv("VERIFY_OUTPUT")));
            VerificationResult result = page.verify("changing my display name persists after refresh");
            check(result.status().equals("passed"));
            check(!result.evidence().isEmpty());
            check(page.find("#name").value().equals("Updated"));
            check(page.evaluate("window.name").equals("original"));
            check(page.evaluate("sessionStorage.getItem('builder')").equals("preserved"));
            check(other.find("#name").value().equals("Other"));
            page.context().recording().stop();
            check(bro.verify("archive evidence", archive).status().equals("inconclusive"));
        } finally { bro.stop(); }
    }
}
