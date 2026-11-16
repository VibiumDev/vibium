# Publishing New Releases

End-to-end checklist for releasing a new version of Vibium.

For automated prereleases from `main`, see
[Publishing Nightly Releases](publishing-nightly-releases.md). Nightly
publication never promotes npm's `latest` tag and does not replace this stable
release checklist.

## Prerequisites

Before your first release, ensure you have:

- **npm**: Logged in (`npm login`) with access to `@vibium` org
- **PyPI**: API token from https://pypi.org/manage/account/token/
- **GitHub**: `gh` CLI authenticated (`gh auth login`)

## Version Bump

`main` takes changes only through a pull request and the merge queue, so
make the bump on a branch:

```bash
git switch -c release/26.11.16
make set-version VERSION=26.11.16
```

The version MUST have three parts (`X.Y.Z`). npm accepts nothing else, and
`set-version` refuses `26.11`. Releases are numbered by date: `YY.M.D`.

This updates:
- `VERSION`
- All `package.json` files and optionalDependencies
- All `pyproject.toml` files and dependency constraints
- Python `__init__.py` version strings
- The Maven coordinates in `README.md`, `CONTRIBUTING.md` and two tutorials
- Regenerates `package-lock.json`

## Build and Test

```bash
make clean-all
make build
make test
```

All tests must pass before proceeding.

## Merge the Bump

```bash
# Not `git add -A`: untracked files in the checkout stay out.
git commit -am "Bump version to 26.11.16"
git push -u origin release/26.11.16
gh pr create --fill
```

Merge the pull request through the merge queue. Then package and publish
from the commit it made on `main`:

```bash
SHA=$(gh pr view release/26.11.16 --json mergeCommit --jq .mergeCommit.oid)
git switch main && git pull
git rev-parse HEAD
```

If `HEAD` is not `$SHA`, other pull requests merged after the bump. Run
`git switch --detach $SHA` so they stay out of the release, and
`git switch main` once it is out.

## Package

```bash
make package
```

This builds wheels for Python and prepares npm packages.

## Publish to PyPI

Six packages total:

| Package | Description |
|---------|-------------|
| `vibium` | Main package (clients/python) |
| `vibium-darwin-arm64` | macOS Apple Silicon binary |
| `vibium-darwin-x64` | macOS Intel binary |
| `vibium-linux-x64` | Linux x64 binary |
| `vibium-linux-arm64` | Linux ARM64 binary |
| `vibium-win32-x64` | Windows x64 binary |

```bash
# Activate the publish venv (has twine installed)
source .venv-publish/bin/activate

# Optional: test on TestPyPI first
twine upload --repository testpypi packages/python/*/dist/*.whl
twine upload --repository testpypi clients/python/dist/*.whl
pip install --index-url https://test.pypi.org/simple/ vibium

# Upload platform packages (prompts for credentials — paste API token as password)
twine upload packages/python/*/dist/*.whl

# Upload main package
twine upload clients/python/dist/*.whl
```

## Publish to npm

**Important:** Platform packages must be published first.

```bash
# Platform packages (all must succeed before main)
(cd packages/linux-x64 && npm publish --access public)
(cd packages/linux-arm64 && npm publish --access public)
(cd packages/darwin-x64 && npm publish --access public)
(cd packages/darwin-arm64 && npm publish --access public)
(cd packages/win32-x64 && npm publish --access public)

# Main package (after all platform packages are live)
(cd packages/vibium && npm publish)
```

## Publish to Maven Central

`make set-version` changed the Maven coordinates in the README and the
tutorials, so the Java client MUST go out with the same release. Follow
[Publishing the Java Client to Maven Central](publish-java-maven-central.md).

## Create GitHub Release

```bash
# Tag the bump's commit on main, and push the tag alone
git tag v26.11.16 $SHA
git push origin v26.11.16

# Notes since the last stable release, not since the last nightly
gh release create v26.11.16 --generate-notes --notes-start-tag v26.8.21 --title "v26.11.16"

# Attach the binaries and their checksums, so an image can pin the release
(cd clicker/bin && shasum -a 256 vibium-linux-amd64 vibium-linux-arm64 \
  vibium-darwin-amd64 vibium-darwin-arm64 vibium-windows-amd64.exe > SHA256SUMS)
gh release upload v26.11.16 clicker/bin/vibium-linux-amd64 clicker/bin/vibium-linux-arm64 \
  clicker/bin/vibium-darwin-amd64 clicker/bin/vibium-darwin-arm64 \
  clicker/bin/vibium-windows-amd64.exe clicker/bin/SHA256SUMS
```

## Verification

Test the published packages work correctly:

```bash
# Test npm
mkdir /tmp/vibium-test && cd /tmp/vibium-test
npm init -y
npm install vibium
node -e "const { browser } = require('vibium'); console.log('npm OK')"

# Test PyPI. vibium requires Python 3.11+; macOS python3 is 3.9.
# Use any 3.11+ interpreter, e.g. `brew install python@3.12`.
python3.12 -m venv /tmp/vibium-pypi-test
/tmp/vibium-pypi-test/bin/pip install vibium==26.11.16
/tmp/vibium-pypi-test/bin/python -c "from vibium import browser; print('PyPI OK')"
```

## Troubleshooting

### "Cannot publish over previously published version"

You've already published this version. Bump to a new version number.

### "You must be logged in to publish"

```bash
npm login
npm whoami  # Verify you're logged in
```

### PyPI upload fails with 403

Your API token may be expired or scoped incorrectly. Create a new token at https://pypi.org/manage/account/token/

### Platform package not found after npm publish

Wait a few seconds for npm registry propagation before publishing the main package.

## Quick Reference

```bash
# Complete release (replace 26.11.16 with your version)
git switch -c release/26.11.16
make set-version VERSION=26.11.16
make clean-all && make build && make test

git commit -am "Bump version to 26.11.16"
git push -u origin release/26.11.16
gh pr create --fill
# merge through the merge queue, then:
SHA=$(gh pr view release/26.11.16 --json mergeCommit --jq .mergeCommit.oid)
git switch main && git pull
git rev-parse HEAD   # not $SHA? git switch --detach $SHA

make package

source .venv-publish/bin/activate
twine upload packages/python/*/dist/*.whl
twine upload clients/python/dist/*.whl

(cd packages/linux-x64 && npm publish --access public)
(cd packages/linux-arm64 && npm publish --access public)
(cd packages/darwin-x64 && npm publish --access public)
(cd packages/darwin-arm64 && npm publish --access public)
(cd packages/win32-x64 && npm publish --access public)
(cd packages/vibium && npm publish)

# The Java client: see publish-java-maven-central.md

git tag v26.11.16 $SHA
git push origin v26.11.16
gh release create v26.11.16 --generate-notes --notes-start-tag v26.8.21 --title "v26.11.16"
# then attach the binaries and SHA256SUMS, as under "Create GitHub Release"
```
