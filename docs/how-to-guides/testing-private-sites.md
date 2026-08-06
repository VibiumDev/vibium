# Testing private sites with remote browsers

A remote browser can only test what it can reach. Your app on
`localhost:3000`, a LAN address, or an internal VPN doesn't exist from
a browser running in a cloud — so every remote-browser setup needs an
answer to: *how does the browser see the app?*

The right answer depends on which kind of remote browser you have,
because the capabilities differ:

| Your browser runs on | You control the machine? | Best answer |
|---|---|---|
| A cloud computer you rented | Yes — root | Join it to your VPN (Tailscale) |
| A managed cloud browser | No | The vendor's tunnel product |
| Either, occasionally | — | SSH reverse tunnel / public staging |

## Cloud computers: join the box to your tailnet

If you run the browser on a machine you control
([cloud computers](cloud-computers/)), the clean solution is a VPN
between the box and wherever the app lives. With Tailscale:

```bash
# on the browser box (root, one time)
curl -fsSL https://tailscale.com/install.sh | sh
tailscale up
```

The box is now on your tailnet, and the browser on it can reach the
app by MagicDNS name — no per-test tunnels, no exposing anything
publicly:

```bash
vibium start http://127.0.0.1:9515          # protocol, as usual
vibium go http://dev-laptop.tailnet-name.ts.net:3000
```

Two connections, two jobs: the SSH tunnel (or `fly proxy`) carries the
**protocol** from vibium to chromedriver, and the tailnet carries the
**page traffic** from the browser to your app. They're independent —
though once the box is on the tailnet you can also run the SSH tunnel
over it (`ssh -L 9515:127.0.0.1:9515 root@browser-box.tailnet-name.ts.net`)
and drop the box's public SSH exposure entirely.

Plain WireGuard works identically if you'd rather not use Tailscale;
you're just doing the key exchange and routing yourself.

## Managed cloud browsers: the vendor's tunnel

You can't install anything on a managed vendor's browser host, so the
grids ship page-traffic tunnels that solve reachability from their
side: BrowserStack Local, Sauce Connect, the LambdaTest tunnel,
TestingBot Tunnel. Each is a small binary you run next to the app; the
vendor's browser routes matching traffic back through it. They change
nothing about how vibium connects — protocol and page traffic stay
separate — and each grid's [guide](cloud-browsers/) covers its tunnel
flag once the guides are verified.

For BiDi-native vendors without a tunnel product, the fallback is
making the app reachable: a deploy preview, a staging URL behind
auth, or Tailscale Funnel (which publishes a tailnet service to the
internet — mind that "publish" means public).

## Occasional use: SSH reverse tunnel

For a one-off against a box you control, plain SSH does it without
any VPN: `ssh -R 3000:localhost:3000 user@browser-box` makes *your*
laptop's app appear at `localhost:3000` *on the box*, and the browser
there can `vibium go http://localhost:3000`. Symmetric with the `-L`
tunnel the recipes already use for the protocol.

## Benchmarking note

The [cloud bench](../../scripts/cloud-bench/) defaults to a public
target because it compares vendors, and a target must be reachable
from every browser's location. Benching a private app only works for
browsers that can reach it — tailnet-joined cloud computers, not
managed vendors — so cross-vendor numbers against a private target
aren't comparable. Set `BENCH_URL` for a house default target.
