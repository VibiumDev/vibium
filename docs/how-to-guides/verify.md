# Verify a live browser or saved recording

`vibium verify` asks a fresh verifier to investigate a claim and return **PASS**,
**FAIL**, or **INCONCLUSIVE**, with evidence. Live checks use your existing local
Chrome or Firefox session.

This guide assumes Vibium and a verifier model are configured. For a first check,
follow the [first-verification tutorial](../tutorials/first-verification.md).
For other providers, see [configuration](../reference/verify.md#provider-configuration).

## Check setup

Run this in the same shell environment you will use for Verify:

```bash
vibium soundcheck
```

It lists missing settings and fixes, then tests the configured provider's
model and tool support. READY means those checks passed. It makes up to two
small model requests (API charges may apply) and leaves the browser alone.
Use `--json` for structured checks; exit 0 means ready and exit 1 means a setup
problem. If your settings are in an environment file, load it first—Vibium does
not load files automatically.

## Check the current browser

After exercising your application's flow, state the outcome you want checked:

```bash
vibium verify "the changed display name persists after refresh"
```

Keep the same terminal environment and `VIBIUM_SESSION` value used for the
browser workflow. The verifier can operate the page—for example, reload it to
check persistence—and may leave it changed.

A useful claim describes a result: “The changed name persists after refresh”
goes further than “Clicking Save shows a message.” Keep the claim faithful to
the requirement; split a broad requirement into separate checks if needed.

## Control whether the browser stays open

If Verify starts a browser, it closes that browser after saving any requested
recording. This applies to PASS, FAIL, INCONCLUSIVE, and execution errors.
If a browser was already running in the session, Verify leaves it open.

To keep a newly started browser open for inspection, add `--keep-open`:

```bash
vibium verify "https://example.com is up" -o status.zip --keep-open
```

Close it later with `vibium stop`. `--keep-open` only applies to live checks.

## Save evidence

Add `--output` to save a recording of the live check:

```bash
vibium verify "the changed display name persists after refresh" -o verification.zip
```

When no recording is active, this starts, saves, and stops one automatically.
If recording is already active, it exports the current recording chunk,
including earlier actions, and leaves recording active. Choose a new output
path. A recording started by Verify includes screenshots, browser actions, and
video when supported (Firefox 154+). An export from an already-active recording
contains the current chunk without its continuous video track.

To record the whole workflow, start recording before the browser actions:

```bash
vibium record start --name account-verification
# Exercise the account-page flow with Vibium.
vibium verify "the changed display name persists after refresh"
vibium record stop
```

Open the saved ZIP in [Record Player](https://player.vibium.dev). The **Verify**
group contains the verifier's actions and result. Sensitive form fields can
cause screenshots and other visuals to be omitted; use test data and review
recordings before sharing. See [recording privacy](../reference/verify.md#recording-privacy).

## Check a saved recording or trace

For a walkthrough using a ZIP from the tutorials, see
[Part 3: Verify a recording or trace](../tutorials/verification-from-a-recording.md).

Use `--input` to investigate evidence already in a ZIP:

```bash
vibium verify "the order confirmation is visible" -i record.zip
vibium verify "the order confirmation is visible" -i trace.zip
```

This reads the archive without launching a browser, replaying actions, or
changing the file. It supports Vibium recordings and Playwright trace format
version 8. The result describes the recorded run, not the current application.

Remember: **input reads a recording; output creates one.** They cannot be
combined. Use `--report` if you want to save the verdict from an input archive.

## Read and save the verdict

| Verdict | What it tells you | Next step |
|---------|-------------------|-----------|
| PASS | The verifier found evidence supporting the claim. | Review whether the evidence covers the intended behavior. |
| FAIL | The verifier found evidence contradicting the claim. | Investigate the failure, fix it, and rerun the same claim. |
| INCONCLUSIVE | The verifier could not decide from the available evidence. | Identify what evidence is missing before rerunning. |

A PASS is a second opinion, not proof of correctness. Keep deterministic tests
for stable, repeatable assertions. Verify is most useful when another browser
investigation adds evidence; it also adds model latency and possible API cost.

To save a JSON verdict, use `--report`. To print JSON, use `--json`:

```bash
vibium verify "the changed display name persists after refresh" --report verdict.json --json
```

**All three verdicts exit with code 0.** In JSON stdout, check
`result.status == "passed"` to require a PASS. In the report file, check the
top-level `status`. Execution errors exit 1 and have no verdict.

If a run errors, check the reported provider or browser problem. `--output`
saves available recording evidence even on an error. If you started recording
manually, use `vibium record stop` to save it.

For option details, limits, and MCP or language APIs, see the
[Verify reference](../reference/verify.md). For why the verifier uses a fresh
context and what that can achieve, read
[Why independent verification matters](../explanation/independent-verification.md).
