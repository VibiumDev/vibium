# File Run results to Linear

Task management is optional. Linear is the first provider. Vibium does not
grow a board. It files a signed issue from a recording, with absolute paths
to the files that created it.

## Setup

```bash
vibium setup tasks
```

That asks whether to configure Linear (default no on the full `vibium setup`).
A yes writes `~/.config/vibium/linear.env` at mode 0600:

- `VIBIUM_LINEAR_API_KEY` (or `LINEAR_API_KEY`)
- `VIBIUM_LINEAR_TEAM` (the prefix on `ENG-123`)

`vibium config init linear` writes the commented template. Non-interactive
setup skips Linear unless the file already exists, and never overwrites it.

## File a pack directory

A pack is a directory with `run.json` plus screenshots, maps, and the
recording zip. That is what a fleet loop should write.

```bash
vibium report linear ./out/pay
# Creates an issue. Title prefers a BROKEN evidence line.

vibium report linear ./out/pay --comment ENG-12
# Signed comment on an existing issue instead of creating.
```

Every create and comment ends with:

```markdown
## Signed by Vibium

Not a hand-written bug report. Opened from a Vibium recording.

- Engine: `vibium 2026.9.16`
- Binary: `/usr/local/bin/vibium`
- Command: `vibium report linear ./out/pay`
- Playbook: `walk`

### Artifacts

- `/abs/out/pay/run.json`
- `/abs/out/pay/run.zip`
- `/abs/out/pay/home.png`
```

The Linear actor is whoever owns the API key. The signature is how a reader
tells Vibium generated the ticket.

## From Run

```bash
vibium run "open every nav item" -o ./out/pay/run.zip --report linear
vibium run "open every nav item" -o ./out/pay/run.zip --report linear:ENG-12
```

`--report linear` creates. `--report linear:ENG-12` comments. Check still
uses `--report` for a JSON verdict file; use `vibium report linear` after
Check, or pass the pack directory.

Do not auto-mark Linear Done from `completed`. Do not file an issue when the
only BROKEN line is "none". Humans own Done.
