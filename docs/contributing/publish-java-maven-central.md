# Publishing the Java Client to Maven Central

Step-by-step guide using the Central Portal (central.sonatype.com).

This document covers immutable stable releases. Automated mutable snapshots use
the separate [nightly release process](publishing-nightly-releases.md).

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

**Note:** If you omit the passphrase, a pinentry dialog will pop up during the staging step asking for it. That's normal.

The build.gradle.kts already has the `maven-publish` and `signing` plugins configured.

Save your Sonatype token (from step 4) somewhere handy — you'll need it for the upload step, but it doesn't go in gradle.properties.

---

## 6. Build and Stage

First, bump the version. This updates `VERSION` and all package manifests (JS, Python, Java) in one step:

```bash
make set-version V=26.8.21
```

Then build from the repo root:

```bash
make build-go-all

cd clients/java
VIBIUM_BIN_PATH=$(git rev-parse --show-toplevel)/clicker/bin/vibium \
  ./gradlew clean build publish -PjavaParallel=1
cd ../..
```

- `make build-go-all` — the JAR packages these binaries.
- `VIBIUM_BIN_PATH` — required. A globally installed `vibium` otherwise
  outranks the build being published and the tests fail against it (#331).
- `-PjavaParallel=1` — runs the browser tests serially; the default of 4
  launches four Chromes at once and some fail to connect. Add `-x test` to
  skip them if you already ran `make test`.

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

Confirm the JAR actually carries the five native binaries. Maven Central
releases are immutable, so an empty JAR cannot be replaced -- only superseded
by another version:

`verifyNativeBinaries` fails the publish if any are missing, so this is a
by-hand confirmation. Name the JAR explicitly — a `vibium-*.jar` glob also
matches the sources and javadoc JARs and prints nothing.

```console
$ V=$(cat VERSION)
$ jar tf clients/java/build/staging-deploy/com/vibium/vibium/$V/vibium-$V.jar | grep natives/
natives/vibium-darwin-amd64
natives/vibium-darwin-arm64
natives/vibium-linux-amd64
natives/vibium-linux-arm64
natives/vibium-windows-amd64.exe
```


---

## 7. Create the Bundle

Maven Central expects a single zip bundle:

```bash
cd clients/java/build/staging-deploy
zip -r ../../../../vibium-bundle.zip com/ -x 'com/vibium/vibium/maven-metadata.xml*'
cd ../../../..
```

This creates `vibium-bundle.zip` in the repo root. `maven-metadata.xml` is
excluded — it describes a repository, not a deployment.

Check it before uploading:

```console
$ unzip -l vibium-bundle.zip | grep -cE 'vibium-[0-9.]+(-sources|-javadoc)?\.(jar|pom)$'
4
$ unzip -l vibium-bundle.zip | grep -c '\.asc$'
5
$ unzip -l vibium-bundle.zip | grep -c 'maven-metadata'
0
```

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
make build-go-all

cd clients/java
VIBIUM_BIN_PATH=$(git rev-parse --show-toplevel)/clicker/bin/vibium \
  ./gradlew clean build publish -PjavaParallel=1
cd ../..

cd clients/java/build/staging-deploy
zip -r ../../../../vibium-bundle.zip com/ -x 'com/vibium/vibium/maven-metadata.xml*'
cd ../../../..

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

The `build.gradle.kts` already has `withSourcesJar()` and `withJavadocJar()`. Just make sure `build` runs before `publish`.

### Namespace verification stuck

DNS propagation can be slow. Check with `dig TXT vibium.com` and wait.

### GPG passphrase prompt hanging in CI

Use `signing.gnupg.passphrase` in `gradle.properties`, or for CI, export the key as an environment variable and use the in-memory signing approach.
