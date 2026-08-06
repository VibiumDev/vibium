# Chrome + chromedriver for Fly.io Machines (self-hosted).
#
# Build context is deploy/self-hosted (so install-chrome.sh is visible):
#
#   cd deploy/self-hosted
#   fly launch --no-deploy --name vibium-browsers -c flyio/fly.toml
#   fly deploy -c flyio/fly.toml
#   fly proxy 9515:9515 -a vibium-browsers
#   vibium start http://127.0.0.1:9515
#
# The app has no public ports; fly proxy (WireGuard) is the only way in.

FROM debian:bookworm-slim

COPY install-chrome.sh /tmp/install-chrome.sh
RUN bash /tmp/install-chrome.sh && rm /tmp/install-chrome.sh

USER vibium
EXPOSE 9515

# --allowed-ips="" accepts connections from any peer — safe only because
# fly.toml exposes no public service; reachability is via fly proxy.
CMD ["/usr/local/bin/chromedriver", "--port=9515", "--allowed-ips=", "--allowed-origins=*"]
