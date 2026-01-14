# Using Proxies with Vibium

Vibium supports routing browser traffic through proxy servers. This is useful for:
- Testing geo-restricted content
- Working behind corporate firewalls
- Privacy and anonymity
- Load balancing and rate limiting

## Supported Proxy Types

- **HTTP/HTTPS proxies**: Standard HTTP proxy protocol
- **SOCKS5 proxies**: More versatile, works with any protocol
- **Authenticated proxies**: Both HTTP and SOCKS5 with username/password

## Command Line Usage

### Basic HTTP Proxy

```bash
./clicker/bin/clicker navigate https://example.com --proxy http://proxy.example.com:8080
```

**Sample output:**
```
Navigating to https://example.com...
✓ Loaded: https://example.com (via proxy http://proxy.example.com:8080)
```

### SOCKS5 Proxy

```bash
./clicker/bin/clicker navigate https://example.com --proxy socks5://proxy.example.com:1080
```

### Authenticated Proxy

```bash
./clicker/bin/clicker navigate https://example.com --proxy http://user:pass@proxy.example.com:8080
```

**Note:** Special characters in passwords should be URL-encoded.

### With Other Flags

Proxy works with all other flags:

```bash
# Headless mode with proxy
./clicker/bin/clicker screenshot https://example.com --headless --proxy http://proxy:8080 -o shot.png

# Serve mode with proxy (all browser sessions use the proxy)
./clicker/bin/clicker serve --proxy socks5://proxy:1080
```

## JavaScript/TypeScript Client

### Async API

```javascript
import { browser } from 'vibium';

// HTTP proxy
const vibe = await browser.launch({
  proxy: 'http://proxy.example.com:8080'
});

// SOCKS5 with auth
const vibe = await browser.launch({
  proxy: 'socks5://user:pass@proxy.example.com:1080'
});

await vibe.go('https://example.com');
await vibe.quit();
```

### Sync API

```javascript
const { browserSync } = require('vibium');

const vibe = browserSync.launch({
  proxy: 'http://proxy.example.com:8080'
});

vibe.go('https://example.com');
vibe.quit();
```

## Python Client

### Async API

```python
from vibium import browser

# HTTP proxy
vibe = await browser.launch(proxy='http://proxy.example.com:8080')
await vibe.go('https://example.com')
await vibe.quit()

# SOCKS5 with auth
vibe = await browser.launch(proxy='socks5://user:pass@proxy.example.com:1080')
```

### Sync API

```python
from vibium import browser_sync

vibe = browser_sync.launch(proxy='http://proxy.example.com:8080')
vibe.go('https://example.com')
vibe.quit()
```

## MCP Server

When using the MCP server (e.g., with Claude Code), pass the proxy parameter to `browser_launch`:

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/call",
  "params": {
    "name": "browser_launch",
    "arguments": {
      "proxy": "http://proxy.example.com:8080"
    }
  }
}
```

The proxy setting applies to all subsequent browser operations in that session.

## Proxy URL Format

Proxy URLs follow this format:

```
[protocol://][username:password@]host:port
```

**Examples:**
- `http://proxy.example.com:8080` - Basic HTTP proxy
- `socks5://proxy.example.com:1080` - SOCKS5 proxy
- `http://user:pass@proxy.example.com:8080` - HTTP proxy with authentication
- `socks5://user:pass@proxy.example.com:1080` - SOCKS5 with authentication

## Troubleshooting

### Connection Fails

If the browser fails to connect through the proxy:

1. Verify the proxy is accessible: `curl -x http://proxy:8080 https://example.com`
2. Check authentication credentials
3. Ensure the proxy supports HTTPS CONNECT method (for HTTPS sites)
4. Try with `--verbose` flag to see detailed connection logs

### SOCKS5 Not Working

Chrome/Chromium requires that SOCKS5 proxies are specified as `socks5://` (not `socks://`).

### Special Characters in Password

URL-encode special characters:
- `@` → `%40`
- `:` → `%3A`
- `/` → `%2F`

Example: `http://user:p%40ssword@proxy:8080` for password `p@ssword`

## Environment Variables

Currently, Vibium does not support environment variables like `HTTP_PROXY` or `HTTPS_PROXY`. You must explicitly pass the proxy URL.

## Security Considerations

- Proxy credentials are passed to Chrome via command-line arguments, which may be visible in process listings
- For sensitive operations, consider using SSH tunnels or VPNs instead
- Authenticated proxy credentials are not encrypted in memory
