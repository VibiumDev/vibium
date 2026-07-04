# CLI Multi-Session Daemon Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let multiple shell scripts / agents use the vibium CLI concurrently on one host by namespacing the daemon (socket, PID file, Windows pipe) per session via a `VIBIUM_SESSION` env var and a global `--session` flag.

**Architecture:** Today every one-shot CLI command proxies through a single per-host daemon (`vibium.sock` / `vibium.pid` / `\\.\pipe\vibium`) driving one browser, so concurrent users stomp on each other. The fix resolves a session name inside the `paths` package (same precedent as `VIBIUM_CACHE_DIR`): `GetSocketPath()`/`GetPIDPath()` gain a `-<session>` suffix when `VIBIUM_SESSION` is set. Because every daemon-touching code path (client dial, PID file, `CleanStale`, `IsRunning`, daemon `Run`) already goes through those two functions, no signatures change anywhere. A global `--session` flag bridges to the env var early via `os.Setenv`, which also makes auto-started daemon children inherit the session for free (`exec.Command` with nil `Env` inherits the parent environment).

**Tech Stack:** Go (stdlib only), cobra (existing dep), node:test for CLI integration tests (existing house style in `tests/daemon/`).

## Global Constraints

- **Version control is jj (Jujutsu), NEVER raw git.** Invoke the `jj-workflow` skill before the first jj command. Always pass `-m` (jj has no staging area, so there is no `add` step).
- jj commits the entire working copy. Commit this plan file as its own change (`jj commit -m "docs: plan for CLI multi-session daemon"`) before starting Task 1, and keep each task's edits isolated to that task's commit.
- Go stdlib only; no new dependencies.
- Do not export anything from a package unless another package needs it.
- Backward compatible: with no session set, paths are byte-for-byte what they are today (`vibium.sock`, `vibium.pid`, `\\.\pipe\vibium`).
- Session names must be safe for filenames and Windows pipe names: `^[A-Za-z0-9_-]{1,64}$`.
- Per project CLAUDE.md: new CLI options need a simple example and sample output.
- Comments are evergreen — no "new", "now supports", or references to the previous behavior.

## File Structure

| File | Action | Responsibility |
|---|---|---|
| `clicker/internal/paths/paths.go` | Modify | Resolve + validate session name; session-suffixed socket/PID/pipe paths |
| `clicker/internal/paths/paths_test.go` | Create | Unit tests for session path resolution and validation |
| `clicker/cmd/clicker/main.go` | Modify | Global `--session` flag, bridged to `VIBIUM_SESSION`, validated in `PersistentPreRunE` |
| `clicker/cmd/clicker/daemon_cmd.go` | Modify | Session examples in help text; session line in `daemon status` output |
| `clicker/internal/daemon/router.go` | Modify | `Session` field in `StatusResult` |
| `tests/daemon/sessions.test.js` | Create | Integration tests: two concurrent sessions are isolated |
| `Makefile` | Modify | Add sessions test to `test-daemon` target |
| `docs/how-to-guides/concurrent-cli-sessions.md` | Create | How-to guide with examples and sample output |

---

### Task 1: Session-aware paths in the `paths` package

**Files:**
- Modify: `clicker/internal/paths/paths.go` (functions `GetSocketPath` at line 170, `GetPIDPath` at line 182; imports at line 3)
- Test: `clicker/internal/paths/paths_test.go` (create)

**Interfaces:**
- Consumes: nothing new.
- Produces (used by Tasks 2 and 3):
  - `paths.SessionName() string` — returns `VIBIUM_SESSION`, empty = default session.
  - `paths.ValidateSessionName(name string) error` — nil for `""` or `^[A-Za-z0-9_-]{1,64}$`.
  - `GetSocketPath() (string, error)` / `GetPIDPath() (string, error)` — unchanged signatures, session-suffixed results, error on invalid session name.

- [ ] **Step 1: Write the failing tests**

Create `clicker/internal/paths/paths_test.go`:

```go
package paths

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestDefaultSessionPaths(t *testing.T) {
	t.Setenv("VIBIUM_SESSION", "")

	sock, err := GetSocketPath()
	if err != nil {
		t.Fatalf("GetSocketPath: %v", err)
	}
	if runtime.GOOS == "windows" {
		if sock != `\\.\pipe\vibium` {
			t.Errorf("socket = %q, want \\\\.\\pipe\\vibium", sock)
		}
	} else if filepath.Base(sock) != "vibium.sock" {
		t.Errorf("socket basename = %q, want vibium.sock", filepath.Base(sock))
	}

	pid, err := GetPIDPath()
	if err != nil {
		t.Fatalf("GetPIDPath: %v", err)
	}
	if filepath.Base(pid) != "vibium.pid" {
		t.Errorf("pid basename = %q, want vibium.pid", filepath.Base(pid))
	}
}

func TestNamedSessionPaths(t *testing.T) {
	t.Setenv("VIBIUM_SESSION", "projA")

	sock, err := GetSocketPath()
	if err != nil {
		t.Fatalf("GetSocketPath: %v", err)
	}
	if runtime.GOOS == "windows" {
		if sock != `\\.\pipe\vibium-projA` {
			t.Errorf("socket = %q, want \\\\.\\pipe\\vibium-projA", sock)
		}
	} else if filepath.Base(sock) != "vibium-projA.sock" {
		t.Errorf("socket basename = %q, want vibium-projA.sock", filepath.Base(sock))
	}

	pid, err := GetPIDPath()
	if err != nil {
		t.Fatalf("GetPIDPath: %v", err)
	}
	if filepath.Base(pid) != "vibium-projA.pid" {
		t.Errorf("pid basename = %q, want vibium-projA.pid", filepath.Base(pid))
	}
}

func TestInvalidSessionNameRejected(t *testing.T) {
	t.Setenv("VIBIUM_SESSION", "bad/name")

	if _, err := GetSocketPath(); err == nil {
		t.Error("GetSocketPath: want error for session name with slash")
	}
	if _, err := GetPIDPath(); err == nil {
		t.Error("GetPIDPath: want error for session name with slash")
	}
}

func TestValidateSessionName(t *testing.T) {
	valid := []string{"", "a", "projA", "proj-a_1", strings.Repeat("x", 64)}
	for _, name := range valid {
		if err := ValidateSessionName(name); err != nil {
			t.Errorf("ValidateSessionName(%q) = %v, want nil", name, err)
		}
	}

	invalid := []string{"bad/name", "a b", "a.b", `a\b`, strings.Repeat("x", 65)}
	for _, name := range invalid {
		if err := ValidateSessionName(name); err == nil {
			t.Errorf("ValidateSessionName(%q) = nil, want error", name)
		}
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd clicker && go test ./internal/paths/`
Expected: FAIL to compile with `undefined: ValidateSessionName`

- [ ] **Step 3: Implement session resolution in paths.go**

In `clicker/internal/paths/paths.go`, change the import block (line 3) to:

```go
import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
)
```

Insert before `GetSocketPath` (currently line 167):

```go
// SessionName returns the daemon session name from the VIBIUM_SESSION
// environment variable. Empty means the default (shared) session.
// Named sessions get their own daemon socket, PID file, and browser,
// so concurrent CLI users on one host stay isolated.
func SessionName() string {
	return os.Getenv("VIBIUM_SESSION")
}

var sessionNameRe = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

// ValidateSessionName checks that a session name is safe to embed in
// socket filenames and Windows named-pipe names.
func ValidateSessionName(name string) error {
	if name == "" {
		return nil
	}
	if !sessionNameRe.MatchString(name) {
		return fmt.Errorf("invalid session name %q: use only letters, digits, '-' and '_' (max 64 chars)", name)
	}
	return nil
}

// sessionSuffix returns "-<name>" for a named session, "" for the default.
func sessionSuffix() (string, error) {
	name := SessionName()
	if err := ValidateSessionName(name); err != nil {
		return "", err
	}
	if name == "" {
		return "", nil
	}
	return "-" + name, nil
}
```

Replace `GetSocketPath` (lines 167–179) with:

```go
// GetSocketPath returns the platform-specific socket path for the daemon.
// macOS/Linux: ~/Library/Caches/vibium/vibium[-<session>].sock or ~/.cache/vibium/vibium[-<session>].sock
// Windows: \\.\pipe\vibium[-<session>] (named pipe)
func GetSocketPath() (string, error) {
	suffix, err := sessionSuffix()
	if err != nil {
		return "", err
	}
	if runtime.GOOS == "windows" {
		return `\\.\pipe\vibium` + suffix, nil
	}
	dir, err := GetDaemonDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "vibium"+suffix+".sock"), nil
}
```

Replace `GetPIDPath` (lines 181–188) with:

```go
// GetPIDPath returns the path to the daemon PID file.
func GetPIDPath() (string, error) {
	suffix, err := sessionSuffix()
	if err != nil {
		return "", err
	}
	dir, err := GetDaemonDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "vibium"+suffix+".pid"), nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd clicker && go test ./internal/paths/`
Expected: PASS (4 tests)

- [ ] **Step 5: Verify nothing else broke**

Run: `cd clicker && go build ./... && go test ./...`
Expected: builds cleanly, all existing tests PASS (every caller of `GetSocketPath`/`GetPIDPath` — `daemon/client.go:100`, `daemon/pidfile.go`, `daemon/status.go:22`, `daemon/daemon.go:58`, `cmd/clicker/daemon_cmd.go`, `cmd/clicker/daemon_client.go:75` — uses the unchanged signatures).

- [ ] **Step 6: Commit**

```bash
jj commit -m "feat(paths): namespace daemon socket and PID paths by VIBIUM_SESSION"
```

---

### Task 2: Global `--session` flag

**Files:**
- Modify: `clicker/cmd/clicker/main.go` (globals at line 31, root command at line 40, flags at line 55)
- Modify: `clicker/cmd/clicker/daemon_cmd.go` (Example block at lines 48–58)

**Interfaces:**
- Consumes: `paths.SessionName()`, `paths.ValidateSessionName(string)` from Task 1.
- Produces: `--session <name>` persistent flag on the root command. Effect: sets `VIBIUM_SESSION` process-wide before any command runs, so the `paths` package and any auto-started daemon child (which inherits the environment via `exec.Command` with nil `Env` in `daemon_client.go:64` and `daemon_cmd.go:265`) resolve the same session. No changes needed in `daemon_client.go` or `daemon_cmd.go` spawn logic.

- [ ] **Step 1: Add the flag and validation**

In `clicker/cmd/clicker/main.go`, add `"github.com/vibium/clicker/internal/paths"` to the imports, and extend the globals block (lines 31–35) to:

```go
// Global flags
var (
	headless   bool
	verbose    bool
	jsonOutput bool
	session    string
)
```

Replace the root command's `PersistentPreRun` (lines 43–48) with a `PersistentPreRunE`:

```go
PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
	// Enable logging only if --verbose is used
	if verbose {
		log.Setup(log.LevelVerbose)
	}
	// Bridge the flag to the env var so the paths package and any
	// auto-started daemon child process resolve the same session.
	if session != "" {
		if err := os.Setenv("VIBIUM_SESSION", session); err != nil {
			return err
		}
	}
	return paths.ValidateSessionName(paths.SessionName())
},
```

After the existing persistent flags (line 57), add:

```go
rootCmd.PersistentFlags().StringVar(&session, "session", "", "Named daemon session for isolated concurrent use (env: VIBIUM_SESSION)")
```

- [ ] **Step 2: Add help examples (project rule: new CLI options need an example)**

In `clicker/cmd/clicker/daemon_cmd.go`, extend the `Example` block of `newDaemonStartCmd` (lines 48–58) by appending:

```go
	Example: `  vibium daemon start
  # Starts daemon in background

  vibium daemon start --foreground
  # Starts daemon in foreground (for debugging)

  vibium daemon start --idle-timeout 30m
  # Auto-shutdown after 30 minutes of inactivity

  vibium daemon start --connect ws://remote:9515/session
  # Connect to a remote browser instead of launching a local one

  vibium --session projA daemon start
  # Isolated session "projA": own daemon, own browser, socket
  # vibium-projA.sock; VIBIUM_SESSION=projA does the same`,
```

- [ ] **Step 3: Build and verify by hand**

Run: `make build-go`
Expected: builds cleanly.

Run: `./clicker/bin/vibium --session 'bad/name' daemon status`
Expected: exits non-zero with `invalid session name "bad/name": use only letters, digits, '-' and '_' (max 64 chars)`

Run: `./clicker/bin/vibium --session demo daemon status`
Expected: `Daemon is not running.` (exit 0 — flag accepted, no daemon started)

Run: `./clicker/bin/vibium daemon status`
Expected: same output as before this change (default session untouched).

- [ ] **Step 4: Commit**

```bash
jj commit -m "feat(cli): add --session flag for isolated concurrent daemon sessions"
```

---

### Task 3: Report the session in `daemon status`

**Files:**
- Modify: `clicker/internal/daemon/router.go` (`StatusResult` at lines 15–22, `handleStatus` at lines 134–143, imports at line 3)
- Modify: `clicker/cmd/clicker/daemon_cmd.go` (`newDaemonStatusCmd` output at lines 138–153)

**Interfaces:**
- Consumes: `paths.SessionName()` from Task 1.
- Produces: `StatusResult.Session string` with JSON key `"session"` — Task 4's integration test asserts on it. The field round-trips to the CLI through the existing `daemon.Status()` JSON unmarshal in `client.go:61-82` with no client changes.

- [ ] **Step 1: Add the field and populate it**

In `clicker/internal/daemon/router.go`, add `"github.com/vibium/clicker/internal/paths"` to the imports. Extend `StatusResult`:

```go
// StatusResult is returned by daemon/status.
type StatusResult struct {
	Version   string `json:"version"`
	PID       int    `json:"pid"`
	Uptime    string `json:"uptime"`
	Socket    string `json:"socket"`
	StartTime string `json:"startTime"`
	Session   string `json:"session"`
}
```

Extend `handleStatus`:

```go
// handleStatus returns daemon status information.
func (d *Daemon) handleStatus() (interface{}, *agent.Error) {
	return StatusResult{
		Version:   d.version,
		PID:       pidSelf(),
		Uptime:    time.Since(d.startTime).Truncate(time.Second).String(),
		Socket:    d.socketPath,
		StartTime: d.startTime.Format(time.RFC3339),
		Session:   paths.SessionName(),
	}, nil
}
```

- [ ] **Step 2: Show it in the CLI status output**

In `clicker/cmd/clicker/daemon_cmd.go`, `newDaemonStatusCmd`: add `"session"` to the JSON map (after `"socket"` at line 144):

```go
			printJSON(map[string]interface{}{
				"running": true,
				"version": status.Version,
				"pid":     status.PID,
				"uptime":  status.Uptime,
				"socket":  status.Socket,
				"session": status.Session,
			})
```

And after the `socket:` line of the text output (line 153), add:

```go
		if status.Session != "" {
			fmt.Printf("session:  %s\n", status.Session)
		}
```

- [ ] **Step 3: Build and verify by hand**

Run: `make build-go && ./clicker/bin/vibium --session demo daemon start --headless && ./clicker/bin/vibium --session demo daemon status`
Expected output (pid/uptime/socket-prefix vary):

```
vibium daemon v0.0.0-dev
status:   running
pid:      12345
uptime:   1s
socket:   /Users/you/Library/Caches/vibium/vibium-demo.sock
session:  demo
```

Run: `./clicker/bin/vibium --session demo daemon status --json`
Expected: JSON object containing `"session": "demo"` and a socket path ending in `vibium-demo.sock`.

Run: `./clicker/bin/vibium --session demo daemon stop`
Expected: `Daemon stopped.`

- [ ] **Step 4: Commit**

```bash
jj commit -m "feat(daemon): report session name in daemon status"
```

---

### Task 4: Integration test — concurrent sessions are isolated

**Files:**
- Create: `tests/daemon/sessions.test.js`
- Modify: `Makefile` (`test-daemon` target, line 274)

**Interfaces:**
- Consumes: `--session` flag (Task 2), `VIBIUM_SESSION` env var (Task 1), `"session"` field in status JSON (Task 3), `VIBIUM` binary path from `tests/helpers.js`.
- Produces: nothing consumed later; this is the automated proof of isolation.

- [ ] **Step 1: Write the test file**

Create `tests/daemon/sessions.test.js`:

```js
/**
 * Daemon Session Tests
 * Tests VIBIUM_SESSION / --session isolation: separate daemons, sockets, PIDs
 */

const { test, describe, before, after } = require('node:test');
const assert = require('node:assert');
const { execSync } = require('node:child_process');
const { VIBIUM } = require('../helpers');

function clicker(args, opts = {}) {
  const result = execSync(`${VIBIUM} ${args}`, {
    encoding: 'utf-8',
    timeout: opts.timeout || 60000,
    env: { ...process.env, ...opts.env },
  });
  return result.trim();
}

function clickerJSON(args, opts = {}) {
  return JSON.parse(clicker(`${args} --json`, opts));
}

// Helper to stop a session's daemon (ignore errors if not running)
function stopDaemon(session) {
  try {
    const flag = session ? `--session ${session} ` : '';
    execSync(`${VIBIUM} ${flag}daemon stop`, { encoding: 'utf-8', timeout: 10000 });
  } catch (e) {
    // Daemon may not be running
  }
}

function stopAll() {
  stopDaemon();
  stopDaemon('s1');
  stopDaemon('s2');
}

describe('Daemon: Named sessions', () => {
  before(stopAll);
  after(stopAll);

  test('two sessions run separate daemons concurrently', () => {
    clicker('--session s1 daemon start --headless');
    clicker('--session s2 daemon start --headless');

    const s1 = clickerJSON('--session s1 daemon status');
    const s2 = clickerJSON('--session s2 daemon status');

    assert.strictEqual(s1.running, true, 's1 should be running');
    assert.strictEqual(s2.running, true, 's2 should be running');
    assert.strictEqual(s1.session, 's1', 'status should name session s1');
    assert.strictEqual(s2.session, 's2', 'status should name session s2');
    assert.notStrictEqual(s1.pid, s2.pid, 'sessions should be separate processes');
    assert.notStrictEqual(s1.socket, s2.socket, 'sessions should use separate sockets');

    const def = clicker('daemon status');
    assert.match(def, /not running/i, 'default session should be unaffected');
  });

  test('stopping one session leaves the other running', () => {
    clicker('--session s1 daemon stop');

    const s1 = clicker('--session s1 daemon status');
    assert.match(s1, /not running/i, 's1 should be stopped');

    const s2 = clickerJSON('--session s2 daemon status');
    assert.strictEqual(s2.running, true, 's2 should still be running');
  });

  test('VIBIUM_SESSION env var selects the session', () => {
    const s2 = clickerJSON('daemon status', { env: { VIBIUM_SESSION: 's2' } });
    assert.strictEqual(s2.running, true, 'env var should route to the s2 daemon');
    assert.strictEqual(s2.session, 's2');
  });

  test('auto-start uses the named session', () => {
    const nav = clickerJSON('go https://example.com --headless', {
      env: { VIBIUM_SESSION: 's1' },
    });
    assert.strictEqual(nav.ok, true, 'navigate should auto-start the s1 daemon');

    const s1 = clickerJSON('--session s1 daemon status');
    assert.strictEqual(s1.running, true, 's1 should be running after auto-start');

    const def = clicker('daemon status');
    assert.match(def, /not running/i, 'default session should be unaffected');
  });

  test('invalid session name is rejected', () => {
    assert.throws(
      () => clicker('--session bad/name daemon status'),
      /invalid session name/i,
      'should reject unsafe session names'
    );
  });
});
```

- [ ] **Step 2: Run the test to verify it passes**

Run: `make build-go && node --test --test-concurrency=1 tests/daemon/sessions.test.js`
Expected: PASS (5 tests). (If run before Tasks 1–3 are in, it fails on the unknown `--session` flag — that ordering confirms the test bites.)

- [ ] **Step 3: Wire it into the Makefile**

In `Makefile` line 274, append `tests/daemon/sessions.test.js` to the `test-daemon` file list:

```make
test-daemon: build-go
	@echo "--- Daemon Tests ---"
	$(TIMEOUT_CMD) node --test $(TEST_FLAGS) --test-concurrency=1 tests/daemon/lifecycle.test.js tests/daemon/concurrency.test.js tests/daemon/cli-commands.test.js tests/daemon/find-refs.test.js tests/daemon/connect.test.js tests/daemon/recording.test.js tests/daemon/sessions.test.js
```

- [ ] **Step 4: Run the full daemon suite**

Run: `make test-daemon`
Expected: all daemon tests PASS, including the pre-existing lifecycle/concurrency tests (they use the default session, which named sessions must not disturb).

- [ ] **Step 5: Commit**

```bash
jj commit -m "test(daemon): verify named sessions isolate concurrent CLI use"
```

---

### Task 5: How-to guide

**Files:**
- Create: `docs/how-to-guides/concurrent-cli-sessions.md`

**Interfaces:**
- Consumes: the flag/env var and status output from Tasks 1–3.
- Produces: user-facing documentation (project rule: new CLI options ship with example + sample output).

- [ ] **Step 1: Write the guide**

Create `docs/how-to-guides/concurrent-cli-sessions.md`:

```markdown
# Running Concurrent CLI Sessions

One-shot CLI commands (`vibium go`, `vibium click`, `vibium find`, ...) share a
background daemon that keeps one browser alive between commands. By default
there is one daemon per host, so two scripts driving the CLI at the same time
would share one browser and interfere with each other.

Named sessions give each script its own daemon and browser.

## Use a session per script

Set `VIBIUM_SESSION` once at the top of a script:

```bash
#!/bin/sh
export VIBIUM_SESSION=checkout-tests

vibium go https://staging.example.com/checkout
vibium fill "#email" "test@example.com"
vibium click "button[type=submit]"

vibium daemon stop   # shut down this session's browser when done
```

Or pass `--session` per command — useful when one script drives two browsers:

```bash
vibium --session buyer go https://shop.example.com
vibium --session seller go https://shop.example.com/admin
```

Both forms are equivalent; the flag takes precedence over the environment.
Session names may use letters, digits, `-` and `_` (max 64 chars).

## Inspecting sessions

Each session has its own daemon, socket, PID file, and idle timeout:

```console
$ vibium --session buyer daemon status
vibium daemon v26.2.0
status:   running
pid:      41823
uptime:   2m10s
socket:   ~/Library/Caches/vibium/vibium-buyer.sock
session:  buyer
```

Commands without a session use the default daemon, which is completely
unaffected by named sessions:

```console
$ vibium daemon status
Daemon is not running.
```

## Cleanup

Sessions auto-shutdown after 30 minutes idle (tune with
`vibium --session <name> daemon start --idle-timeout 5m`), or stop them
explicitly with `vibium --session <name> daemon stop`.
```

- [ ] **Step 2: Verify the sample output matches reality**

Run: `./clicker/bin/vibium --session buyer daemon start --headless && ./clicker/bin/vibium --session buyer daemon status && ./clicker/bin/vibium --session buyer daemon stop`
Expected: text output has the same fields and shapes as the guide's sample (values differ). Fix the guide if the format drifted.

- [ ] **Step 3: Commit**

```bash
jj commit -m "docs: how-to guide for concurrent CLI sessions"
```

---

### Task 6: Full verification

**Files:** none (verification only).

- [ ] **Step 1: Go unit tests**

Run: `cd clicker && go test ./...`
Expected: PASS.

- [ ] **Step 2: Daemon and CLI integration suites**

Run: `make test-daemon && make test-cli`
Expected: PASS.

- [ ] **Step 3: Full test suite (project rule before finishing)**

Run: `make test`
Expected: PASS (slow — runs JS, MCP, Python, and Java suites with real browsers).

- [ ] **Step 4: Confirm no stray session daemons or files**

Run: `ls ~/Library/Caches/vibium/ | grep -E 'vibium.*\.(sock|pid)' || echo clean`
Expected: `clean` (or only the default `vibium.sock`/`vibium.pid` if a default daemon is intentionally running).
