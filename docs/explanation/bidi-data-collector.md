# BiDi Data Collection: How Vibium Reads Request Bodies

WebDriver BiDi network events tell you *about* requests — the URL, method, headers, status code — but they don't include the actual body bytes. This is by design: cloning every request and response body would be expensive, and most automation doesn't need it.

When you *do* need body data (reading POST payloads, capturing response content), BiDi provides an opt-in system: **data collectors**.

## The Problem

When a `network.beforeRequestSent` event fires, you get:

```json
{
  "method": "network.beforeRequestSent",
  "params": {
    "request": {
      "request": "req-123",
      "url": "https://api.example.com/users",
      "method": "POST",
      "headers": [{"name": "Content-Type", "value": {"type": "string", "value": "application/json"}}],
      "bodySize": 42
    }
  }
}
```

You can see that 42 bytes were sent, but you can't see *what* those bytes are. `bodySize` tells you a body exists — getting the actual content requires a different mechanism.

## How Data Collectors Work

Data collection is a three-step protocol:

```
  Vibium                          Browser
    │                               │
    │  1. addDataCollector          │
    │  "watch request bodies"       │
    │──────────────────────────────►│
    │                               │
    │           ┌───────────────────┤
    │           │ 2. Clones request │
    │           │    bodies into    │
    │           │    buffer as they │
    │           │    happen         │
    │           └───────────────────┤
    │                               │
    │  3. getData(request: "req-1") │
    │──────────────────────────────►│
    │                               │
    │  body bytes                   │
    │◄──────────────────────────────│
    │                               │
```

### Step 1: Register a collector

Tell the browser what to collect and how much to buffer:

```json
{
  "method": "network.addDataCollector",
  "params": {
    "dataTypes": ["request"],
    "maxEncodedDataSize": 10485760
  }
}
```

- `dataTypes` — what to collect: `"request"`, `"response"`, or both
- `maxEncodedDataSize` — max bytes per individual item (not total). Requests larger than this are silently skipped.

The browser returns a collector ID:

```json
{
  "result": {
    "collector": "collector-abc-123"
  }
}
```

From this point on, the browser clones the body of every matching network request into a buffer. The cloning happens at the network layer, before the page's JavaScript sees the data.

### Step 2: Wait for a request to happen

The collector runs in the background. You don't need to do anything special — just let requests happen naturally (page navigation, fetch calls, form submissions, etc.).

### Step 3: Retrieve the collected data

When you want the body for a specific request, ask for it by request ID:

```json
{
  "method": "network.getData",
  "params": {
    "dataType": "request",
    "request": "req-123"
  }
}
```

The browser returns the raw bytes:

```json
{
  "result": {
    "bytes": {
      "type": "string",
      "value": "{\"name\":\"Alice\",\"email\":\"alice@example.com\"}"
    }
  }
}
```

If the request hasn't completed yet, `network.getData` blocks until the body is available. If the body was too large for the collector's `maxEncodedDataSize`, or no collector was active when the request happened, you get a `no such network data` error.

### Cleanup: Remove the collector

When you're done collecting, tear down the collector to free resources:

```json
{
  "method": "network.removeDataCollector",
  "params": {
    "collector": "collector-abc-123"
  }
}
```

Already-collected data remains available until the browser discards it (subject to a global memory limit).

## Why Opt-In?

Body cloning has real costs:

- **Memory** — The browser must buffer a copy of every matching body. A page that loads 50 images and 20 API calls could consume significant memory.
- **Performance** — Cloning the network stream adds overhead to every request.
- **Privacy** — Request bodies can contain credentials, tokens, PII. Not collecting them by default is safer.

The opt-in model means you only pay these costs when you need them. Most automation (clicking buttons, reading text, taking screenshots) never touches request bodies.

## How Vibium Uses This

### `request.postData()`

When you call `page.route()` or `page.onRequest()`, Vibium automatically sets up a data collector for request bodies behind the scenes. This makes `request.postData()` work:

```javascript
await page.route('**/api/**', async (route) => {
  const body = await route.request.postData();
  console.log('API received:', body);  // '{"name":"Alice"}'
  route.continue();
});
```

Without a data collector, `postData()` would always return `null` — the BiDi event simply doesn't include the body.

The collector is set up lazily (only when you register a route or request listener) and torn down when you call `page.removeAllListeners('request')`. If the browser doesn't support `network.addDataCollector`, `postData()` gracefully returns `null`.

### Scoping

The `addDataCollector` command accepts optional `contexts` and `userContexts` parameters to limit collection to specific pages or browser contexts. Vibium currently registers a session-wide collector (no context filter) so that `postData()` works regardless of which page made the request.

## Common Use Cases

### Verifying API Payloads

The most common case: your test clicks a button, and you want to verify what data was sent to the server.

```javascript
page.onRequest(async (req) => {
  if (req.method() === 'POST' && req.url().includes('/api/checkout')) {
    const body = JSON.parse(await req.postData());
    assert(body.items.length === 3);
    assert(body.total === 59.97);
  }
});

await page.find({ role: 'button', text: 'Place Order' }).click();
```

### Request Mocking with Body Inspection

Intercept requests, inspect the body, and return different mock responses based on what was sent:

```javascript
await page.route('**/api/search', async (route) => {
  const body = JSON.parse(await route.request.postData());

  if (body.query === 'empty') {
    route.fulfill({ status: 200, body: JSON.stringify({ results: [] }) });
  } else {
    route.fulfill({ status: 200, body: JSON.stringify({ results: ['item1'] }) });
  }
});
```

### Form Submission Validation

Verify that form data is properly encoded before it hits the server:

```javascript
const reqPromise = page.waitForRequest('**/submit');

await page.find('#name').fill('Jane Doe');
await page.find('#email').fill('jane@example.com');
await page.find({ role: 'button', text: 'Submit' }).click();

const req = await reqPromise;
const body = await req.postData();
// body === 'name=Jane+Doe&email=jane%40example.com'
```

### Response Body Collection: `response.body()` and `response.json()`

When you call `page.onResponse()` or `page.waitForResponse()`, Vibium sets up a data collector that includes the `"response"` data type. This makes `response.body()` and `response.json()` work:

```javascript
page.onResponse(async (resp) => {
  if (resp.url().includes('/api/users')) {
    const data = await resp.json();
    console.log('Got users:', data);  // [{name: "Alice"}, ...]
  }
});
```

Or with `waitForResponse`:

```javascript
const resp = await page.waitForResponse('**/api/users');
const body = await resp.body();    // raw string
const data = await resp.json();    // parsed JSON
```

Under the hood, `body()` calls `network.getData` with `"dataType": "response"` and the request ID from the `network.responseCompleted` event. Base64-encoded responses (binary content) are automatically decoded. If no data collector was active when the response arrived, `body()` returns `null`.

### Network Tracing

Collecting both request and response bodies gives you a complete record of all network traffic — useful for debugging, generating HAR files, or replay testing:

```json
{
  "method": "network.addDataCollector",
  "params": {
    "dataTypes": ["request", "response"],
    "maxEncodedDataSize": 5242880
  }
}
```

Combined with `network.beforeRequestSent` and `network.responseCompleted` events (which provide URLs, headers, timing, etc.), you have everything needed to reconstruct the full HTTP conversation.

## Limits and Edge Cases

- **`maxEncodedDataSize` is per-item, not total.** Setting it to 10 MB means each individual request body can be up to 10 MB, not 10 MB across all requests.
- **The browser has a global memory cap** (`max total collected size`) for all collected data. When it's exceeded, older data gets evicted. The exact limit is browser-dependent.
- **Collector must be active when the request happens.** You can't retroactively collect data for requests that already completed before the collector was registered.
- **Binary bodies are base64-encoded** in the `bytes` field (with `"type": "base64"`). Text bodies use `"type": "string"`.
- **The `disown` flag** on `network.getData` lets you release a specific collector's claim on the data, freeing memory earlier.
