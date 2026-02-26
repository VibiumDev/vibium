# Process Cleanup

## Process Tree

When Vibium launches a browser session, it creates this process hierarchy:

```
vibium (Go binary)
  └── chromedriver
        └── Chrome for Testing (main browser process)
              ├── chrome_crashpad_handler (crash reporting)
              ├── GPU helper
              ├── Network helper
              ├── Storage helper
              ├── Renderer helper (one per tab/frame)
              └── ...more helpers
```

A single browser session spawns 8-12 processes.

## The Problem

When we kill chromedriver, its children (Chrome + helpers) get **reparented to PID 1** (launchd on macOS, init on Linux) before we can kill them. The process tree disintegrates mid-cleanup, leaving orphaned "zombie" processes.

```
Before kill:                    After chromedriver dies:
chromedriver                    (gone)
  └── Chrome                    Chrome (parent = 1, orphaned!)
        └── GPU helper            └── GPU helper
        └── Renderer              └── Renderer
```

### Why not kill Chrome first?

We try - `DELETE /session` tells chromedriver to quit Chrome gracefully. But this can fail:

- **Ctrl+C** - Signal kills chromedriver before DELETE completes
- **Timeout** - DELETE request fails or Chrome doesn't exit fast enough
- **Race** - Chromedriver dies before Chrome fully shuts down

A future improvement could find Chrome's PID and kill its tree explicitly before killing chromedriver.

## The Solution

Our cleanup strategy in `launcher.go:Close()`:

1. **DELETE /session** - Ask chromedriver to quit Chrome gracefully (best effort)
2. **Kill process tree** - Recursively find and kill all descendants using `pgrep -P`
3. **Kill orphans** - Sweep for any Chrome/chromedriver processes with parent PID 1 and kill them

```go
func killProcessTree(pid int) {
    descendants := getDescendants(pid)  // recursive pgrep -P
    // Kill children first (deepest first)
    for i := len(descendants) - 1; i >= 0; i-- {
        syscall.Kill(descendants[i], syscall.SIGKILL)
    }
    syscall.Kill(pid, syscall.SIGKILL)

    // Sweep for orphans that escaped
    killOrphanedChromeProcesses()
}
```

## What if vibium itself dies?

| Scenario | What happens |
|----------|--------------|
| Normal exit | `Close()` cleans up children |
| Ctrl+C | Signal handler calls `KillAll()` |
| `kill -9` | Orphans created, nothing we can do |

For `kill -9`, the orphan cleanup in the *next* session's `Close()` will find and kill them. Or use `make double-tap`.

The vibium binary itself doesn't leave zombies - it's just a Go process that exits cleanly. The issue is its children (chromedriver/Chrome) escaping cleanup.

## Manual Cleanup

If zombie processes accumulate during development:

```bash
make double-tap
```

This kills all Chrome for Testing and chromedriver processes.

## Debugging

Check for zombie processes:

```bash
pgrep -lf 'Chrome for Testing'
pgrep -lf chromedriver
```

Check parent PIDs to see if they're orphaned:

```bash
ps -o pid,ppid,comm -p $(pgrep -f 'Chrome for Testing')
```

Orphaned processes will have `PPID = 1`.
