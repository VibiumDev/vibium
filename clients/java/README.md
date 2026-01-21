# Vibium Java Client (WIP)

Java client for Vibium's Clicker WebSocket proxy (`clicker serve`). This mirrors the JS/Python clients: it spawns Clicker, connects to `ws://localhost:<port>`, sends standard WebDriver BiDi commands, and uses Vibium's proxy extensions:

- `vibium:find` (waits for selector, returns element info)
- `vibium:click` (selector-based click)
- `vibium:type` (selector-based type)

## Prerequisites

- Java 11+
- Network access (first run only) to download:
  - Clicker (if not already installed / not on `PATH`)
  - Chrome for Testing + chromedriver (if not already installed)

If you already have Clicker available, set:

- `VIBIUM_CLICKER_PATH` (or `CLICKER_PATH`) to the `clicker` binary
- Or ensure `clicker` is on `PATH`

## Dependency (planned)

This is not published to Maven Central yet. When it is, the intended coordinates are:

```xml
<dependency>
  <groupId>com.vibium</groupId>
  <artifactId>vibium-java</artifactId>
  <version>0.1.4</version>
</dependency>
```

## Quick Start

```java
import com.vibium.Browser;
import com.vibium.BrowserAsync;
import com.vibium.Element;
import com.vibium.ElementAsync;
import com.vibium.Vibe;
import com.vibium.VibeAsync;

public class Example {
  public static void main(String[] args) throws Exception {
    try (Vibe vibe = Browser.launch()) {
      vibe.go("https://example.com");
      Element link = vibe.find("a", 10_000);
      System.out.println(link.text());
      link.click(10_000);
      byte[] png = vibe.screenshot();
      java.nio.file.Files.write(java.nio.file.Path.of("screenshot.png"), png);
    }
  }
}
```

Async (CompletableFuture-based):

```java
VibeAsync vibe = BrowserAsync.launch().get();
vibe.go("https://example.com").get();
ElementAsync link = vibe.find("a", 10_000).get();
System.out.println(link.text().get());
vibe.quit().get();
```

## Configuration

```java
import com.vibium.Browser;
import com.vibium.LaunchOptions;
import com.vibium.Vibe;

Vibe vibe = Browser.launch(new LaunchOptions()
  .headless(true)
  .port(9516)
  .timeoutMs(30_000)
  .clickerPath(System.getenv("VIBIUM_CLICKER_PATH"))
);
```

## Environment Variables

- `VIBIUM_CLICKER_PATH`: path to a `clicker` binary (preferred).
- `CLICKER_PATH`: alias supported by JS/Python; also accepted here.
- `VIBIUM_SKIP_CLICKER_DOWNLOAD=1`: disable auto-downloading Clicker from npm.
- `VIBIUM_SKIP_BROWSER_DOWNLOAD=1`: disable auto-installing Chrome for Testing (requires `clicker install` to have been run already).

## Tests

- Unit tests: `mvn test`
- Integration tests (spawns Clicker + Chrome): `mvn verify -DskipITs=false`
- Soak / flake check: `./scripts/soak-it.sh 10` or `powershell -File .\\scripts\\soak-it.ps1 -Iterations 10`

## Notes

- This module targets Clicker's current proxy contract (`clicker serve`), not raw chromedriver/WebDriver.
- If Clicker prints `Server listening on ws://localhost:<port>`, the client extracts the port from stdout.
