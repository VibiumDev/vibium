# Publishing the Java Client to Maven Central

Step-by-step guide using the Central Portal (central.sonatype.com).

This document covers immutable stable releases. Automated mutable snapshots use
the separate [nightly release process](publishing-nightly-releases.md).

**Already done the one-time setup?** Skip to [Step 6: Build the Bundle](#6-build-the-bundle).

---

## 1. Create a Sonatype Account

Go to [central.sonatype.com](https://central.sonatype.com) and sign in with GitHub.

---

## 2. Verify the Namespace

You need to prove you own `com.vibium` (i.e., the domain `vibium.com`).

1. In the Central Portal, go to **Namespaces**
2. Click **Add Namespace**, enter `com.vibium`
3. It will give you a verification key (something like `abcd1234`)
4. Add a DNS TXT record to `vibium.com`:
   ```
   Type: TXT
   Host: @
   Value: <verification-key>
   ```
5. Click **Verify** — usually completes within a few minutes

You can check DNS propagation:
```bash
dig TXT vibium.com
```

---

## 3. Generate a GPG Key

Maven Central requires all artifacts to be GPG-signed.

```bash
# Install GPG (macOS)
brew install gnupg

# Generate a key (use the email associated with your Sonatype account)
gpg --full-generate-key
# Choose: (1) RSA and RSA, 4096 bits, 0 (no expiration), your name/email, Comment: blank, (O)kay

# List keys to get the key ID
gpg --list-keys --keyid-format short
# Look for the 8-character ID after "rsa4096/"

# Publish the public key to a keyserver
gpg --keyserver keyserver.ubuntu.com --send-keys <KEY_ID>
```

Verify it was published:
```bash
gpg --keyserver keyserver.ubuntu.com --recv-keys <KEY_ID>
```

---

## 4. Generate a Sonatype Token

1. Go to [central.sonatype.com/account/token](https://central.sonatype.com/account/token)
2. Click **Generate User Token**
3. Token Name: `vibium-publish`, Expiration: **Does not expire**
4. You'll get a username and password (they look like random strings, not your login)
5. Save these — you'll need them for the upload step

---

## 5. Configure Gradle Properties

Create or edit `~/.gradle/gradle.properties`:

```properties
# GPG signing (key ID from step 3)
signing.gnupg.keyName=<KEY_ID>
signing.gnupg.passphrase=<your-gpg-passphrase>   # optional — skips the pinentry dialog
```

**Note:** If you omit the passphrase, a pinentry dialog will pop up during the staging step asking for it. That's normal.

The build.gradle.kts already has the `maven-publish` and `signing` plugins configured.

Save your Sonatype token (from step 4) somewhere handy — you'll need it for the upload step, but it doesn't go in gradle.properties.

---

## 6. Build the Bundle

Bump the version — this updates `VERSION` and all package manifests (JS,
Python, Java) in one step — then test and package:

```bash
make set-version V=26.8.21
make test-java
make package-java
```

```console
package-java: 26.8.21 staged, 4 artifacts signed, 5 natives
package-java: vibium-bundle.zip ready to upload
```

`package-java` cross-compiles the native binaries the JAR packages, stages
signed artifacts into `clients/java/build/staging-deploy/`, checks that every
artifact exists, is signed, and carries all five natives, and zips
`vibium-bundle.zip` at the repo root. It fails rather than producing a bundle
Central will reject. It does not run tests — `make test-java` does, or
`make test` if you want the whole suite.

Run both through `make`, not `./gradlew` directly: the Makefile supplies
`VIBIUM_BIN_PATH` (so a globally installed `vibium` cannot outrank the build
under test, #331) and the shim that skips the ~15s Metal stall per Chrome
launch on an affected macOS VM. A bare `./gradlew` gets neither.

To look at what was staged:

```bash
ls -R clients/java/build/staging-deploy/com/vibium/vibium/
```

---

## 7. Upload to Central Portal

### Option A: Web UI

1. Go to [central.sonatype.com/publishing](https://central.sonatype.com/publishing)
2. Click **Publish Component**
3. Upload `vibium-bundle.zip`
4. Wait for validation (checks signatures, POM, javadoc, etc.)
5. Click **Publish** to release

### Option B: API (curl)

```bash
# Get your token from step 4
SONATYPE_USER="<token-username>"
SONATYPE_PASS="<token-password>"

# Upload the bundle
curl -X POST \
  "https://central.sonatype.com/api/v1/publisher/upload?publishingType=USER_MANAGED" \
  -H "Authorization: Bearer $(echo -n "$SONATYPE_USER:$SONATYPE_PASS" | base64)" \
  -F "bundle=@vibium-bundle.zip"
```

This returns a deployment ID. Save it, then check the status:

```bash
DEPLOYMENT_ID="<id-from-upload-response>"

curl -s -X POST \
  "https://central.sonatype.com/api/v1/publisher/status?id=$DEPLOYMENT_ID" \
  -H "Authorization: Bearer $(echo -n "$SONATYPE_USER:$SONATYPE_PASS" | base64)"
```

Once validation passes, publish it:

```bash
curl -X POST \
  "https://central.sonatype.com/api/v1/publisher/deployment/$DEPLOYMENT_ID" \
  -H "Authorization: Bearer $(echo -n "$SONATYPE_USER:$SONATYPE_PASS" | base64)"
```

---

## 8. Verify

After publishing, artifacts appear on Maven Central within ~30 minutes.

Check: `https://repo1.maven.org/maven2/com/vibium/vibium/`

Test it in a fresh project:

```bash
mkdir /tmp/vibium-java-test && cd /tmp/vibium-java-test

cat > Test.java << 'EOF'
import com.vibium.Vibium;
import com.vibium.types.StartOptions;

public class Test {
    public static void main(String[] args) {
        var bro = Vibium.start(new StartOptions().headless(true));
        try {
            var page = bro.page();
            page.go("data:text/html,<h1>ok</h1>");
            if (!page.find("h1").text().equals("ok")) throw new AssertionError();
            System.out.println("Vibium OK");
        } finally {
            bro.stop();
        }
    }
}
EOF

# Download the JAR
VERSION=<version>
curl -LO "https://repo1.maven.org/maven2/com/vibium/vibium/$VERSION/vibium-$VERSION.jar"
curl -LO "https://repo1.maven.org/maven2/com/google/code/gson/gson/2.11.0/gson-2.11.0.jar"

javac -cp "vibium-$VERSION.jar:gson-2.11.0.jar" Test.java
java -cp ".:vibium-$VERSION.jar:gson-2.11.0.jar" Test
```

This launches a browser from the JAR's packaged binary. Printing a class name
would pass even on a JAR with no binaries in it.

---

## Quick Reference

```bash
# Full publish flow (from repo root)
make set-version V=<version>
make test-java
make package-java

# Upload via web: central.sonatype.com → Publishing → Upload vibium-bundle.zip → Publish
```

---

## Troubleshooting

### "Invalid POM" during validation

The POM needs: name, description, url, licenses, developers, and scm. These are already in `build.gradle.kts`.

### "Invalid signature"

Make sure your GPG public key is published to a keyserver:
```bash
gpg --keyserver keyserver.ubuntu.com --send-keys <KEY_ID>
```

Central Portal checks these keyservers: `keyserver.ubuntu.com`, `keys.openpgp.org`, `pgp.mit.edu`.

### "Missing javadoc JAR" or "Missing sources JAR"

The `build.gradle.kts` already has `withSourcesJar()` and `withJavadocJar()`, and `make package-java` fails if either is missing from the staged tree.

### Namespace verification stuck

DNS propagation can be slow. Check with `dig TXT vibium.com` and wait.

### GPG passphrase prompt hanging in CI

Use `signing.gnupg.passphrase` in `gradle.properties`, or for CI, export the key as an environment variable and use the in-memory signing approach.
