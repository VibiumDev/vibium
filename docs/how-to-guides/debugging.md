# Debugging Guide

Low-level tools and troubleshooting tips for Vibium contributors.

## Verbose Logging

Add `-v` or `--verbose` to any vibium command to see debug output:

```bash
./clicker/bin/vibium go https://example.com -v
```

This shows BiDi protocol messages, timing info, and internal state.

## Dev Commands

These commands are for debugging and testing internals. They're not part of the public API.

### launch-test

Launch the selected browser and print its BiDi session information. Chrome
launches through chromedriver; Firefox uses its native BiDi endpoint:

```bash
./clicker/bin/vibium launch-test
# Prints the session ID and BiDi WebSocket URL.

./clicker/bin/vibium launch-test --engine firefox
# Prints the session ID; Firefox uses the native connection directly.
```

Useful for verifying that the selected browser can launch and accept a BiDi
session. Chrome also prints a WebSocket URL for manual testing.

### bidi-test

Launch browser, connect via BiDi, and send a `session.status` command:

```bash
./clicker/bin/vibium bidi-test
```

Verifies the full launch → connect → command pipeline works.

### ws-test

Interactive WebSocket tester. Connect to a URL and send/receive messages:

```bash
./clicker/bin/vibium ws-test ws://localhost:9222/...
```

Type JSON messages and see responses. Useful for debugging BiDi protocol issues.

### is actionable

Check all actionability conditions for an element:

```bash
./clicker/bin/vibium is actionable https://example.com "button"
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

If tests fail or you kill vibium mid-run, Chrome and chromedriver processes may linger:

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

1. Verify it's installed: `./clicker/bin/vibium paths`
2. Reinstall if needed: `./clicker/bin/vibium install`
3. On macOS, you may need to allow it in System Preferences → Security & Privacy

### Windows: "Access is denied. (0x5)" for Chrome for Testing

If Windows reports sandbox/access errors for `chrome.exe` under `%LOCALAPPDATA%\vibium\chrome-for-testing\`:

1. Delete `%LOCALAPPDATA%\vibium\chrome-for-testing\` and run `vibium install` again.
2. Add antivirus/endpoint-security exclusions for `%LOCALAPPDATA%\vibium\`.
3. Verify your user can execute files from that folder (no policy lock/block).
4. Re-run with debug logs enabled:
   - PowerShell: `$env:VIBIUM_DEBUG=1`
   - CMD: `set VIBIUM_DEBUG=1`

If this still fails, include the full debug output in your issue.

### Tests Hang

If tests hang indefinitely:

1. Run with verbose: `./clicker/bin/vibium go https://example.com -v`
2. Check for zombie processes: `make double-tap`
3. Check daemon status: `./clicker/bin/vibium daemon status`

## Inspecting BiDi Traffic

For deep debugging, run vibium with verbose mode and pipe to a file:

```bash
./clicker/bin/vibium go https://example.com -v 2>&1 | tee bidi.log
```

Search the log for `->` (sent) and `<-` (received) BiDi messages.
