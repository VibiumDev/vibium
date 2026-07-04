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
vibium daemon v26.5.31
status:   running
pid:      40767
uptime:   2m10s
socket:   /Users/paul/Library/Caches/vibium/vibium-buyer.sock
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
