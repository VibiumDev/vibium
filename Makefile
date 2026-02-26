# Windows: use Git Bash as shell so Unix commands (cp, rm, mkdir, etc.) work
ifeq ($(OS),Windows_NT)
  SHELL := C:/PROGRA~1/Git/usr/bin/bash.exe
  .SHELLFLAGS := -c
  EXE := .exe
else
  EXE :=
endif

.PHONY: all build build-go build-js build-go-all package package-js package-python install-browser deps clean clean-go clean-js clean-npm-packages clean-python-packages clean-packages clean-cache clean-all serve test test-cli test-js test-mcp test-daemon test-python double-tap get-version set-version help

# Version from VERSION file
# Note: GnuWin32 Make 3.81 runs $(shell) via CreateProcess, not SHELL,
# so 'cat' must be on PATH (add Git's usr/bin — see docs/local-dev-setup-x86-windows.md)
VERSION := $(shell cat VERSION)

# Default target
all: build

# Build everything (Go + JS)
build: build-go build-js

# Build vibium binary
build-go: deps
	cp skills/vibe-check/SKILL.md clicker/cmd/clicker/SKILL.md
	cd clicker && go build -ldflags="-X main.version=$(VERSION)" -o bin/vibium$(EXE) ./cmd/clicker
	@if [ -d node_modules/@vibium ]; then \
		platform=$$(node -e "console.log(require('os').platform()+'-'+(require('os').arch()==='x64'?'x64':'arm64'))"); \
		target="node_modules/@vibium/$$platform/bin/vibium$(EXE)"; \
		if [ -f "$$target" ]; then cp clicker/bin/vibium$(EXE) "$$target"; fi; \
	fi

# Build JS client
build-js: deps
	cd clients/javascript && npm run build

# Cross-compile vibium for all platforms (static binaries)
# Output: clicker/bin/vibium-{os}-{arch}[.exe]
build-go-all:
	@echo "Cross-compiling vibium for all platforms..."
	cd clicker && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w -X main.version=$(VERSION)" -o bin/vibium-linux-amd64 ./cmd/clicker
	cd clicker && CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-s -w -X main.version=$(VERSION)" -o bin/vibium-linux-arm64 ./cmd/clicker
	cd clicker && CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w -X main.version=$(VERSION)" -o bin/vibium-darwin-amd64 ./cmd/clicker
	cd clicker && CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w -X main.version=$(VERSION)" -o bin/vibium-darwin-arm64 ./cmd/clicker
	cd clicker && CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="-s -w -X main.version=$(VERSION)" -o bin/vibium-windows-amd64.exe ./cmd/clicker
	@echo "Done. Built binaries:"
	@ls -lh clicker/bin/vibium-*

# Build all packages (npm + Python)
package: package-js package-python

# Build all npm packages for publishing
package-js: build-go-all build-js
	@echo "Copying binaries to platform packages..."
	mkdir -p packages/linux-x64/bin packages/linux-arm64/bin packages/darwin-x64/bin packages/darwin-arm64/bin packages/win32-x64/bin
	cp clicker/bin/vibium-linux-amd64 packages/linux-x64/bin/vibium
	cp clicker/bin/vibium-linux-arm64 packages/linux-arm64/bin/vibium
	cp clicker/bin/vibium-darwin-amd64 packages/darwin-x64/bin/vibium
	cp clicker/bin/vibium-darwin-arm64 packages/darwin-arm64/bin/vibium
	cp clicker/bin/vibium-windows-amd64.exe packages/win32-x64/bin/vibium.exe
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
	cp clicker/bin/vibium-linux-amd64 packages/python/vibium_linux_x64/src/vibium_linux_x64/bin/vibium
	cp clicker/bin/vibium-linux-arm64 packages/python/vibium_linux_arm64/src/vibium_linux_arm64/bin/vibium
	cp clicker/bin/vibium-darwin-amd64 packages/python/vibium_darwin_x64/src/vibium_darwin_x64/bin/vibium
	cp clicker/bin/vibium-darwin-arm64 packages/python/vibium_darwin_arm64/src/vibium_darwin_arm64/bin/vibium
	cp clicker/bin/vibium-windows-amd64.exe packages/python/vibium_win32_x64/src/vibium_win32_x64/bin/vibium.exe
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
	./clicker/bin/vibium$(EXE) install

# Install npm dependencies (skip if node_modules exists)
deps:
	@if [ ! -d "node_modules" ]; then npm install; fi

# Start the proxy server
serve: build-go
	./clicker/bin/vibium$(EXE) serve

# Build everything and run all tests: make test
test: build install-browser test-cli test-js test-mcp test-python

# Run CLI tests (tests the vibium binary directly)
# Process tests run separately with --test-concurrency=1 to avoid interference
# VIBIUM_ONESHOT=1 ensures tests use one-shot mode (no daemon)
test-cli: build-go
	@echo "━━━ CLI Tests ━━━"
	VIBIUM_ONESHOT=1 node --test tests/cli/navigation.test.js tests/cli/elements.test.js tests/cli/actionability.test.js tests/cli/page-reading.test.js tests/cli/input-tools.test.js tests/cli/tabs.test.js tests/cli/find-refs.test.js
	@echo "━━━ CLI Process Tests (sequential) ━━━"
	VIBIUM_ONESHOT=1 node --test --test-concurrency=1 tests/cli/process.test.js

# Run JS library tests (sequential to avoid resource exhaustion)
test-js: build
	@echo "━━━ JS Async API Tests ━━━"
	node --test --test-concurrency=1 tests/js/async/async-api.test.js tests/js/async/auto-wait.test.js tests/js/async/browser-modes.test.js
	@echo "━━━ JS Sync API Tests ━━━"
	node --test --test-concurrency=1 tests/js/sync/sync-api.test.js
	@echo "━━━ JS Element Finding Tests ━━━"
	node --test --test-concurrency=1 tests/js/async/elements.test.js
	@echo "━━━ JS Interaction Tests ━━━"
	node --test --test-concurrency=1 tests/js/async/interaction.test.js
	@echo "━━━ JS Element State Tests ━━━"
	node --test --test-concurrency=1 tests/js/async/state.test.js
	@echo "━━━ JS Input & Eval Tests (Keyboard, Mouse, Screenshots, Eval) ━━━"
	node --test --test-concurrency=1 tests/js/async/input-eval.test.js
	@echo "━━━ JS Network & Dialog Tests ━━━"
	node --test --test-concurrency=1 tests/js/async/network-dialog.test.js tests/js/sync/network-events.test.js
	@echo "━━━ JS WebSocket Monitoring Tests ━━━"
	node --test --test-concurrency=1 tests/js/async/websocket.test.js tests/js/sync/websocket-sync.test.js
	@echo "━━━ JS Console & Error Tests ━━━"
	node --test --test-concurrency=1 tests/js/async/console-error.test.js tests/js/sync/console-error.test.js
	@echo "━━━ JS Download & File Tests ━━━"
	node --test --test-concurrency=1 tests/js/async/download-file.test.js tests/js/sync/download-sync.test.js
	@echo "━━━ JS Tracing Tests ━━━"
	node --test --test-concurrency=1 tests/js/async/tracing.test.js
	@echo "━━━ JS Clock Tests ━━━"
	node --test --test-concurrency=1 tests/js/async/clock.test.js
	@echo "━━━ JS Emulation Tests ━━━"
	node --test --test-concurrency=1 tests/js/async/emulation.test.js
	@echo "━━━ JS Accessibility Tests ━━━"
	node --test --test-concurrency=1 tests/js/async/a11y.test.js
	@echo "━━━ JS A11y Tree Tutorial Tests ━━━"
	node --test --test-concurrency=1 tests/js/async/a11y-tree-tutorial.test.js tests/js/sync/a11y-tree-tutorial-sync.test.js
	@echo "━━━ JS Downloads Tutorial Tests ━━━"
	node --test --test-concurrency=1 tests/js/async/downloads-tutorial.test.js tests/js/sync/downloads-tutorial-sync.test.js
	@echo "━━━ JS Cookie & Storage Tests ━━━"
	node --test --test-concurrency=1 tests/js/async/cookies.test.js
	@echo "━━━ JS Frame Tests ━━━"
	node --test --test-concurrency=1 tests/js/async/frames.test.js
	@echo "━━━ JS Navigation & Lifecycle Tests ━━━"
	node --test --test-concurrency=1 tests/js/async/object-model.test.js tests/js/async/navigation.test.js tests/js/async/lifecycle.test.js
	@echo "━━━ JS Process Tests (sequential) ━━━"
	node --test --test-concurrency=1 tests/js/async/process.test.js tests/js/sync/process.test.js

# Run MCP server tests (sequential - browser sessions)
test-mcp: build-go
	@echo "━━━ MCP Server Tests ━━━"
	node --test --test-concurrency=1 tests/mcp/server.test.js

# Run daemon tests (sequential - daemon lifecycle)
test-daemon: build-go
	@echo "━━━ Daemon Tests ━━━"
	node --test --test-concurrency=1 tests/daemon/lifecycle.test.js tests/daemon/concurrency.test.js tests/daemon/cli-commands.test.js tests/daemon/find-refs.test.js

# Run Python client tests
test-python: build install-browser
	@echo "━━━ Python Client Tests ━━━"
	@cd clients/python && \
		if [ ! -d ".venv" ]; then python3 -m venv .venv; fi && \
		. .venv/bin/activate && \
		pip install -q -e ../../packages/python/vibium_darwin_arm64 -e ".[test]" && \
		VIBIUM_BIN_PATH=$(CURDIR)/clicker/bin/vibium$(EXE) \
		python -m pytest ../../tests/py/ -v --tb=short -x

# Kill zombie Chrome and chromedriver processes
double-tap:
	@echo "Killing zombie processes..."
ifeq ($(OS),Windows_NT)
	@cmd //c "taskkill /F /IM chrome.exe" 2>/dev/null || true
	@cmd //c "taskkill /F /IM chromedriver.exe" 2>/dev/null || true
else
	@pkill -9 -f 'Chrome for Testing' 2>/dev/null || true
	@pkill -9 -f chromedriver 2>/dev/null || true
endif
	@sleep 1
	@echo "Done."

# Clean Go binaries
clean-go:
	rm -rf clicker/bin

# Clean JS dist
clean-js:
	rm -rf clients/javascript/dist

# Clean built npm packages
clean-npm-packages:
	rm -f packages/*/bin/vibium packages/*/bin/vibium.exe
	rm -rf packages/vibium/dist
	rm -f packages/*/LICENSE packages/*/NOTICE clients/javascript/LICENSE clients/javascript/NOTICE

# Clean Python packages (venv, dist, platform binaries)
clean-python-packages:
	rm -rf clients/python/.venv clients/python/dist
	rm -f packages/python/*/src/*/bin/vibium packages/python/*/src/*/bin/vibium.exe
	rm -rf packages/python/*/dist
	rm -f packages/python/*/LICENSE packages/python/*/NOTICE clients/python/LICENSE clients/python/NOTICE

# Clean all built packages (npm + Python)
clean-packages: clean-npm-packages clean-python-packages

# Clean cached Chrome for Testing
clean-cache:
ifeq ($(OS),Windows_NT)
	rm -rf "$$LOCALAPPDATA/vibium/chrome-for-testing"
else
	rm -rf ~/Library/Caches/vibium/chrome-for-testing
	rm -rf ~/.cache/vibium/chrome-for-testing
endif

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

# Show available targets
help:
	@echo "Available targets:"
	@echo ""
	@echo "Build:"
	@echo "  make                       - Build everything (default)"
	@echo "  make build-go              - Build vibium binary"
	@echo "  make build-js              - Build JS client"
	@echo "  make build-go-all          - Cross-compile vibium for all platforms"
	@echo ""
	@echo "Package:"
	@echo "  make package               - Build all packages (npm + Python)"
	@echo "  make package-js            - Build npm packages only"
	@echo "  make package-python        - Build Python wheels only"
	@echo ""
	@echo "Test:"
	@echo "  make test                  - Build everything and run all tests (CLI + JS + MCP + Python)"
	@echo "  make test-cli              - Run CLI tests only"
	@echo "  make test-js               - Run JS library tests only"
	@echo "  make test-mcp              - Run MCP server tests only"
	@echo "  make test-daemon           - Run daemon lifecycle tests"
	@echo "  make test-python           - Run Python client tests"
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
	@echo "  make clean-go              - Clean Go binaries"
	@echo "  make clean-js              - Clean JS client dist"
	@echo "  make clean-npm-packages    - Clean built npm packages"
	@echo "  make clean-python-packages - Clean Python packages"
	@echo "  make clean-packages        - Clean all packages (npm + Python)"
	@echo "  make clean-cache           - Clean cached Chrome for Testing"
	@echo "  make clean-all             - Clean everything"
	@echo ""
	@echo "  make help                  - Show this help"
