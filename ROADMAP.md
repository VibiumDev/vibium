# Vibium Roadmap

Things we'd like to work on. Priorities will follow what people need.

## Right now

- Make browser startup, shutdown, and error handling more reliable.
- Finish real-world testing with Anthropic, Gemini, and local models.
- Make setup smooth in Codex and Claude.
- Evaluate how well independent checks catch mistakes, using the same model or a different one.

## Find an element by description

The main remaining piece of AI-powered locators: ask for an element and get
one back. Something like `vibe.find("the blue submit button")`.

We still need to work out ambiguous matches and how prompts fit alongside
CSS selectors and semantic locators.

## Records

Help agents remember pages and paths between sessions, so they don't keep
rediscovering the same things.

If that proves useful, give people a way to browse and edit the memory too.
There's an [early UI prototype](https://vibium-cortex.lovable.app/?dataset=view-action-sample).

## More browsers

Edge and Safari, as people need them. Video recording in more browsers,
as their protocol support allows.

## A .NET client

Bring official C# support to Vibium. There's already an experimental
[community implementation](https://github.com/webdriverbidi-net/vibium-net)
to learn from.

## Easier CI and cloud setup

Container images and recipes for running Vibium in CI. Further out: hosted
Run and Check, cloud browsers for those operations, and PR integration.

## More recording formats

Support more Playwright trace versions. Explore saving a new Check's evidence
alongside an existing recording, while keeping the original intact.

## What's missing?

[Open an issue](https://github.com/VibiumDev/vibium/issues). Tell us what you're
trying to do and what's getting in the way.
