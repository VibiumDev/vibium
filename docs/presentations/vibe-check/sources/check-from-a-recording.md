# Part 3: Check a recording or trace

**Check tutorials:** [Part 1: Live websites](first-check.md) · [Part 2: Coding agents](check-with-a-coding-agent.md) · Part 3

Use the `sitecheck.zip` from Part 1 to ask a fresh model about that recorded
run, then save its verdict in a report. You can run these commands yourself
or ask your coding agent to use the `check` skill.

## Before you start

Use the same Vibium binary and AI settings as Part 1. In a new terminal,
load the settings before running the check:

```bash
source ~/.config/vibium/ai.env
vibium ready ai
```

Wait for **READY**, then work in the directory containing `sitecheck.zip`.
If your settings file is elsewhere, source that file instead. No browser
needs to be open for this tutorial.

## 1. Check the saved run

```bash
vibium check "https://var.parts was up during the recorded run" -i sitecheck.zip
```

`-i` means **input**. Check reads evidence from the ZIP and returns **PASS**,
**FAIL**, or **INCONCLUSIVE**, with a short explanation. It still calls your
configured model, but it doesn't launch a browser, replay actions, or change
the original ZIP.

Read the evidence. This verdict concerns the recorded run, not whether the
site is up now. A recording's earlier PASS is an assessment the new model can
examine, not a result it must repeat.

## 2. Save the new verdict

Repeat the command with `--report`:

```bash
vibium check "https://var.parts was up during the recorded run" -i sitecheck.zip --report sitecheck-result.json
```

Open `sitecheck-result.json` in your editor. It contains `status`, `claim`,
`summary`, and `evidence`. Status is `passed`, `failed`, or `inconclusive`.
Choose a new report filename each time; existing files are not overwritten.

You now have the original browser recording and a separate assessment of it.
The report is JSON, not another recording. Use `-o` when making a new live
recording; it cannot be combined with `-i`.

## Optional: use a Playwright trace

The same input flag accepts a supported Playwright trace. For example, if
`trace.zip` contains a checkout run:

```bash
vibium check "the order confirmation was visible in the recorded run" -i trace.zip
```

Choose a claim relevant to your trace. Supported Vibium recordings and
Playwright traces use **trace format version 8**—the archive format version,
not the Playwright package version. The filename doesn't determine the format;
Vibium inspects the contents.

An unsupported format produces an execution error. A supported recording with
insufficient evidence may produce INCONCLUSIVE. A live check can gather new
observations; an archive check is limited to what was saved.

For other options and automation exit codes, see the
[Check reference](../reference/check.md).
