# Debugging Guide

Low-level tools and troubleshooting tips for Vibium contributors.

## Verbose Logging

Add `-v` or `--verbose` to any clicker command to see debug output:

```bash
./clicker/bin/clicker navigate https://example.com -v
```

This shows BiDi protocol messages, timing info, and internal state.

## Dev Commands

These commands are for debugging and testing internals. They're not part of the public API.

### launch-test

Launch browser via chromedriver and print the BiDi WebSocket URL:

```bash
./clicker/bin/clicker launch-test
# Output: ws://localhost:xxxxx/session/...
```

Useful for verifying chromedriver works and getting a WebSocket URL for manual testing.

### bidi-test

Launch browser, connect via BiDi, and send a `session.status` command:

```bash
./clicker/bin/clicker bidi-test
```

Verifies the full launch → connect → command pipeline works.

### ws-test

Interactive WebSocket tester. Connect to a URL and send/receive messages:

```bash
./clicker/bin/clicker ws-test ws://localhost:9222/...
```

Type JSON messages and see responses. Useful for debugging BiDi protocol issues.

### check-actionable

Check all actionability conditions for an element:

```bash
./clicker/bin/clicker check-actionable https://example.com "button"
# Output:
# Checking actionability for selector: button
# ✓ Visible: true
# ✓ Stable: true
# ✓ ReceivesEvents: true
# ✓ Enabled: true
# ✗ Editable: false
```

Useful when clicks or typing fail silently. Shows which condition isn't met.

## Troubleshooting

### Zombie Processes

If tests fail or you kill clicker mid-run, Chrome and chromedriver processes may linger:

```bash
make double-tap
```

This kills all `Chrome for Testing` and `chromedriver` processes.

### Connection Refused

If you see "Failed to connect to ws://localhost:9515":

1. Check if chromedriver is running: `ps aux | grep chromedriver`
2. Check if the port is in use: `lsof -i :9515`
3. Kill zombies and retry: `make double-tap`

### Chrome Won't Launch

If Chrome fails to start:

1. Verify it's installed: `./clicker/bin/clicker paths`
2. Reinstall if needed: `./clicker/bin/clicker install`
3. On macOS, you may need to allow it in System Preferences → Security & Privacy

### Tests Hang

If tests hang indefinitely:

1. Run with verbose: `./clicker/bin/clicker navigate https://example.com -v`
2. Check for zombie processes: `make double-tap`
3. Try with `--wait-close 5` to keep browser open and inspect state

## Inspecting BiDi Traffic

For deep debugging, run clicker with verbose mode and pipe to a file:

```bash
./clicker/bin/clicker navigate https://example.com -v 2>&1 | tee bidi.log
```

Search the log for `->` (sent) and `<-` (received) BiDi messages.
