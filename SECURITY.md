# Security Policy

## Reporting a vulnerability

Please report security issues privately, not in a public issue or pull request.

Use GitHub's private vulnerability reporting: open the repository's **Security** tab and click **Report a vulnerability**. That opens a private advisory visible only to the maintainers.

Please include what is needed to reproduce it: the vibium version, the OS and browser engine, steps or a short proof of concept, and the impact you see. A maintainer will follow up to confirm the issue and coordinate a fix and disclosure.

## Supported versions

vibium ships from a single release line. Security fixes land on `main` and go out in the next release, so please reproduce against the latest release or `main` before reporting.

## Scope

In scope:

- The `vibium` binary: the CLI, `serve`, `pipe`, the daemon, and the MCP server.
- The official JavaScript, Python, and Java clients.

Out of scope:

- Vulnerabilities in the browsers vibium drives. Report those to Chrome or Firefox directly.
- Vulnerabilities in third-party dependencies. Report those upstream, though a heads-up here is welcome.
- Behavior that requires an already-compromised local account or a daemon the operator has deliberately exposed off localhost.
