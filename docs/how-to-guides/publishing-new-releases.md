# Publishing New Releases

End-to-end checklist for releasing a new version of Vibium.

## Prerequisites

Before your first release, ensure you have:

- **npm**: Logged in (`npm login`) with access to `@vibium` org
- **PyPI**: API token from https://pypi.org/manage/account/token/
- **GitHub**: `gh` CLI authenticated (`gh auth login`)

## Pre-release Checklist

```bash
# Clean everything
make clean-all

# Build and test
make build
make test
```

All tests must pass before proceeding.

## Version Bump

```bash
make set-version VERSION=0.1.4
```

This updates:
- All `package.json` files and optionalDependencies
- All `pyproject.toml` files and dependency constraints
- Python `__init__.py` version strings
- Regenerates `package-lock.json`

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

## Create GitHub Release

```bash
# Commit version bump
git add -A && git commit -m "bump version to 0.1.4"

# Tag and push
git tag v0.1.4
git push origin main --tags

# Create release with auto-generated notes
gh release create v0.1.4 --generate-notes --title "v0.1.4"
```

## Verification

Test the published packages work correctly:

```bash
# Test npm
mkdir /tmp/vibium-test && cd /tmp/vibium-test
npm init -y
npm install vibium
node -e "const { browser } = require('vibium'); console.log('npm OK')"

# Test PyPI
pip install --upgrade vibium
python3 -c "from vibium import browser_sync; print('PyPI OK')"
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
# Complete release (replace 0.1.4 with your version)
make clean-all && make test
make set-version VERSION=0.1.4
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

git add -A && git commit -m "bump version to 0.1.4"
git tag v0.1.4
git push origin main --tags
gh release create v0.1.4 --generate-notes --title "v0.1.4"
```
