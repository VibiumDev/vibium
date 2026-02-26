#!/usr/bin/env node

/**
 * Manual test: route.fulfill() — Mock API responses
 *
 * Run: node tests/manual/route-fulfill.js
 *
 * Demonstrates intercepting requests and returning mock responses
 * instead of hitting the real server. Uses a local HTTP server.
 * Opens a visible browser so you can see what's happening.
 */

const http = require("http");
const { browser } = require("../../clients/javascript/dist");

const PASS = "\x1b[32mPASS\x1b[0m";
const FAIL = "\x1b[31mFAIL\x1b[0m";
const INFO = "\x1b[36mINFO\x1b[0m";
const BOLD = "\x1b[1m";
const RESET = "\x1b[0m";

function section(title) {
    console.log(`\n${BOLD}━━━ ${title} ━━━${RESET}`);
}

async function main() {
    // Start a local test server
    const server = http.createServer((req, res) => {
        if (req.url === "/api/users") {
            res.writeHead(200, { "Content-Type": "application/json" });
            res.end(JSON.stringify({ users: ["Alice", "Bob"], source: "real-server" }));
        } else if (req.url === "/api/status") {
            res.writeHead(200, { "Content-Type": "application/json" });
            res.end(JSON.stringify({ status: "ok", source: "real-server" }));
        } else {
            res.writeHead(200, { "Content-Type": "text/html" });
            res.end(`<!DOCTYPE html>
<html><head><title>Test App</title></head>
<body>
  <h1>Test App</h1>
  <div id="result"></div>
</body></html>`);
        }
    });

    const baseURL = await new Promise((resolve) => {
        server.listen(0, "127.0.0.1", () => {
            resolve(`http://127.0.0.1:${server.address().port}`);
        });
    });
    console.log(`${INFO} Test server running at ${baseURL}`);

    const b = await browser.launch({ headless: false });

    try {
        const page = await b.page();

        // ─── 1. Mock a JSON API response ───

        section("route.fulfill: Mock a JSON API response");

        await page.go(baseURL);
        await page.wait(500);

        await page.route("**/api/users", (route) => {
            console.log(`  ${INFO} Intercepted: ${route.request.url()}`);
            route.fulfill({
                status: 200,
                contentType: "application/json",
                body: JSON.stringify({
                    users: ["Mocked-Alice", "Mocked-Bob", "Mocked-Charlie"],
                    source: "route.fulfill()",
                }),
            });
        });

        const result = await page.eval(`
            fetch('/api/users').then(r => r.json())
        `);

        console.log(`  Response: ${JSON.stringify(result)}`);
        console.log(
            `  ${result.source === "route.fulfill()" ? PASS : FAIL} Got mocked response (not real server)`,
        );
        console.log(
            `  ${result.users.length === 3 ? PASS : FAIL} Mock has 3 users (real server has 2)`,
        );

        await page.unroute("**/api/users");

        // ─── 2. Mock with custom status and headers ───

        section("route.fulfill: Custom status code and headers");

        await page.route("**/api/status", (route) => {
            console.log(`  ${INFO} Intercepted: ${route.request.url()}`);
            route.fulfill({
                status: 418,
                headers: {
                    "Content-Type": "text/plain",
                    "X-Teapot": "yes",
                    "X-Powered-By": "Vibium",
                },
                body: "I'm a teapot! (mocked by Vibium)",
            });
        });

        const teapot = await page.eval(`
            fetch('/api/status')
                .then(r => r.text().then(body => ({
                    status: r.status,
                    body,
                    teapot: r.headers.get('X-Teapot'),
                    poweredBy: r.headers.get('X-Powered-By'),
                })))
        `);

        console.log(`  Status: ${teapot.status}`);
        console.log(`  Body: "${teapot.body}"`);
        console.log(`  X-Teapot: ${teapot.teapot}`);
        console.log(`  X-Powered-By: ${teapot.poweredBy}`);
        console.log(`  ${teapot.status === 418 ? PASS : FAIL} Status code is 418`);
        console.log(`  ${teapot.teapot === "yes" ? PASS : FAIL} Custom header X-Teapot present`);
        console.log(`  ${teapot.poweredBy === "Vibium" ? PASS : FAIL} Custom header X-Powered-By present`);

        await page.unroute("**/api/status");

        // ─── 3. Serve a full mock HTML page ───

        section("route.fulfill: Serve a complete mock page");

        await page.route("**/mock-page", (route) => {
            console.log(`  ${INFO} Intercepted: ${route.request.url()}`);
            route.fulfill({
                status: 200,
                contentType: "text/html",
                body: `<!DOCTYPE html>
<html>
<head><title>Vibium Mock Page</title></head>
<body style="font-family: system-ui; padding: 2em; background: #1a1a2e; color: #eee;">
  <h1 style="color: #00d4ff;">This page was served by route.fulfill()</h1>
  <p>No server was involved — the response was injected by Vibium.</p>
</body>
</html>`,
            });
        });

        await page.go(`${baseURL}/mock-page`);
        await page.wait(500);

        const title = await page.title();
        const bodyText = await page.eval("document.body.innerText");

        console.log(`  Page title: "${title}"`);
        console.log(`  ${title === "Vibium Mock Page" ? PASS : FAIL} Mock HTML page loaded`);
        console.log(`  ${bodyText.includes("route.fulfill") ? PASS : FAIL} Page body has expected content`);
        console.log(`  ${INFO} Look at the browser — you should see the styled mock page`);

        await page.unroute("**/mock-page");

        // ─── 4. Selective mocking — mock API, let page through ───

        section("route.fulfill: Selective mocking (mock API, real page)");

        await page.route("**/api/users", (route) => {
            console.log(`  ${INFO} Mocking: ${route.request.url()}`);
            route.fulfill({
                status: 200,
                contentType: "application/json",
                body: JSON.stringify({ users: ["Only-Mock"], source: "mock" }),
            });
        });

        // Navigate to the real page (not intercepted by the /api/users route)
        await page.go(baseURL);
        await page.wait(500);

        const realTitle = await page.title();
        console.log(`  Real page title: "${realTitle}"`);
        console.log(`  ${realTitle === "Test App" ? PASS : FAIL} Real page loaded normally`);

        // Fetch the mocked API from within the real page
        const apiResult = await page.eval(`
            fetch('/api/users').then(r => r.json())
        `);

        console.log(`  API response: ${JSON.stringify(apiResult)}`);
        console.log(`  ${apiResult.source === "mock" ? PASS : FAIL} API is mocked while page is real`);

        await page.unroute("**/api/users");

        // ─── Done ───

        section("Done");
        console.log(
            "  All checks complete. Review the browser window, then press Ctrl+C to exit.\n",
        );

        // Keep browser open so user can inspect
        await new Promise(() => {});
    } catch (err) {
        console.error(`\n${FAIL} Error: ${err.message}`);
        console.error(err.stack);
        await b.close();
        server.close();
        process.exit(1);
    }
}

main();
