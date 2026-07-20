# Windows: use Git Bash as shell so Unix commands (cp, rm, mkdir, etc.) work
ifeq ($(OS),Windows_NT)
  SHELL := C:/PROGRA~1/Git/usr/bin/bash.exe
  .SHELLFLAGS := -c
  EXE := .exe
  PYTHON := python
  VENV_ACTIVATE := .venv/Scripts/activate
  PYTHON_PLATFORM_PKG := vibium_win32_x64
else
  EXE :=
  PYTHON := python3
  VENV_ACTIVATE := .venv/bin/activate
  UNAME_S := $(shell uname -s)
  UNAME_M := $(shell uname -m)
  ifeq ($(UNAME_S),Darwin)
    ifeq ($(UNAME_M),arm64)
      PYTHON_PLATFORM_PKG := vibium_darwin_arm64
    else
      PYTHON_PLATFORM_PKG := vibium_darwin_x64
    endif
  else
    ifeq ($(UNAME_M),aarch64)
      PYTHON_PLATFORM_PKG := vibium_linux_arm64
    else
      PYTHON_PLATFORM_PKG := vibium_linux_x64
    endif
  endif
endif

.PHONY: all build build-go build-js build-go-all package package-js package-python install-browser deps clean clean-go clean-js clean-npm-packages clean-python-packages clean-packages clean-cache clean-all serve test test-cli test-js test-js-async test-js-sync test-js-process test-mcp test-daemon test-python test-java test-cleanup double-tap get-version set-version build-java package-java publish-java clean-java jshell help

# Version from VERSION file
# Note: GnuWin32 Make 3.81 runs $(shell) via CreateProcess, not SHELL,
# so 'cat' must be on PATH (add Git's usr/bin — see docs/how-to-guides/local-dev-setup-x86-windows.md)
VERSION := $(shell cat VERSION)
# Allow V= as shorthand for VERSION=
ifdef V
  override VERSION := $(V)
endif

# Per-group test timeout in seconds (override: make test TEST_TIMEOUT=300).
# This is the outer wrapper for a whole phase (test-cli, test-js, etc.) and
# only fires when something has gone catastrophically wrong. A healthy
# sequential test-js phase is ~6-10 minutes because Chrome launch is ~16s
# per file on macOS (see clients/javascript/src/clicker/process.ts). For
# faster iteration bump JS_PARALLEL (default 3) to use more cores.
TEST_TIMEOUT ?= 600
TIMEOUT_CMD := node scripts/timeout.mjs $(TEST_TIMEOUT)
# Same watchdog for recipes that cd out of the repo root (pytest, gradlew).
# Empty on Windows: timeout.mjs spawns via cmd.exe, which can't run ./gradlew.
ifeq ($(OS),Windows_NT)
  TIMEOUT_CMD_ABS :=
else
  TIMEOUT_CMD_ABS := node $(CURDIR)/scripts/timeout.mjs $(TEST_TIMEOUT)
endif

# Node test runner flags: per-test timeout + force exit on dangling handles.
# The slowest healthy test today is ~20s (websocket "monitoring survives
# page navigation"). 30s gives headroom while making a hung test surface
# in 30s instead of 2 minutes — so a flake takes ~30s × N_stuck_tests to
# trip the phase wrapper instead of ~120s × N.
# On Node 20, --test-timeout limits each test FILE, not each test; CI
# overrides this to file scale (see .github/workflows/test.yml).
NODE_TEST_TIMEOUT ?= 30000
TEST_FLAGS := --test-timeout=$(NODE_TEST_TIMEOUT) --test-force-exit

# Default target
all: build

# Build everything (Go + JS + Java)
build: build-go build-js build-java

# Build vibium binary
build-go: deps
	cp skills/vibe-check/SKILL.md clicker/cmd/clicker/SKILL.md
	cd clicker && go build -ldflags="-X main.version=$(VERSION) -X github.com/vibium/clicker/internal/api.Version=$(VERSION)" -o bin/vibium$(EXE) ./cmd/clicker
	@if [ -d node_modules/@vibium ]; then \
		platform=$$(node -e "console.log(require('os').platform()+'-'+(require('os').arch()==='x64'?'x64':'arm64'))"); \
		target_dir="node_modules/@vibium/$$platform/bin"; \
		if [ -d "node_modules/@vibium/$$platform" ]; then \
			mkdir -p "$$target_dir"; \
			cp clicker/bin/vibium$(EXE) "$$target_dir/vibium$(EXE)"; \
		fi; \
	fi

# Build JS client
build-js: deps
	cd clients/javascript && npm run build

# Cross-compile vibium for all platforms (static binaries)
# Output: clicker/bin/vibium-{os}-{arch}[.exe]
build-go-all:
	@echo "Cross-compiling vibium for all platforms..."
	cd clicker && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w -X main.version=$(VERSION) -X github.com/vibium/clicker/internal/api.Version=$(VERSION)" -o bin/vibium-linux-amd64 ./cmd/clicker
	cd clicker && CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-s -w -X main.version=$(VERSION) -X github.com/vibium/clicker/internal/api.Version=$(VERSION)" -o bin/vibium-linux-arm64 ./cmd/clicker
	cd clicker && CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w -X main.version=$(VERSION) -X github.com/vibium/clicker/internal/api.Version=$(VERSION)" -o bin/vibium-darwin-amd64 ./cmd/clicker
	cd clicker && CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w -X main.version=$(VERSION) -X github.com/vibium/clicker/internal/api.Version=$(VERSION)" -o bin/vibium-darwin-arm64 ./cmd/clicker
	cd clicker && CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="-s -w -X main.version=$(VERSION) -X github.com/vibium/clicker/internal/api.Version=$(VERSION)" -o bin/vibium-windows-amd64.exe ./cmd/clicker
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
# test-js-sync runs OUTSIDE the parallel group on every platform. Each of
# test-js-async/sync/python/java fans out to *_PARALLEL headless Chromes, and
# running all of them at once over-subscribes the machine (~14 concurrent
# browsers on a 12-core box), pushing cold Chrome launches past the client's
# ready timeout and cascading into cancellations. Keeping test-js-sync separate
# caps the peak; the *_PARALLEL defaults are tuned conservatively for the same
# reason. Bump *_PARALLEL on machines with more cores/memory.
test: build install-browser
	@START_TIME=$$(date +%s); \
	"$(MAKE)" test-cli test-cleanup && \
	"$(MAKE)" test-js-process test-cleanup && \
	"$(MAKE)" -j 4 test-js-async test-mcp test-python test-java && \
	"$(MAKE)" test-cleanup && \
	"$(MAKE)" test-js-sync; \
	EXIT=$$?; \
	"$(MAKE)" test-cleanup; \
	END_TIME=$$(date +%s); \
	ELAPSED=$$((END_TIME - START_TIME)); \
	MINS=$$((ELAPSED / 60)); \
	SECS=$$((ELAPSED % 60)); \
	echo ""; \
	if [ $$EXIT -eq 0 ]; then \
		echo "--- All tests passed in $${MINS}m$${SECS}s ---"; \
	else \
		echo "--- Tests failed after $${MINS}m$${SECS}s ---"; \
		exit $$EXIT; \
	fi

# Kill any Chrome/chromedriver processes left over from tests.
# The bracketed characters stop Linux pkill from SIGKILLing the recipe's
# own shell, whose command line contains the pattern.
test-cleanup:
	@$(CURDIR)/clicker/bin/vibium$(EXE) daemon stop 2>/dev/null || true
	@pkill -9 -f 'Chrome for [T]esting' 2>/dev/null || true
	@pkill -9 -f 'chrome-for-testin[g]' 2>/dev/null || true
	@pkill -9 -f 'chromedrive[r]' 2>/dev/null || true
	@pkill -9 -f 'sync-test-server.j[s]' 2>/dev/null || true

# Run CLI tests (tests the vibium binary directly)
# Process tests run separately with --test-concurrency=1 to avoid interference
test-cli: build-go
	@echo "--- CLI Tests (no daemon) ---"
	$(TIMEOUT_CMD) node --test $(TEST_FLAGS) --test-concurrency=1 tests/cli/is-installed.test.js tests/cli/packaging.test.js tests/cli/wrapper.test.js
	@echo "--- CLI Tests ---"
	@$(CURDIR)/clicker/bin/vibium$(EXE) daemon stop 2>/dev/null || true
	@$(CURDIR)/clicker/bin/vibium$(EXE) daemon start --headless
	$(TIMEOUT_CMD) node --test $(TEST_FLAGS) --test-concurrency=1 tests/cli/navigation.test.js tests/cli/elements.test.js tests/cli/actionability.test.js tests/cli/page-reading.test.js tests/cli/input-tools.test.js tests/cli/pages.test.js tests/cli/page-context.test.js tests/cli/find-refs.test.js
	@$(CURDIR)/clicker/bin/vibium$(EXE) daemon stop 2>/dev/null || true
	@echo "--- CLI Process Tests (sequential) ---"
	$(TIMEOUT_CMD) node --test $(TEST_FLAGS) --test-concurrency=1 tests/cli/process.test.js

# Run JS library tests
# Each test file owns its own browser (top-level before/after), so files are
# independent and safe to run in parallel. JS_PARALLEL controls the fan-out:
# default 4 gives ~3x speedup vs sequential. (Previously sequential because
# we suspected parallel-induced flakes; root cause was a cross-process Chrome
# temp-dir cleanup race in clicker/internal/browser/launcher.go, now fixed.)
# Process tests stay sequential because they assert on Chrome process lifecycle.
#
# The async/sync/process subgroups are also exposed as separate make targets
# (test-js-async, test-js-sync, test-js-process) so `make test` can run the
# parallel-safe groups concurrently with test-mcp/test-python/test-java via
# `$(MAKE) -j 5`. The process group must run alone because it asserts on
# Chrome PID baselines.
JS_PARALLEL ?= 3

test-js-async: build-go
	@echo "--- JS Async Tests (parallel x$(JS_PARALLEL)) ---"
	$(TIMEOUT_CMD) node --test $(TEST_FLAGS) --test-concurrency=$(JS_PARALLEL) \
		tests/js/async/async-api.test.js \
		tests/js/async/auto-wait.test.js \
		tests/js/async/browser-modes.test.js \
		tests/js/async/elements.test.js \
		tests/js/async/interaction.test.js \
		tests/js/async/state.test.js \
		tests/js/async/input-eval.test.js \
		tests/js/async/network-dialog.test.js \
		tests/js/async/console-error.test.js \
		tests/js/async/clock.test.js \
		tests/js/async/emulation.test.js \
		tests/js/async/a11y.test.js \
		tests/js/async/a11y-tree-tutorial.test.js \
		tests/js/async/websocket.test.js \
		tests/js/async/download-file.test.js \
		tests/js/async/recording.test.js \
		tests/js/async/downloads-tutorial.test.js \
		tests/js/async/cookies.test.js \
		tests/js/async/storage.test.js \
		tests/js/async/frames.test.js \
		tests/js/async/object-model.test.js \
		tests/js/async/navigation.test.js \
		tests/js/async/lifecycle.test.js

test-js-sync: build-go
	@echo "--- JS Sync Tests (parallel x$(JS_PARALLEL)) ---"
	$(TIMEOUT_CMD) node --test $(TEST_FLAGS) --test-concurrency=$(JS_PARALLEL) \
		tests/js/sync/sync-api.test.js \
		tests/js/sync/network-events.test.js \
		tests/js/sync/websocket-sync.test.js \
		tests/js/sync/console-error.test.js \
		tests/js/sync/download-sync.test.js \
		tests/js/sync/a11y-tree-tutorial-sync.test.js \
		tests/js/sync/downloads-tutorial-sync.test.js

test-js-process: build-go
	@echo "--- JS Process Tests (sequential) ---"
	$(TIMEOUT_CMD) node --test $(TEST_FLAGS) --test-concurrency=1 \
		tests/js/async/process.test.js \
		tests/js/sync/process.test.js

# Backward-compat aggregate: run all three JS test groups sequentially.
test-js: test-js-async test-js-sync test-js-process

# Run MCP server tests (sequential - browser sessions)
test-mcp: build-go
	@echo "--- MCP Server Tests ---"
	$(TIMEOUT_CMD) node --test $(TEST_FLAGS) --test-concurrency=1 tests/mcp/server.test.js

# Run daemon tests (sequential - daemon lifecycle)
test-daemon: build-go
	@echo "--- Daemon Tests ---"
	$(TIMEOUT_CMD) node --test $(TEST_FLAGS) --test-concurrency=1 tests/daemon/lifecycle.test.js tests/daemon/concurrency.test.js tests/daemon/cli-commands.test.js tests/daemon/find-refs.test.js tests/daemon/connect.test.js tests/daemon/recording.test.js

# Run Python client tests
# PY_PARALLEL: pytest-xdist worker count. Each worker spawns its own Chrome,
# so each adds ~150 MB of memory pressure. Default 4 is conservative; bump
# for faster CI. Module-scoped browser fixture means xdist's default
# loadfile distribution gives each file to a single worker — safe under
# parallel since each file owns its own browser via conftest.py.
PY_PARALLEL ?= 3
test-python: build-go install-browser
	@echo "--- Python Client Tests (parallel x$(PY_PARALLEL)) ---"
	@cd clients/python && \
		if [ ! -d ".venv" ]; then $(PYTHON) -m venv .venv; fi && \
		. $(VENV_ACTIVATE) && \
		if ! python -c "import vibium, xdist" 2>/dev/null; then \
			python -m pip install --quiet --upgrade pip && \
			pip install -e ../../packages/python/$(PYTHON_PLATFORM_PKG) -e ".[test]"; \
		fi && \
		VIBIUM_BIN_PATH=$(CURDIR)/clicker/bin/vibium$(EXE) \
		$(TIMEOUT_CMD_ABS) python -m pytest ../../tests/py/ -v --tb=short -x -n $(PY_PARALLEL) --dist=loadfile

# Build Java client JAR (dev — no native binaries, fast)
build-java: build-go
	@if [ ! -f clients/java/gradlew ]; then cd clients/java && gradle wrapper; fi
	cd clients/java && ./gradlew build -x test

# Run Java client tests
# JAVA_PARALLEL: number of parallel test JVMs (each spawns its own Chrome).
# Default 4; bump for faster CI on machines with more memory.
JAVA_PARALLEL ?= 3
test-java: build-go install-browser
	@echo "--- Java Client Tests (parallel x$(JAVA_PARALLEL)) ---"
	cd clients/java && VIBIUM_BIN_PATH=$(CURDIR)/clicker/bin/vibium$(EXE) $(TIMEOUT_CMD_ABS) ./gradlew test -PjavaParallel=$(JAVA_PARALLEL)

# Package Java JAR with native binaries
package-java: build-go-all
	cd clients/java && ./gradlew jar

# Publish Java JAR to Maven Central
publish-java: package-java
	cd clients/java && ./gradlew publishAllPublicationsToSonatypeCentralRepository

# Interactive JShell with the Java client
jshell: build-java
	VIBIUM_BIN_PATH=$(CURDIR)/clicker/bin/vibium$(EXE) jshell --class-path "$$(find clients/java/build/libs -name 'vibium-*.jar' ! -name '*-sources*' ! -name '*-javadoc*' | head -1):$$(find clients/java/build/dependencies -name '*.jar' | paste -sd ':' -)"

# Clean Java build artifacts
clean-java:
	cd clients/java && ./gradlew clean

# Kill zombie Chrome and chromedriver processes
double-tap:
	@echo "Killing zombie processes..."
ifeq ($(OS),Windows_NT)
	@cmd //c "taskkill /F /IM chrome.exe" 2>/dev/null || true
	@cmd //c "taskkill /F /IM chromedriver.exe" 2>/dev/null || true
else
	@pkill -9 -f 'Chrome for [T]esting' 2>/dev/null || true
	@pkill -9 -f 'chrome-for-testin[g]' 2>/dev/null || true
	@pkill -9 -f 'chromedrive[r]' 2>/dev/null || true
	@pkill -9 -f 'sync-test-server.j[s]' 2>/dev/null || true
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
	rm -f packages/*/*.tgz
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
# Usage: make set-version VERSION=x.x.x  (or V=x.x.x)
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
	@# The just-bumped @vibium/* platform packages aren't published to npm yet, so
	@# npm leaves their lockfile entries without a version field. That crashes
	@# `npm install` on fresh clones with "Invalid Version: " during dedupe. Backfill
	@# the version so the lockfile is valid; npm fills in resolved/integrity on the
	@# first real install once the packages are published.
	@VIBIUM_VERSION="$(VERSION)" node -e 'const fs=require("fs"),f="package-lock.json",v=process.env.VIBIUM_VERSION,l=JSON.parse(fs.readFileSync(f,"utf8"));let n=0;for(const[k,e]of Object.entries(l.packages||{}))if(/(^|\/)@vibium\//.test(k)&&e&&e.version==null){e.version=v;n++}fs.writeFileSync(f,JSON.stringify(l,null,2)+"\n");console.log("Backfilled "+n+" @vibium lockfile entr"+(n===1?"y":"ies")+" with version "+v)'
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
	@echo "  make build-java            - Build Java client JAR"
	@echo "  make jshell               - Interactive JShell with the Java client"
	@echo "  make build-go-all          - Cross-compile vibium for all platforms"
	@echo ""
	@echo "Package:"
	@echo "  make package               - Build all packages (npm + Python)"
	@echo "  make package-js            - Build npm packages only"
	@echo "  make package-python        - Build Python wheels only"
	@echo "  make package-java          - Build Java JAR with native binaries"
	@echo ""
	@echo "Test:"
	@echo "  make test                  - Build everything and run all tests (CLI + JS + MCP + Python + Java)"
	@echo "  make test-cli              - Run CLI tests only"
	@echo "  make test-js               - Run JS library tests only"
	@echo "  make test-mcp              - Run MCP server tests only"
	@echo "  make test-daemon           - Run daemon lifecycle tests"
	@echo "  make test-python           - Run Python client tests"
	@echo "  make test-java             - Run Java client tests"
	@echo ""
	@echo "Other:"
	@echo "  make install-browser       - Install Chrome for Testing"
	@echo "  make deps                  - Install npm dependencies"
	@echo "  make serve                 - Start proxy server on :9515"
	@echo "  make double-tap            - Kill zombie Chrome/chromedriver processes"
	@echo "  make get-version           - Show current version"
	@echo "  make set-version VERSION=x.x.x - Set version across all packages (V= also works)"
	@echo ""
	@echo "Clean:"
	@echo "  make clean                 - Clean binaries and JS dist"
	@echo "  make clean-go              - Clean Go binaries"
	@echo "  make clean-js              - Clean JS client dist"
	@echo "  make clean-npm-packages    - Clean built npm packages"
	@echo "  make clean-python-packages - Clean Python packages"
	@echo "  make clean-packages        - Clean all packages (npm + Python)"
	@echo "  make clean-java            - Clean Java build artifacts"
	@echo "  make clean-cache           - Clean cached Chrome for Testing"
	@echo "  make clean-all             - Clean everything"
	@echo ""
	@echo "  make help                  - Show this help"
