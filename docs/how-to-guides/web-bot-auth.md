# Web Bot Auth (Cloudflare Verified Bots)

Some sites put automated traffic behind a challenge like Cloudflare Turnstile.
Beating that challenge is not a game Vibium plays. The supported path is the
cooperative one: identify the agent cryptographically with
[Web Bot Auth](https://datatracker.ietf.org/doc/html/draft-meunier-web-bot-auth-architecture),
the signing half of Cloudflare's
[Verified Bots](https://developers.cloudflare.com/bots/concepts/bot/verified-bots/)
program, so a site that chooses to allow your bot can recognize it. This does
not hide automation; it declares it.

When enabled, Vibium signs every outgoing request with an Ed25519
[RFC 9421 HTTP Message Signature](https://www.rfc-editor.org/rfc/rfc9421.html)
over the request authority, adding `Signature`, `Signature-Input`, and
(optionally) `Signature-Agent` headers.

## 1. Make a signing key

```sh
vibium webbotauth keygen -o web-bot-auth.pem
# Wrote web-bot-auth.pem (keyid: poqkLGiymh_W0uP6PZFw-dvez3QJT5SolqXBCW38r0U)
```

The keyid is the base64url JWK thumbprint of the public key. Keep the `.pem`
private.

## 2. Publish the public key

Serve the directory at
`https://<your-host>/.well-known/http-message-signatures-directory`:

```sh
vibium webbotauth directory --key web-bot-auth.pem
# {"keys":[{"kty":"OKP","crv":"Ed25519","x":"...","kid":"...","nbf":...}]}
```

Enroll that host with Cloudflare's Verified Bots program so participating sites
can choose to allow it. Signing works without enrollment, but only enrolled,
allowed bots are let through.

Note that the draft expects the directory *response* to be signed with the
same key, which a static file host cannot do. A plain JWKS file may be enough
to get enrolled today, but if enrollment or verification rejects your
directory, that response signature is the first thing to check.

## 3. Run Vibium with signing on

Point `VIBIUM_WEB_BOT_AUTH_KEY` at the key, and set
`VIBIUM_WEB_BOT_AUTH_DIRECTORY` to the https URL hosting the directory so the
signed `Signature-Agent` header tells verifiers where to fetch the key:

```sh
export VIBIUM_WEB_BOT_AUTH_KEY=./web-bot-auth.pem
export VIBIUM_WEB_BOT_AUTH_DIRECTORY=https://your-host.example
```

Then drive the browser through a client library as usual — every request the
browser makes is signed. A bad key path disables signing with a message on
stderr rather than stopping the browser, so check that line if requests are
not being signed.

## Scope and limits

- Signing covers the client-library path (`vibium pipe`, which the JS, Python,
  and Java clients launch). Signing in the daemon-backed CLI, MCP, Run, and
  Check paths is not wired yet; those drive the browser through a separate
  stack.
- A session that signs cannot also use `page.route`: signing owns the request
  intercept, and a client route would race its continue. Use one or the other
  per session. The same applies to calling `network.addIntercept` directly
  over raw BiDi: a second intercept races the signer's continue and is not
  guarded the way `page.route` is.
- Signatures carry a short validity window and a per-request nonce to limit
  replay. The authority signed is the request URL host.
