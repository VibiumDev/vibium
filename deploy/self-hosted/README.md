# Self-hosted browsers

The industry axis is **managed vs. self-hosted**. Managed browser
clouds (Sauce Labs, BrowserStack, TestMu, Kernel, …) sell browsers as
a service: you get a session URL and never a shell. Self-hosting is
the alternative: root on a machine you control — a rented cloud VM or
on-premises hardware like the Mac mini on your desk — with the
browser layer yours to run.

Standing one up turns out to be the same three steps everywhere:
install a version-matched Chrome + chromedriver, keep the port off
the public internet, and point vibium at it. This directory factors
the work along those lines — one shared installer that prepares any
Debian/Ubuntu machine ([`setup-chrome.sh`](setup-chrome.sh), detailed
below), and a short recipe per platform for creating the machine and
reaching it:

| Platform | Cold start | Best for |
|---|---|---|
| [exe.dev](exe-dev/) | ~2s VM boot | Everything-cloud: agent + vibium + Chrome on one VM |
| [Fly.io](flyio/) | seconds (sub-second restart) | Parallel fleets: `fleet.sh up 25`, per-machine DNS, ~$0 stopped |
| [DigitalOcean](digitalocean/) | ~1 min (post-snapshot) | Simple, predictable droplets |
| [Hetzner](hetzner/) | ~1 min (post-snapshot) | Budget instances; dedicated bare metal available |
| [GCP](gcp/) | ~1 min (post-image) | You already live on Google Cloud; gcloud-managed tunnel |
| [AWS](aws/) | ~1 min (post-AMI) | You already live in AWS — several shapes, see its page |

DigitalOcean, Hetzner, and GCP share one
[`cloud-init.yml`](cloud-init.yml); it runs unchanged on most other
clouds that accept user-data (Azure, Vultr, Linode, …), which is why
those don't get pages of their own — they add nothing the six above
don't have. Oracle's always-free ARM tier becomes interesting the day
Chrome for Testing ships linux-arm64.

## What setup-chrome.sh does

The shared installer turns a bare Debian/Ubuntu x86-64 box into a
self-hosted browser, idempotently:

1. **Installs Chrome's library dependencies** via apt (the NSS/GTK/X11
   libraries and fonts a headless Chrome still links against — the usual
   reason a copied Chrome binary won't start on a fresh server).
2. **Resolves the latest stable Chrome for Testing + the matching
   chromedriver** from Google's `last-known-good-versions` feed, so the
   browser and driver versions can never drift apart, and unpacks both
   under `/opt/chrome-for-testing` with symlinks at
   `/usr/local/bin/chrome` and `/usr/local/bin/chromedriver`.
3. **Creates a system user `vibium`** and runs everything as it — Chrome
   refuses to run as root, and a browser is a remote-code-execution
   surface that shouldn't have root anyway.
4. **With `--service`**: installs and starts a systemd unit
   (`chromedriver.service`) listening on `127.0.0.1:9515`, restart-on-
   crash. Loopback-only on purpose: reach it through an SSH tunnel, or
   edit the unit to add `--allowed-ips=""` when your platform provides
   the network isolation (the Fly recipe's Dockerfile does exactly that).

x86-64 Linux only for now — Chrome for Testing doesn't ship linux-arm64
builds yet; the script gains ARM support the week Google's feed does.

## Connecting

Every platform converges on the same connect step, because an http(s)
URL to chromedriver is a classic WebDriver endpoint vibium speaks
natively:

```bash
vibium start http://127.0.0.1:9515   # through the tunnel
```

Security model: chromedriver has no authentication, so nothing here
exposes it publicly. Reachability is an SSH tunnel (`ssh -L`) or the
platform's private network (`fly proxy`). Browser automation is remote
code execution — treat the port accordingly.
