# Part 3: Verify a recording or trace

**Verify tutorials:** [Part 1: Live websites](first-verification.md) · [Part 2: Coding agents](verification-with-a-coding-agent.md) · Part 3

Check a claim against a saved browser run. You'll use the `sitecheck.zip`
from Part 1, then save the new verdict in a report. You can do this yourself
or ask your coding agent to run the same commands.

## Before you start

Use the same Vibium installation and verifier settings as before. Run
`vibium soundcheck` in your terminal and wait for **READY**.

Have your `sitecheck.zip` available. If you have a different Vibium recording
or Playwright trace, use its filename and a claim about what that run should
show. Playwright traces must use **trace format version 8**; this is the
archive format, not the Playwright package version.

## 1. Check the saved run

In the folder containing `sitecheck.zip`, run:

```bash
vibium verify "https://pinthing.com was up during the recorded run" -i sitecheck.zip
```

`-i` means **input**. The verifier reads the recording's evidence and returns
**PASS**, **FAIL**, or **INCONCLUSIVE**, with a short explanation. It doesn't
open a browser, replay the actions, or change the ZIP. It still calls your
configured verifier model.

Read the evidence alongside the verdict. The result describes what happened
when the recording was made; it cannot tell you whether the site is up now.
Even if the recording contains an earlier PASS, the new verifier must assess
the evidence for your claim.

## 2. Try another recording or trace

The same flag works for a Vibium recording named `record.zip` or a Playwright
trace named `trace.zip`. For example, if your saved run includes an order
confirmation, choose the command matching your file:

```bash
vibium verify "the order confirmation was visible in the recorded run" -i record.zip
```

```bash
vibium verify "the order confirmation was visible in the recorded run" -i trace.zip
```

The filename doesn't select the format; Vibium checks the archive contents.
An unsupported trace format produces an error. If a supported archive lacks
the screenshots or other evidence needed for the claim, the verifier may
return **INCONCLUSIVE**. A new live check can gather evidence that the saved
run is missing.

## 3. Save the new verdict

Repeat your PinThing check with `--report`:

```bash
vibium verify "https://pinthing.com was up during the recorded run" -i sitecheck.zip --report sitecheck-result.json
```

Open `sitecheck-result.json` to see the status, claim, summary, and evidence.
Choose a new report filename each time; Vibium won't overwrite an existing
file. The original ZIP remains unchanged.

Use **`-i` to read a recording**, **`-o` to create a live recording**, and
**`--report` to save a verdict**. You can't combine `-i` and `-o`.

Keep the [how-to guide](../how-to-guides/verify.md)
handy for everyday commands, or read
[why independent verification matters](../explanation/independent-verification.md)
for its benefits and limits.
