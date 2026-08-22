# Getting Started with Vibium (Java)

A complete beginner's guide. No prior experience required.

---

## What You'll Build

A program that opens a browser, visits a website, takes a screenshot, and clicks a link. All in about 10 lines of code.

---

## Step 1: Install Java

Vibium requires Java 11 or higher (we recommend the latest LTS). Check if you have it:

```bash
java --version
```

If you see a version number (`11.0.0` or higher), skip to Step 2.

### macOS

```bash
# Install Homebrew (if you don't have it)
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"

# Install Java
brew install openjdk
```

### Windows

```powershell
winget install EclipseAdoptium.Temurin.21.JDK
```

### Linux

```bash
# Ubuntu/Debian
sudo apt-get install openjdk-21-jdk

# Or use your distro's package manager
```

---

## Step 2: Create a Project Folder

### Gradle (recommended)

```bash
mkdir my-first-bot
cd my-first-bot
```

Create a `build.gradle` file:

```groovy
plugins {
    id 'application'
}

repositories {
    mavenCentral()
}

dependencies {
    implementation 'com.vibium:vibium:26.3.18'
}

application {
    mainClass = 'Hello'
}
```

### Maven

```bash
mkdir my-first-bot
cd my-first-bot
```

Create a `pom.xml` file:

```xml
<project>
    <modelVersion>4.0.0</modelVersion>
    <groupId>com.example</groupId>
    <artifactId>my-first-bot</artifactId>
    <version>1.0</version>

    <dependencies>
        <dependency>
            <groupId>com.vibium</groupId>
            <artifactId>vibium</artifactId>
            <version>26.3.18</version>
        </dependency>
    </dependencies>
</project>
```

### No build tool

Download the JAR directly and compile with `javac`:

```bash
mkdir my-first-bot
cd my-first-bot

# Download the vibium JAR and its dependency (Gson)
curl -LO https://repo1.maven.org/maven2/com/vibium/vibium/26.3.18/vibium-26.3.18.jar
curl -LO https://repo1.maven.org/maven2/com/google/code/gson/gson/2.11.0/gson-2.11.0.jar
```

Chrome for Testing and chromedriver download automatically on first `Vibium.start()`. Everything is cached in a platform-specific directory:

| Platform | Cache path |
|----------|------------|
| Linux | `~/.cache/vibium/` |
| macOS | `~/Library/Caches/vibium/` |
| Windows | `%LOCALAPPDATA%\vibium\` |

---

## Step 3: Write Your First Program

Create a file called `Hello.java`:

```java
import com.vibium.Vibium;
import java.nio.file.Files;
import java.nio.file.Path;

public class Hello {
    public static void main(String[] args) throws Exception {
        // Launch a browser (you'll see it open!)
        var bro = Vibium.start();
        var vibe = bro.page();

        // Go to a website
        vibe.go("https://example.com");
        System.out.println("Loaded example.com");

        // Take a screenshot
        var png = vibe.screenshot();
        Files.write(Path.of("screenshot.png"), png);
        System.out.println("Saved screenshot.png");

        // Find and click the link
        var link = vibe.find("a");
        System.out.println("Found link: " + link.text());
        link.click();
        System.out.println("Clicked!");

        // Close the browser
        bro.stop();
        System.out.println("Done!");
    }
}
```

---

## Step 4: Run It

### Gradle

```bash
# Place Hello.java in src/main/java/
mkdir -p src/main/java
mv Hello.java src/main/java/

gradle run
```

### Maven

```bash
# Place Hello.java in src/main/java/
mkdir -p src/main/java
mv Hello.java src/main/java/

mvn compile exec:java -Dexec.mainClass=Hello
```

### No build tool

```bash
javac -cp "vibium-26.3.18.jar:gson-2.11.0.jar" Hello.java
java -cp ".:vibium-26.3.18.jar:gson-2.11.0.jar" Hello
```

You should see:
1. A Chrome window open
2. example.com load
3. The browser click "More information..."
4. The browser close

Check your folder - there's now a `screenshot.png` file!

---

## What Just Happened?

| Line | What It Does |
|------|--------------|
| `Vibium.start()` | Opens Chrome, returns a Browser |
| `bro.page()` | Gets the default page (tab) |
| `vibe.go(url)` | Navigates to a URL |
| `vibe.screenshot()` | Captures the page as PNG bytes |
| `vibe.find(selector)` | Finds an element by CSS selector |
| `link.text()` | Gets the element's text content |
| `link.click()` | Clicks the element |
| `bro.stop()` | Closes the browser |

---

## Next Steps

**Hide the browser** (run headless):
```java
import com.vibium.types.StartOptions;

var bro = Vibium.start(new StartOptions().headless(true));
```

**Use JavaScript instead:**
See [Getting Started (JavaScript)](getting-started-js.md) for the JS version.

**Use Python instead:**
See [Getting Started (Python)](getting-started-python.md) for the Python version.

**Let AI control the browser:**
See [Agent Setup](../../README.md#agent-setup) for CLI setup and [Getting Started with MCP](getting-started-mcp.md) for MCP server setup.

---

## Troubleshooting

### "command not found: java"

Java isn't installed or isn't in your PATH. See [Step 1](#step-1-install-java) above.

### "chromedriver not found" or "Chrome not found"

Chrome for Testing downloads automatically on first use. If auto-install fails (e.g. behind a corporate proxy), see [Advanced: Manual Browser Install](#advanced-manual-browser-install) below.

### "package com.vibium does not exist"

Make sure the vibium JAR is on your classpath. If using Gradle/Maven, run the build command to download dependencies first.

### Browser doesn't open

Try running with headless mode disabled (it's disabled by default, but just in case):
```java
Browser bro = Vibium.start(new StartOptions().headless(false));
```

### Permission denied (Linux)

You might need to install dependencies for Chrome:
```bash
sudo apt-get install -y libgbm1 libnss3 libatk-bridge2.0-0 libdrm2 libxkbcommon0 libxcomposite1 libxdamage1 libxfixes3 libxrandr2 libasound2
```

### Custom cache directory

By default, Chrome for Testing installs to the platform-specific cache path shown in [Step 2](#step-2-create-a-project-folder). To change this (e.g. if your IT policy restricts writes to `%LOCALAPPDATA%`), set `VIBIUM_CACHE_DIR`:

```bash
# macOS/Linux
export VIBIUM_CACHE_DIR=/path/to/allowed/dir

# Windows (PowerShell)
$env:VIBIUM_CACHE_DIR = "C:\path\to\allowed\dir"
```

---

## Advanced: Manual Browser Install

If Chrome for Testing fails to auto-download (e.g. behind a corporate proxy), you can trigger the install manually.

### No build tool

```bash
java -jar vibium-26.3.18.jar install
```

### Maven

```bash
mvn compile exec:java -Dexec.mainClass=com.vibium.CLI -Dexec.args="install"
```

### Gradle

Add a task to your `build.gradle`:

```groovy
task vibiumInstall(type: JavaExec) {
    classpath = sourceSets.main.runtimeClasspath
    mainClass = 'com.vibium.CLI'
    args 'install'
}
```

Then run:

```bash
gradle vibiumInstall
```

### With overrides

Combine with environment variables as needed:

```bash
# Install to a custom directory
VIBIUM_CACHE_DIR=/path/to/dir java -jar vibium-26.3.18.jar install
VIBIUM_CACHE_DIR=/path/to/dir mvn compile exec:java -Dexec.mainClass=com.vibium.CLI -Dexec.args="install"
VIBIUM_CACHE_DIR=/path/to/dir gradle vibiumInstall

# Skip auto-download entirely (bring your own Chrome)
export VIBIUM_SKIP_BROWSER_DOWNLOAD=1
```

---

## You Did It!

You just automated a browser with Java. The same techniques work for:
- Web scraping
- Testing websites
- Automating repetitive tasks
- Building AI agents that can browse the web

Questions? [Open an issue](https://github.com/VibiumDev/vibium/issues).
