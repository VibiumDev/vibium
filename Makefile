.PHONY: all build build-go build-js build-go-all package package-js package-python install-browser deps clean clean-go clean-js clean-npm-packages clean-python-packages clean-packages clean-cache clean-all serve test test-cli test-js test-mcp test-python test-java test-java-it double-tap get-version set-version help

# Version from VERSION file
VERSION := $(shell cat VERSION)

# Default target
all: build

# Build everything (Go + JS)
build: build-go build-js

# Build clicker binary
build-go: deps
	cd clicker && go build -ldflags="-X main.version=$(VERSION)" -o bin/clicker ./cmd/clicker

# Build JS client
build-js: deps
	cd clients/javascript && npm run build

# Cross-compile clicker for all platforms (static binaries)
# Output: clicker/bin/clicker-{os}-{arch}[.exe]
build-go-all:
	@echo "Cross-compiling clicker for all platforms..."
	cd clicker && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w -X main.version=$(VERSION)" -o bin/clicker-linux-amd64 ./cmd/clicker
	cd clicker && CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-s -w -X main.version=$(VERSION)" -o bin/clicker-linux-arm64 ./cmd/clicker
	cd clicker && CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w -X main.version=$(VERSION)" -o bin/clicker-darwin-amd64 ./cmd/clicker
	cd clicker && CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w -X main.version=$(VERSION)" -o bin/clicker-darwin-arm64 ./cmd/clicker
	cd clicker && CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="-s -w -X main.version=$(VERSION)" -o bin/clicker-windows-amd64.exe ./cmd/clicker
	@echo "Done. Built binaries:"
	@ls -lh clicker/bin/clicker-*

# Build all packages (npm + Python)
package: package-js package-python

# Build all npm packages for publishing
package-js: build-go-all build-js
	@echo "Copying binaries to platform packages..."
	mkdir -p packages/linux-x64/bin packages/linux-arm64/bin packages/darwin-x64/bin packages/darwin-arm64/bin packages/win32-x64/bin
	cp clicker/bin/clicker-linux-amd64 packages/linux-x64/bin/clicker
	cp clicker/bin/clicker-linux-arm64 packages/linux-arm64/bin/clicker
	cp clicker/bin/clicker-darwin-amd64 packages/darwin-x64/bin/clicker
	cp clicker/bin/clicker-darwin-arm64 packages/darwin-arm64/bin/clicker
	cp clicker/bin/clicker-windows-amd64.exe packages/win32-x64/bin/clicker.exe
	@echo "Copying LICENSE and NOTICE to npm packages..."
	@for pkg in packages/linux-x64 packages/linux-arm64 packages/darwin-x64 packages/darwin-arm64 packages/win32-x64 packages/vibium clients/javascript; do \
		cp LICENSE NOTICE "$$pkg/"; \
	done
	@echo "Building main vibium package..."
	mkdir -p packages/vibium/dist
	cp -r clients/javascript/dist/* packages/vibium/dist/
	@echo "All npm packages ready for publishing!"

# Build all Python packages (wheels)
package-python: build-go-all
	@echo "Copying binaries to Python platform packages..."
	mkdir -p packages/python/vibium_linux_x64/src/vibium_linux_x64/bin packages/python/vibium_linux_arm64/src/vibium_linux_arm64/bin packages/python/vibium_darwin_x64/src/vibium_darwin_x64/bin packages/python/vibium_darwin_arm64/src/vibium_darwin_arm64/bin packages/python/vibium_win32_x64/src/vibium_win32_x64/bin
	cp clicker/bin/clicker-linux-amd64 packages/python/vibium_linux_x64/src/vibium_linux_x64/bin/clicker
	cp clicker/bin/clicker-linux-arm64 packages/python/vibium_linux_arm64/src/vibium_linux_arm64/bin/clicker
	cp clicker/bin/clicker-darwin-amd64 packages/python/vibium_darwin_x64/src/vibium_darwin_x64/bin/clicker
	cp clicker/bin/clicker-darwin-arm64 packages/python/vibium_darwin_arm64/src/vibium_darwin_arm64/bin/clicker
	cp clicker/bin/clicker-windows-amd64.exe packages/python/vibium_win32_x64/src/vibium_win32_x64/bin/clicker.exe
	@echo "Copying LICENSE and NOTICE to Python packages..."
	@for pkg in packages/python/vibium_linux_x64 packages/python/vibium_linux_arm64 packages/python/vibium_darwin_x64 packages/python/vibium_darwin_arm64 packages/python/vibium_win32_x64 clients/python; do \
		cp LICENSE NOTICE "$$pkg/"; \
	done
	@echo "Building Python wheels..."
	@if [ ! -d ".venv-publish" ]; then \
		echo "Creating .venv-publish..."; \
		python3 -m venv .venv-publish && \
		. .venv-publish/bin/activate && \
		pip install -q twine; \
	fi
	@. .venv-publish/bin/activate && \
		cd packages/python/vibium_darwin_arm64 && pip wheel . -w dist --no-deps && \
		cd ../vibium_darwin_x64 && pip wheel . -w dist --no-deps && \
		cd ../vibium_linux_x64 && pip wheel . -w dist --no-deps && \
		cd ../vibium_linux_arm64 && pip wheel . -w dist --no-deps && \
		cd ../vibium_win32_x64 && pip wheel . -w dist --no-deps && \
		cd ../../../clients/python && pip wheel . -w dist --no-deps
	@echo "Done. Python wheels:"
	@ls -lh packages/python/*/dist/*.whl clients/python/dist/*.whl 2>/dev/null || true

# Install Chrome for Testing (required for tests)
install-browser: build-go
	./clicker/bin/clicker install

# Install npm dependencies (skip if node_modules exists)
deps:
	@if [ ! -d "node_modules" ]; then npm install; fi

# Start the proxy server
serve: build-go
	./clicker/bin/clicker serve

# Run all tests
test: build install-browser test-cli test-js test-mcp

# Run CLI tests (tests the clicker binary directly)
# Process tests run separately with --test-concurrency=1 to avoid interference
test-cli: build-go
	@echo "━━━ CLI Tests ━━━"
	node --test tests/cli/navigation.test.js tests/cli/elements.test.js tests/cli/actionability.test.js
	@echo "━━━ CLI Process Tests (sequential) ━━━"
	node --test --test-concurrency=1 tests/cli/process.test.js

# Run JS library tests (sequential to avoid resource exhaustion)
test-js: build
	@echo "━━━ JS Library Tests ━━━"
	node --test --test-concurrency=1 tests/js/async-api.test.js tests/js/sync-api.test.js tests/js/auto-wait.test.js tests/js/browser-modes.test.js
	@echo "━━━ JS Process Tests (sequential) ━━━"
	node --test --test-concurrency=1 tests/js/process.test.js

# Run MCP server tests (sequential - browser sessions)
test-mcp: build-go
	@echo "━━━ MCP Server Tests ━━━"
	node --test --test-concurrency=1 tests/mcp/server.test.js

# Run Python client tests
test-python: package-python
	@echo "━━━ Python Client Tests ━━━"
	@cd clients/python && \
		if [ ! -d ".venv" ]; then python3 -m venv .venv; fi && \
		. .venv/bin/activate && \
		pip install -q -e ../../packages/python/vibium_darwin_arm64 -e . && \
		python tests/test_basic.py

# Kill zombie Chrome and chromedriver processes
double-tap:
	@echo "Killing zombie processes..."
	@pkill -9 -f 'Chrome for Testing' 2>/dev/null || true
	@pkill -9 -f chromedriver 2>/dev/null || true
	@sleep 1
	@echo "Done."

# Clean clicker binaries
clean-go:
	rm -rf clicker/bin

# Clean JS dist
clean-js:
	rm -rf clients/javascript/dist

# Clean built npm packages
clean-npm-packages:
	rm -f packages/*/bin/clicker packages/*/bin/clicker.exe
	rm -rf packages/vibium/dist
	rm -f packages/*/LICENSE packages/*/NOTICE clients/javascript/LICENSE clients/javascript/NOTICE

# Clean Python packages (venv, dist, platform binaries)
clean-python-packages:
	rm -rf clients/python/.venv clients/python/dist
	rm -f packages/python/*/src/*/bin/clicker packages/python/*/src/*/bin/clicker.exe
	rm -rf packages/python/*/dist
	rm -f packages/python/*/LICENSE packages/python/*/NOTICE clients/python/LICENSE clients/python/NOTICE

# Clean all built packages (npm + Python)
clean-packages: clean-npm-packages clean-python-packages

# Clean cached Chrome for Testing
clean-cache:
	rm -rf ~/Library/Caches/vibium/chrome-for-testing
	rm -rf ~/.cache/vibium/chrome-for-testing

# Clean everything (binaries + JS dist + packages + cache)
clean-all: clean-go clean-js clean-packages clean-cache

# Alias for clean-go + clean-js
clean: clean-go clean-js

# Show current version
get-version:
	@cat VERSION

# Update version across all packages
# Usage: make set-version VERSION=x.x.x
set-version:
	@if [ -z "$(VERSION)" ]; then echo "Usage: make set-version VERSION=x.x.x"; exit 1; fi
	@echo "$(VERSION)" > VERSION
	@# Update all package.json version fields
	@for f in package.json packages/*/package.json clients/javascript/package.json; do \
		sed -i '' 's/"version": "[^"]*"/"version": "$(VERSION)"/' "$$f"; \
	done
	@# Update optionalDependencies versions in main package
	@sed -i '' 's/"\(@vibium\/[^"]*\)": "[^"]*"/"\1": "$(VERSION)"/g' packages/vibium/package.json
	@# Update all pyproject.toml files
	@for f in clients/python/pyproject.toml packages/python/*/pyproject.toml; do \
		sed -i '' 's/^version = "[^"]*"/version = "$(VERSION)"/' "$$f"; \
	done
	@# Update platform package dependency versions in main Python package
	@sed -i '' 's/vibium-\([^>]*\)>=[0-9][0-9]*\.[0-9][0-9]*\.[0-9][0-9]*/vibium-\1>=$(VERSION)/g' clients/python/pyproject.toml
	@# Update Python __version__ in __init__.py files
	@sed -i '' 's/__version__ = "[^"]*"/__version__ = "$(VERSION)"/' clients/python/src/vibium/__init__.py
	@for f in packages/python/*/src/*/__init__.py; do \
		sed -i '' 's/__version__ = "[^"]*"/__version__ = "$(VERSION)"/' "$$f"; \
	done
	@# Regenerate package-lock.json with new versions
	@rm -f package-lock.json
	@npm install --package-lock-only --silent
	@echo "Updated version to $(VERSION) in all files"
	@echo "Files updated:"
	@echo "  - VERSION"
	@echo "  - package.json (root)"
	@echo "  - packages/vibium/package.json (including optionalDependencies)"
	@echo "  - packages/*/package.json (5 platform packages)"
	@echo "  - clients/javascript/package.json"
	@echo "  - clients/python/pyproject.toml (version + dependencies)"
	@echo "  - clients/python/src/vibium/__init__.py"
	@echo "  - packages/python/*/pyproject.toml (5 platform packages)"
	@echo "  - packages/python/*/src/*/__init__.py (5 platform packages)"
	@echo "  - package-lock.json (regenerated)"

# Run Java client unit tests
test-java:
	@echo "=== Java Client Tests ==="
	cd clients/java && mvn test

# Run Java client integration tests (spawns Clicker + Chrome)
test-java-it:
	@echo "=== Java Client Integration Tests ==="
	cd clients/java && mvn verify -DskipITs=false

# Show available targets
help:
	@echo "Available targets:"
	@echo ""
	@echo "Build:"
	@echo "  make                       - Build everything (default)"
	@echo "  make build-go              - Build clicker binary"
	@echo "  make build-js              - Build JS client"
	@echo "  make build-go-all          - Cross-compile clicker for all platforms"
	@echo ""
	@echo "Package:"
	@echo "  make package               - Build all packages (npm + Python)"
	@echo "  make package-js            - Build npm packages only"
	@echo "  make package-python        - Build Python wheels only"
	@echo ""
	@echo "Test:"
	@echo "  make test                  - Run all tests (CLI + JS + MCP)"
	@echo "  make test-cli              - Run CLI tests only"
	@echo "  make test-js               - Run JS library tests only"
	@echo "  make test-mcp              - Run MCP server tests only"
	@echo "  make test-python           - Run Python client tests"
	@echo "  make test-java             - Run Java client unit tests"
	@echo "  make test-java-it          - Run Java client integration tests"
	@echo ""
	@echo "Other:"
	@echo "  make install-browser       - Install Chrome for Testing"
	@echo "  make deps                  - Install npm dependencies"
	@echo "  make serve                 - Start proxy server on :9515"
	@echo "  make double-tap            - Kill zombie Chrome/chromedriver processes"
	@echo "  make get-version           - Show current version"
	@echo "  make set-version VERSION=x.x.x - Set version across all packages"
	@echo ""
	@echo "Clean:"
	@echo "  make clean                 - Clean binaries and JS dist"
	@echo "  make clean-go              - Clean clicker binaries"
	@echo "  make clean-js              - Clean JS client dist"
	@echo "  make clean-npm-packages    - Clean built npm packages"
	@echo "  make clean-python-packages - Clean Python packages"
	@echo "  make clean-packages        - Clean all packages (npm + Python)"
	@echo "  make clean-cache           - Clean cached Chrome for Testing"
	@echo "  make clean-all             - Clean everything"
	@echo ""
	@echo "  make help                  - Show this help"
