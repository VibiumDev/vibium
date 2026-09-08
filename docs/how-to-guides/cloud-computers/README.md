# Browsers on cloud computers

There are two ways to get a browser in the cloud. You can rent the
**browser**: managed vendors (Sauce Labs, BrowserStack, TestMu
(LambdaTest), Kernel, …) sell browsers as a service — a session URL,
never a shell.
Or you can rent the **computer** — a machine you get root on — and
run the browser layer yourself; the industry calls this self-hosting.
These recipes are the second path: each turns a rented computer into
a browser server that vibium drives.

## What you'll need

- **vibium on your dev machine** (`npm install -g vibium`). That's the
  client side — nothing else installs locally.
- **An account on one platform** from the table below, with its CLI
  installed and logged in (`doctl`, `hcloud`, `gcloud`, `fly`, or the
  AWS CLI — each recipe's "Prereqs" section lists the exact steps and
  whether a payment method is required up front).
- **An SSH keypair** on most platforms, for the tunnel; Fly and GCP
  manage access through their own CLIs instead.

Budget ~10–30 minutes for the first machine, most of it waiting on
Chrome to install; after snapshotting, subsequent machines start in
about a minute.

## What's provided here

- [`install-chrome.sh`](../../../deploy/cloud-computers/install-chrome.sh) — one installer that turns any
  Debian/Ubuntu x86-64 machine into a browser server (detailed below).
- [`cloud-init.yml`](../../../deploy/cloud-computers/cloud-init.yml) — a first-boot wrapper for that
  installer. Most VM clouds accept a first-boot configuration file
  (the cloud-init "user-data" standard); this one just fetches and
  runs the installer, so machine creation and setup collapse into one
  command.
- **A recipe per platform** — the create, connect, and teardown
  commands, with each platform's own footguns called out (billing that
  survives power-off, image choices, quota walls).
- [`flyio/fleet.sh`](../../../deploy/cloud-computers/flyio/fleet.sh) — start/stop/address N parallel
  browsers on Fly.

## How it fits together

1. **Create the machine** with your recipe's create command. On
   DigitalOcean, Hetzner, and GCP the cloud-init file runs the
   installer automatically; on exe.dev you run it over SSH; on Fly it's
   baked into the Docker image.
2. The installer leaves **chromedriver running as a service on the
   machine's loopback**, port 9515. Nothing is exposed to the internet.
3. **Open a tunnel** from your dev machine — `ssh -N -L
   9515:127.0.0.1:9515 <user>@<machine>` on the SSH platforms,
   `fly proxy 9515:9515` on Fly. Port 9515 on localhost is now the
   remote browser.
4. **Point vibium at it**:

   ```bash
   vibium start http://127.0.0.1:9515
   vibium go https://example.com
   vibium title
   vibium stop
   ```

   An http(s) URL is a classic WebDriver endpoint, which vibium speaks
   natively — it creates the session and talks WebDriver BiDi through
   it. From here everything behaves like a local browser: the CLI, the
   JS/Python/Java clients, and the MCP server all work unchanged.
5. **Tear down** with the recipe's delete command when you're done —
   on most clouds that's what stops the billing.

## Choose a platform

| Platform | Cold start | Best for |
|---|---|---|
| [exe.dev](exe-dev.md) | ~2s VM boot | Everything-cloud: agent + vibium + Chrome on one VM |
| [Fly.io](flyio.md) | seconds (sub-second restart) | Parallel fleets: `fleet.sh up 25`, per-machine DNS, ~$0 stopped |
| [DigitalOcean](digitalocean.md) | ~1 min (post-snapshot) | Simple, predictable droplets |
| [Hetzner](hetzner.md) | ~1 min (post-snapshot) | Budget instances; dedicated bare metal available |
| [GCP](gcp.md) | ~1 min (post-image) | You already live on Google Cloud; gcloud-managed tunnel |
| [AWS](aws.md) | ~1 min (post-AMI) | You already live in AWS — several shapes, see its page |

The cloud-init file works unchanged on most other clouds that accept
user-data (Azure, Vultr, Linode, …), which is why those don't get
pages of their own — they add nothing the six above don't have.
Oracle's always-free ARM tier becomes interesting the day Chrome for
Testing ships linux-arm64.

## What install-chrome.sh does

The shared installer turns a bare Debian/Ubuntu x86-64 box into a
browser server, idempotently:

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

## Reaching a private app

The browser on the box can only test what it can reach — for apps on
localhost or an internal network, see
[testing private sites](../testing-private-sites.md) (short version:
join the box to your tailnet).

## Security model

chromedriver has no authentication, so nothing here exposes it
publicly. Reachability is an SSH tunnel (`ssh -L`) or the platform's
private network (`fly proxy`). Browser automation is remote code
execution — treat the port accordingly.
