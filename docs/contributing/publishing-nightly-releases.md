# Publishing Nightly Releases

Vibium publishes tested prereleases from the newest CI-green commit on `main`.
The workflow runs daily when that commit has changed and can also be dispatched
manually. Stable releases remain governed by
[Publishing New Releases](publishing-new-releases.md).

## Channels

| Ecosystem | Nightly coordinate | Stable default |
|---|---|---|
| npm | `npm install vibium@nightly` | `npm install vibium` |
| PyPI | `pip install --pre vibium==<version>` | `pip install vibium` |
| Maven | `com.vibium:vibium:YYYY.M-SNAPSHOT` | Maven Central release |
| GitHub | `nightly-<version>-<short-sha>` prerelease | `v*` release |

`nightly-candidate` is an intentionally public, unsupported npm implementation
tag. npm requires a tag when a version is published, so the workflow uses it
while testing exact public versions. Do not document it as an installation
channel. The supported `nightly` tag moves only after public smoke tests pass.

npm and PyPI spell the same build differently. For example:

```text
npm:  2026.8.21-dev.20260821153042
PyPI: 2026.8.21.dev20260821153042
```

The GitHub release's `manifest.json` maps both spellings to the commit and its
artifacts. Maven's monthly snapshot is replaceable by design; the manifest is
updated with the resolved timestamped snapshot after verification.

The GitHub JAR and Maven snapshot are separate builds of the selected commit.
The GitHub JAR embeds natives labeled with the npm nightly version, while the
Central JAR embeds natives labeled with the monthly snapshot version. The
GitHub manifest checksums therefore describe the GitHub JAR, not the mutable
snapshot served by Central.

## One-time Repository Setup

Create a GitHub environment named `nightly` and restrict deployment branches to
`main`. Require reviewers for the first live run, then remove that requirement
before enabling the schedule.

Configure the exact workflow file, `.github/workflows/nightly.yml`, as a trusted
publisher for all six npm packages and all six PyPI projects. npm trusted
publishing needs `npm publish` permission. Enable snapshots for the Sonatype
`com.vibium` namespace.

Add these environment secrets:

- `NPM_DIST_TAG_TOKEN`: granular write token limited to the six npm packages;
  it is used only because npm OIDC does not authorize `dist-tag` changes.
- `MAVEN_CENTRAL_USERNAME` and `MAVEN_CENTRAL_PASSWORD`.
- `GPG_PRIVATE_KEY`: base64-encoded signing key export.
- `GPG_PASSPHRASE`.

Add `NPM_DIST_TAG_TOKEN_EXPIRES_AT` as an environment variable containing an
ISO date. Rotate the token before that date and update both values together.
The workflow emits a warning during the final promotion step starting 30 days
before expiry and gives an explicit rotation error after expiry.

## Rollout

1. Dispatch with `dry_run` enabled. This builds, inventories, and locally
   installs every package without using publishing credentials.
2. Require a reviewer on the `nightly` environment and dispatch once with
   `dry_run` disabled.
3. Confirm exact public installs, stable npm/pip resolution, the Maven snapshot,
   release assets, and the generated manifest.
4. Remove the environment reviewer requirement. The schedule is now automatic.

## Failure Recovery

The workflow uses one non-cancelling `nightly-release` concurrency group, so a
manual run cannot overlap the scheduled run. Coordinates derive from the commit
timestamp and are deterministic. A rerun skips versions that already exist and
continues verification and promotion.

Registry publication is not transactional. A failed run can leave exact npm
candidates, PyPI prereleases, or a Maven snapshot visible. It must not move
`vibium@nightly` or publish the draft GitHub release. The workflow creates or
updates the `Nightly release failure` issue with the failed jobs and run URL,
then closes it after recovery.

GitHub package and binary assets are removed after 30 days, while each release,
tag, manifest, and checksum file remains for audit. Central retains snapshots
for approximately 90 days. npm and PyPI versions are immutable; review the size
summary monthly and address PyPI quota before any project reaches 70%.
