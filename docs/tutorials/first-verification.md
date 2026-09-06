# Part 1: Verify a live website

**Verify tutorials:** Part 1 · [Part 2: Coding agents](verification-with-a-coding-agent.md) · [Part 3: Recordings and traces](verification-from-a-recording.md)

Check whether a website is up, then save a recording of the check. You don't
need to write a script or use a coding agent.

## Before you start

You need [Vibium installed](../../README.md#agent-setup) and a
[configured verifier](../reference/verify.md#provider-configuration). Check
that you're ready:

```bash
vibium soundcheck
```

When it says **READY**, continue below. If it reports a problem, follow the fix
it gives you.

## 1. Check a website

```bash
vibium verify "https://pinthing.com is up"
```

A browser opens, the verifier checks the website, and the result appears in
your terminal. Here's the output from an example run:

```text
VERIFY: https://pinthing.com is up

PASS

https://pinthing.com is reachable and renders successfully, including after a reload.
- Navigation completed at https://pinthing.com/.
- The loaded page reported the title “PinThing Demo” and displayed rendered red 3D graphics.
- After reloading, the page loaded again with the title “PinThing Demo”.
```

**PASS** means the evidence supports your claim. **FAIL** means it contradicts
the claim. **INCONCLUSIVE** means the verifier couldn't decide. Your result
may differ as the website changes. This check covers availability, not every
feature on the site.

The browser closes when the check finishes. If you already had a Vibium
browser running, it stays open. Add `--keep-open` if you want a newly opened
browser to stay open too.

## 2. Save a recording

Repeat the check with `-o` and a filename:

```bash
vibium verify "https://pinthing.com is up" -o sitecheck.zip
```

You'll get the verdict and evidence again, followed by:

```text
Recording saved to sitecheck.zip
```

The ZIP contains the screenshots, browser actions, and result. Use a new
filename each time; Vibium won't overwrite an existing recording.

For **video**, install Firefox if needed, then run the same check with it:

```bash
vibium install --engine firefox
vibium verify "https://pinthing.com is up" -o sitecheck-firefox.zip --engine firefox
```

Firefox 154 or newer adds a WebM video to the recording. Chrome currently
saves the other evidence without a video track. Start from a closed browser
when switching engines; `vibium stop` closes an existing session.

## 3. See what happened

Open [Record Player](https://player.vibium.dev) and drop your ZIP onto it.
Select the **Verify** group to see the verdict and evidence, step through the
screenshots, or play the video when present.

Here's a screenshot from the Chrome check above:

![PinThing rendered during the verification](../images/first-verification-pinthing.jpg)

<details>
<summary>See the Firefox video from this example</summary>

In our Firefox run, the site responded, but the demo stayed blank because
the browser couldn't create a WebGL context. The verifier reported that in
its evidence. This video shows that run; the rendered screenshot above is
from Chrome.

![Video preview of the PinThing verification](../images/first-verification-pinthing.gif)

[Watch the original WebM video](../images/first-verification-pinthing.webm).

</details>

You now have a verdict you can read and a recording you can review or share.
Continue to [Part 2: Verify with your coding agent](verification-with-a-coding-agent.md),
or use the ZIP you just saved in
[Part 3: Verify a recording or trace](verification-from-a-recording.md).
