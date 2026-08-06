# Chrome + chromedriver browser box for Fly.io Machines.
#
# Build context is deploy/browser-box (so setup-chrome.sh is visible):
#
#   cd deploy/browser-box
#   fly launch --no-deploy --name vibium-browsers -c flyio/fly.toml
#   fly deploy -c flyio/fly.toml
#   fly proxy 9515:9515 -a vibium-browsers
#   vibium start http://127.0.0.1:9515
#
# The app has no public ports; fly proxy (WireGuard) is the only way in.

FROM debian:bookworm-slim

COPY setup-chrome.sh /tmp/setup-chrome.sh
RUN bash /tmp/setup-chrome.sh && rm /tmp/setup-chrome.sh

USER vibium
EXPOSE 9515

# --allowed-ips="" accepts connections from any peer — safe only because
# fly.toml exposes no public service; reachability is via fly proxy.
CMD ["/usr/local/bin/chromedriver", "--port=9515", "--allowed-ips=", "--allowed-origins=*"]
