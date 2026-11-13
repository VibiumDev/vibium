# Codegen reference

Use this page to look up options, the action mapping, and limits.
For the workflow, see [Generate a script from a recording](../tutorials/recording.md#generate-a-script-from-a-recording).

## CLI options

```bash
vibium codegen -i <recording.zip> [options]
```

| Option | Purpose | Example |
|--------|---------|---------|
| `-i`, `--input` | Recording zip to read (required) | `vibium codegen -i record.zip` |
| `-l`, `--language` | Script language: `javascript` (default) or `python` | `vibium codegen -i record.zip -l python` |
| `-o`, `--output` | Save the script as a new file instead of printing it | `vibium codegen -i record.zip -o replay.mjs` |
| `--json` | Print the script in the CLI JSON envelope | `vibium codegen -i record.zip --json` |

Codegen reads the zip offline. It launches no browser and needs no daemon or
AI configuration. The input MUST be a trace format version 8 zip — a Vibium
recording or a Playwright trace. `--output` never replaces an existing file.
An unknown `--language` is an error.

JavaScript output uses top-level `await`, so it MUST run as an ES module:
save it as `.mjs` or run it in a `"type": "module"` package. Python output
uses the sync API and runs as a plain script.

## Exit codes

| Outcome | Exit code |
|---------|-----------|
| Script generated | 0 |
| Invalid input, unknown language, existing output file, or read failure | 1 |

## Action mapping

Recorded actions map to client calls. Element actions re-find their target
with the exact recorded locator — CSS selector or semantic keys (`role`,
`text`, `label`, `placeholder`, `alt`, `title`, `testid`, `xpath`, `near`).
A recorded nth-match index becomes `findAll(...)[n]` (`find_all(...)[n]`).
Consecutive actions on the same locator share one element variable.

| Recorded action | JavaScript | Python |
|-----------------|------------|--------|
| `Page.navigate` | `vibe.go(url)` | `vibe.go(url)` |
| `Page.reload` / `back` / `forward` | `vibe.reload()` … | `vibe.reload()` … |
| `Page.screenshot` | `vibe.screenshot()` | `vibe.screenshot()` |
| `Page.setViewport` | `vibe.setViewport({ width, height })` | `vibe.set_viewport({'width': …, 'height': …})` |
| `Page.eval` | `vibe.evaluate(expr)` | `vibe.evaluate(expr)` |
| `Page.setContent` | `vibe.setContent(html)` | `vibe.set_content(html)` |
| `Page.waitForURL` / `waitForLoad` / `waitForFunction` / `wait` | `vibe.waitForURL(pattern)` … | `vibe.wait_for_url(pattern)` … |
| `Page.scroll` | `vibe.scroll(direction, amount)` | `vibe.scroll(direction, amount)` |
| `Element.click` / `dblclick` / `hover` / `focus` / `tap` / `clear` / `unset` / `scrollIntoView` | `vibe.find(loc).click()` … | `vibe.find(loc).click()` … |
| `Element.fill` / `type` / `press` / `selectOption` | `vibe.find(loc).fill(value)` … | `vibe.find(loc).fill(value)` … |
| `Element.set` | `vibe.find(loc).set()` or `.set(false)` | `.set()` or `.set(False)` |
| `Element.dragTo` | `.dragTo(await vibe.find(target))` | `.drag_to(vibe.find(target))` |
| `Element.setFiles` | `.setFiles([...])` | `.set_files([...])` |
| `Element.dispatchEvent` | `.dispatchEvent(type, init)` | `.dispatch_event(type, init)` |
| `Element.waitFor` | `.waitUntil(state)` | `.wait_until(state)` |
| `Page.keyboard.*` / `mouse.*` / `touch.*` | `vibe.keyboard.press(key)` … | `vibe.keyboard.press(key)` … |
| Action group | `// group: <title>` | `# group: <title>` |

## Skipped events

- Pure queries change nothing on replay and are omitted: `Element.text`,
  `isVisible`, `getAttribute`, `Page.title`, `Page.url`, and the other
  getters, plus bare `find`/`findAll` calls.
- Noise and internal operations are omitted: `Element.highlight`,
  recording control, Run, and Check operations, and raw BiDi commands.
- An action whose recorded result was an error becomes a comment:
  `// skipped failed action: Element.click (<message>)`.
- An unmapped action becomes a comment: `// skipped unsupported action: …`.
- Screenshots, DOM snapshots, and network events carry no action to replay.

## Limitations

- The script drives one page. Recordings of multi-page flows replay every
  action on the first page.
- Recorded timeouts are dropped; replay uses the client defaults.
- Redacted values replay literally: a password recorded as `[REDACTED]`
  is typed as `[REDACTED]`. Edit the script before running it.
- Skipped getters mean the script replays actions, not assertions. Add your
  own checks, or verify the replay with
  [`vibium check`](../tutorials/check-from-a-recording.md).
