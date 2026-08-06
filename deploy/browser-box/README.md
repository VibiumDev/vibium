# Browser boxes: DIY remote browsers for vibium

Kits that stand up a Chrome + chromedriver box on your own compute and
connect vibium to it. One shared installer (`setup-chrome.sh`), one thin
recipe per platform:

| Kit | Cold start | Best for |
|---|---|---|
| [exe.dev](exe-dev/) | ~2s VM boot | Everything-cloud: agent + vibium + Chrome on one VM |
| [Fly.io](flyio/) | seconds (sub-second restart) | Parallel fleets: `fleet.sh up 25`, per-machine DNS, ~$0 stopped |
| [DigitalOcean](digitalocean/) | ~1 min (post-snapshot) | Simple, predictable droplets |
| [AWS](aws/) | ~1 min (post-AMI) | You already live in AWS |

Every kit converges on the same connect step, because an http(s) URL to
chromedriver is a classic WebDriver endpoint vibium speaks natively:

```bash
vibium start http://127.0.0.1:9515   # through the kit's tunnel
```

Security model: chromedriver has no authentication, so no kit exposes it
publicly. Reachability is an SSH tunnel (`ssh -L`) or the platform's
private network (`fly proxy`). Browser automation is remote code
execution — treat the port accordingly.
