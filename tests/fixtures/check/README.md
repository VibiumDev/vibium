# Check archive fixture

`playwright-checkout.zip` is a genuine Playwright **1.58.2**, trace-format **8**
archive. It contains a local routed checkout page, a click that changes its
heading to “Order confirmed,” DOM snapshots (including subtree references),
screenshots, console evidence, and network metadata. It has no Vibium extensions.

`generate-playwright.cjs` documents how it was created. Run it with Playwright
1.58.2 installed in a separate fixture-generation directory and optionally set
`CHROME_EXECUTABLE` to a local Chrome executable. `PLAYWRIGHT_PACKAGE` can point
to that directory's `node_modules/playwright`. Playwright is used only to create
this test artifact; Vibium's implementation uses its existing Go/BiDi runtime.

Newer trace versions must be added deliberately with compatibility tests. The
reader rejects unsupported versions instead of guessing at their structure.
