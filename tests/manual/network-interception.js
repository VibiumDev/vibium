#!/usr/bin/env node

/**
 * Manual test: Network Interception & Dialogs
 *
 * Run: node tests/manual/network-interception.js
 *
 * Tests against real websites — requires internet connection.
 * Opens a visible browser so you can see what's happening.
 */

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
    const b = await browser.launch({ headless: false });

    try {
        // ─── 1. route.abort — block images ───

        section("route.abort: Block all images on Wikipedia");

        const page = await b.page();

        let blockedCount = 0;
        await page.route("**/*.{png,jpg,jpeg,gif,svg,webp,ico}", (route) => {
            blockedCount++;
            route.abort();
        });

        await page.go("https://en.wikipedia.org/wiki/Golden_Gate_Bridge");
        await page.wait(2000);

        console.log(`  Blocked ${blockedCount} image requests`);
        console.log(
            `  ${blockedCount > 0 ? PASS : FAIL} Images should be missing from the page`,
        );
        console.log(
            `  ${INFO} Look at the browser — the page should have broken image placeholders`,
        );

        await page.wait(3000);

        // Remove the image-blocking route
        await page.unroute("**/*.{png,jpg,jpeg,gif,svg,webp,ico}");

        // ─── 2. route.continue with header override ───

        section("route.continue: Log and pass through requests");

        const interceptedUrls = [];
        await page.route("**", (route) => {
            interceptedUrls.push(route.request.url());
            route.continue();
        });

        await page.go("https://httpbin.org/html");
        await page.wait(1000);

        const title = await page.title();
        console.log(`  Page title: "${title}"`);
        console.log(`  Intercepted ${interceptedUrls.length} requests:`);
        interceptedUrls.slice(0, 5).forEach((u) => console.log(`    ${u}`));
        if (interceptedUrls.length > 5)
            console.log(`    ... and ${interceptedUrls.length - 5} more`);
        console.log(
            `  ${interceptedUrls.length > 0 ? PASS : FAIL} Requests intercepted and continued`,
        );

        await page.unroute("**");

        // ─── 3. onRequest / onResponse ───

        section("onRequest + onResponse: Monitor network traffic");

        const requests = [];
        const responses = [];

        page.onRequest((req) => {
            requests.push({ url: req.url(), method: req.method() });
        });

        page.onResponse((resp) => {
            responses.push({ url: resp.url(), status: resp.status() });
        });

        await page.go("https://httpbin.org/get");
        await page.wait(1500);

        console.log(
            `  Captured ${requests.length} requests, ${responses.length} responses`,
        );
        requests
            .slice(0, 3)
            .forEach((r) => console.log(`    ${r.method} ${r.url}`));
        responses
            .slice(0, 3)
            .forEach((r) => console.log(`    ${r.status} ${r.url}`));
        console.log(`  ${requests.length > 0 ? PASS : FAIL} onRequest fired`);
        console.log(`  ${responses.length > 0 ? PASS : FAIL} onResponse fired`);

        const hasHeaders =
            requests.length > 0 && typeof requests[0].method === "string";
        console.log(
            `  ${hasHeaders ? PASS : FAIL} request.method() returns string`,
        );

        // ─── 4. waitForResponse ───

        section("waitForResponse: Wait for a specific API response");

        await page.go("https://httpbin.org/html");
        await page.wait(500);

        const responsePromise = page.waitForResponse("**/get?test=1");

        // Trigger the request via fetch
        await page.eval("fetch('https://httpbin.org/get?test=1')");

        const resp = await responsePromise;
        console.log(`  Response URL: ${resp.url()}`);
        console.log(`  Response status: ${resp.status()}`);
        console.log(
            `  Response headers: ${JSON.stringify(Object.keys(resp.headers()).slice(0, 5))}`,
        );
        console.log(
            `  ${resp.status() === 200 ? PASS : FAIL} waitForResponse resolved with status 200`,
        );
        console.log(
            `  ${resp.url().includes("/get?test=1") ? PASS : FAIL} URL matches expected pattern`,
        );

        // ─── 5. waitForRequest ───

        section("waitForRequest: Wait for a specific outgoing request");

        const requestPromise = page.waitForRequest("**/post");

        await page.eval(`
      fetch('https://httpbin.org/post', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ hello: 'vibium' })
      })
    `);

        const req = await requestPromise;
        console.log(`  Request URL: ${req.url()}`);
        console.log(`  Request method: ${req.method()}`);
        console.log(
            `  ${req.method() === "POST" ? PASS : FAIL} waitForRequest captured POST`,
        );
        console.log(
            `  ${req.url().includes("/post") ? PASS : FAIL} URL matches expected pattern`,
        );

        // ─── 6. Dialogs ───
        // Each dialog type uses a fresh page to avoid handler accumulation.

        section("Dialogs: alert, confirm, prompt");

        // Alert
        const alertPage = await b.newPage();
        await alertPage.go("https://httpbin.org/html");
        await alertPage.wait(500);

        let alertMessage = "";
        alertPage.onDialog((dialog) => {
            alertMessage = dialog.message();
            dialog.accept();
        });

        await alertPage.eval('alert("Hello from Vibium!")');
        console.log(`  Alert message: "${alertMessage}"`);
        console.log(
            `  ${alertMessage === "Hello from Vibium!" ? PASS : FAIL} alert captured and accepted`,
        );
        await alertPage.close();

        // Confirm (accept)
        const confirmPage = await b.newPage();
        await confirmPage.go("https://httpbin.org/html");
        await confirmPage.wait(500);

        confirmPage.onDialog((dialog) => {
            dialog.accept();
        });

        const confirmResult = await confirmPage.eval(
            'confirm("Do you like Vibium?")',
        );
        console.log(`  confirm() returned: ${confirmResult}`);
        console.log(
            `  ${confirmResult === true ? PASS : FAIL} confirm accepted returns true`,
        );
        await confirmPage.close();

        // Prompt (accept with text)
        const promptPage = await b.newPage();
        await promptPage.go("https://httpbin.org/html");
        await promptPage.wait(500);

        promptPage.onDialog((dialog) => {
            console.log(
                `  Dialog type: ${dialog.type()}, message: "${dialog.message()}"`,
            );
            dialog.accept("Vibium is great");
        });

        const promptResult = await promptPage.eval(
            'prompt("What do you think?")',
        );
        console.log(`  prompt() returned: "${promptResult}"`);
        console.log(
            `  ${promptResult === "Vibium is great" ? PASS : FAIL} prompt accepted with custom text`,
        );
        await promptPage.close();

        // Dismiss
        const dismissPage = await b.newPage();
        await dismissPage.go("https://httpbin.org/html");
        await dismissPage.wait(500);

        dismissPage.onDialog((dialog) => {
            dialog.dismiss();
        });

        const dismissResult = await dismissPage.eval('confirm("Cancel this?")');
        console.log(`  confirm() after dismiss: ${dismissResult}`);
        console.log(
            `  ${dismissResult === false ? PASS : FAIL} confirm dismissed returns false`,
        );
        await dismissPage.close();

        // ─── 7. setHeaders ───

        section("setHeaders: Add custom headers to all requests");

        const page3 = await b.newPage();

        await page3.setHeaders({ "X-Vibium-Test": "manual-test-123" });
        await page3.go("https://httpbin.org/headers");
        await page3.wait(1000);

        const content = await page3.content();
        const hasCustomHeader =
            content.includes("X-Vibium-Test") &&
            content.includes("manual-test-123");
        console.log(
            `  ${hasCustomHeader ? PASS : FAIL} Custom header X-Vibium-Test visible in httpbin response`,
        );
        if (hasCustomHeader) {
            console.log(
                `  ${INFO} httpbin.org/headers echoes back all request headers`,
            );
        }

        await page3.close();

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
        process.exit(1);
    }
}

main();
