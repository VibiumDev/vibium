.PHONY: all build build-go build-js build-all-platforms package package-platforms package-main package-python-platforms package-python install-browser deps clean clean-bin clean-js clean-packages clean-python-packages clean-cache clean-all serve test test-cli test-js test-mcp test-python double-tap help

# Default target
all: build

# Build everything (Go + JS)
build: build-go build-js

# Build clicker binary
build-go:
	cd clicker && go build -o bin/clicker ./cmd/clicker

# Build JS client
build-js: deps
	cd clients/javascript && npm run build

# Cross-compile clicker for all platforms (static binaries)
# Output: clicker/bin/clicker-{os}-{arch}[.exe]
build-all-platforms:
	@echo "Cross-compiling clicker for all platforms..."
	cd clicker && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o bin/clicker-linux-amd64 ./cmd/clicker
	cd clicker && CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o bin/clicker-linux-arm64 ./cmd/clicker
	cd clicker && CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o bin/clicker-darwin-amd64 ./cmd/clicker
	cd clicker && CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o bin/clicker-darwin-arm64 ./cmd/clicker
	cd clicker && CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o bin/clicker-windows-amd64.exe ./cmd/clicker
	@echo "Done. Built binaries:"
	@ls -lh clicker/bin/clicker-*

# Copy binaries to platform packages for npm publishing
package-platforms: build-all-platforms
	@echo "Copying binaries to platform packages..."
	mkdir -p packages/linux-x64/bin packages/linux-arm64/bin packages/darwin-x64/bin packages/darwin-arm64/bin packages/win32-x64/bin
	cp clicker/bin/clicker-linux-amd64 packages/linux-x64/bin/clicker
	cp clicker/bin/clicker-linux-arm64 packages/linux-arm64/bin/clicker
	cp clicker/bin/clicker-darwin-amd64 packages/darwin-x64/bin/clicker
	cp clicker/bin/clicker-darwin-arm64 packages/darwin-arm64/bin/clicker
	cp clicker/bin/clicker-windows-amd64.exe packages/win32-x64/bin/clicker.exe
	@echo "Done. Package binaries:"
	@ls -lh packages/*/bin/clicker*

# Build main vibium package (copy JS dist)
package-main: build-js
	@echo "Building main vibium package..."
	mkdir -p packages/vibium/dist
	cp -r clients/javascript/dist/* packages/vibium/dist/
	@echo "Done. Main package ready at packages/vibium/"

# Build all packages for npm publishing
package: package-platforms package-main
	@echo "All npm packages ready for publishing!"

# Copy binaries to Python platform packages
package-python-platforms: build-all-platforms
	@echo "Copying binaries to Python platform packages..."
	mkdir -p packages/python/vibium_linux_x64/src/vibium_linux_x64/bin packages/python/vibium_linux_arm64/src/vibium_linux_arm64/bin packages/python/vibium_darwin_x64/src/vibium_darwin_x64/bin packages/python/vibium_darwin_arm64/src/vibium_darwin_arm64/bin packages/python/vibium_win32_x64/src/vibium_win32_x64/bin
	cp clicker/bin/clicker-linux-amd64 packages/python/vibium_linux_x64/src/vibium_linux_x64/bin/clicker
	cp clicker/bin/clicker-linux-arm64 packages/python/vibium_linux_arm64/src/vibium_linux_arm64/bin/clicker
	cp clicker/bin/clicker-darwin-amd64 packages/python/vibium_darwin_x64/src/vibium_darwin_x64/bin/clicker
	cp clicker/bin/clicker-darwin-arm64 packages/python/vibium_darwin_arm64/src/vibium_darwin_arm64/bin/clicker
	cp clicker/bin/clicker-windows-amd64.exe packages/python/vibium_win32_x64/src/vibium_win32_x64/bin/clicker.exe
	@echo "Done. Python platform packages ready."

# Build all Python packages (wheels)
package-python: package-python-platforms
	@echo "Building Python wheels..."
	cd packages/python/vibium_darwin_arm64 && pip wheel . -w dist --no-deps
	cd packages/python/vibium_darwin_x64 && pip wheel . -w dist --no-deps
	cd packages/python/vibium_linux_x64 && pip wheel . -w dist --no-deps
	cd packages/python/vibium_linux_arm64 && pip wheel . -w dist --no-deps
	cd packages/python/vibium_win32_x64 && pip wheel . -w dist --no-deps
	cd clients/python && pip wheel . -w dist --no-deps
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
test: build test-cli test-js test-mcp

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
test-python: package-python-platforms
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
clean-bin:
	rm -rf clicker/bin

# Clean JS dist
clean-js:
	rm -rf clients/javascript/dist

# Clean built npm packages
clean-packages:
	rm -f packages/*/bin/clicker packages/*/bin/clicker.exe
	rm -rf packages/vibium/dist

# Clean built Python packages
clean-python-packages:
	rm -f packages/python/*/src/*/bin/clicker packages/python/*/src/*/bin/clicker.exe
	rm -rf packages/python/*/dist clients/python/dist

# Clean cached Chrome for Testing
clean-cache:
	rm -rf ~/Library/Caches/vibium/chrome-for-testing
	rm -rf ~/.cache/vibium/chrome-for-testing

# Clean everything (binaries + JS dist + packages + Python + cache)
clean-all: clean-bin clean-js clean-packages clean-python-packages clean-cache

# Alias for clean-bin + clean-js
clean: clean-bin clean-js

# Show available targets
help:
	@echo "Available targets:"
	@echo ""
	@echo "Build:"
	@echo "  make                    - Build everything (default)"
	@echo "  make build-go           - Build clicker binary"
	@echo "  make build-js           - Build JS client"
	@echo "  make build-all-platforms - Cross-compile clicker for all platforms"
	@echo ""
	@echo "Package (npm):"
	@echo "  make package            - Build all npm packages"
	@echo "  make package-platforms  - Build platform packages only"
	@echo "  make package-main       - Build main package only"
	@echo ""
	@echo "Package (Python):"
	@echo "  make package-python     - Build all Python wheels"
	@echo "  make package-python-platforms - Copy binaries to Python packages"
	@echo ""
	@echo "Test:"
	@echo "  make test               - Run all tests (CLI + JS + MCP)"
	@echo "  make test-cli           - Run CLI tests only"
	@echo "  make test-js            - Run JS library tests only"
	@echo "  make test-mcp           - Run MCP server tests only"
	@echo "  make test-python        - Run Python client tests"
	@echo ""
	@echo "Other:"
	@echo "  make install-browser    - Install Chrome for Testing"
	@echo "  make deps               - Install npm dependencies"
	@echo "  make serve              - Start proxy server on :9515"
	@echo "  make double-tap         - Kill zombie Chrome/chromedriver processes"
	@echo ""
	@echo "Clean:"
	@echo "  make clean              - Clean binaries and JS dist"
	@echo "  make clean-packages     - Clean built npm packages"
	@echo "  make clean-python-packages - Clean built Python packages"
	@echo "  make clean-cache        - Clean cached Chrome for Testing"
	@echo "  make clean-all          - Clean everything"
	@echo ""
	@echo "  make help               - Show this help"
