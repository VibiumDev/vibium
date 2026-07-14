# Publishing the Java Client to Maven Central

Step-by-step guide using the Central Portal (central.sonatype.com).

**Already done the one-time setup?** Skip to [Step 6: Build and Stage](#6-build-and-stage).

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

**Note:** If you omit the passphrase, a pinentry dialog will pop up during `./gradlew publish` asking for it. That's normal.

The build.gradle.kts already has the `maven-publish` and `signing` plugins configured.

Save your Sonatype token (from step 4) somewhere handy — you'll need it for the upload step, but it doesn't go in gradle.properties.

---

## 6. Build and Stage

First, bump the version. This updates `VERSION` and all package manifests (JS, Python, Java) in one step:

```bash
make set-version V=26.3.18
```

Then build from the repo root:

```bash
# Build all platform binaries (required for the JAR)
make build-go-all

# Clean and rebuild the Java client, then stage signed artifacts
cd clients/java
./gradlew clean build publish -PjavaParallel=1
cd ../..
```

> **`-PjavaParallel=1` is intentional.** `build` runs the test suite, and each
> test class launches its own Chrome. At the default `maxParallelForks=4`, four
> Chromes start at once and a couple often fail to connect
> (`VibiumConnectionException` / `IOException` at `Vibium.start()`) — a launch
> race, not a real failure. Running the tests serially avoids it. If you've
> already validated with `make test` and just want to stage artifacts, you can
> skip the tests instead with `./gradlew clean build publish -x test` (the
> published jar/sources/javadoc are identical either way).

This creates the signed artifacts in `clients/java/build/staging-deploy/`.

Verify the staged files:
```bash
ls -R clients/java/build/staging-deploy/com/vibium/vibium/
```

You should see:
```
vibium-<version>.jar
vibium-<version>.jar.asc          (GPG signature)
vibium-<version>.pom
vibium-<version>.pom.asc
vibium-<version>-sources.jar
vibium-<version>-sources.jar.asc
vibium-<version>-javadoc.jar
vibium-<version>-javadoc.jar.asc
```

Plus `.md5` and `.sha1` checksums for each.

---

## 7. Create the Bundle

Maven Central expects a single zip bundle:

```bash
cd clients/java/build/staging-deploy
zip -r ../../../../vibium-bundle.zip com/
cd ../../../..
```

This creates `vibium-bundle.zip` in the repo root.

---

## 8. Upload to Central Portal

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

## 9. Verify

After publishing, artifacts appear on Maven Central within ~30 minutes.

Check: `https://repo1.maven.org/maven2/com/vibium/vibium/`

Test it in a fresh project:

```bash
mkdir /tmp/vibium-java-test && cd /tmp/vibium-java-test

cat > Test.java << 'EOF'
import com.vibium.Vibium;
public class Test {
    public static void main(String[] args) {
        System.out.println("Vibium loaded: " + Vibium.class.getName());
    }
}
EOF

# Download the JAR
VERSION=$(cat /path/to/vibium/VERSION)
curl -LO "https://repo1.maven.org/maven2/com/vibium/vibium/$VERSION/vibium-$VERSION.jar"
curl -LO "https://repo1.maven.org/maven2/com/google/code/gson/gson/2.11.0/gson-2.11.0.jar"

javac -cp "vibium-$VERSION.jar:gson-2.11.0.jar" Test.java
java -cp ".:vibium-$VERSION.jar:gson-2.11.0.jar" Test
```

---

## Quick Reference

```bash
# Full publish flow (from repo root)
make build-go-all
# -PjavaParallel=1 runs the browser tests serially to avoid Chrome launch races
cd clients/java && ./gradlew clean build publish -PjavaParallel=1 && cd ../..

cd clients/java/build/staging-deploy && zip -r ../../../../vibium-bundle.zip com/ && cd ../../../..

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

The `build.gradle.kts` already has `withSourcesJar()` and `withJavadocJar()`. Just make sure `./gradlew build` runs before `./gradlew publish`.

### Namespace verification stuck

DNS propagation can be slow. Check with `dig TXT vibium.com` and wait.

### GPG passphrase prompt hanging in CI

Use `signing.gnupg.passphrase` in `gradle.properties`, or for CI, export the key as an environment variable and use the in-memory signing approach.
