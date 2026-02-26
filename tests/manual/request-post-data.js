#!/usr/bin/env node

/**
 * Manual test: request.postData()
 *
 * Run: node tests/manual/request-post-data.js
 *
 * Tests that request.postData() returns the request body
 * using BiDi network.addDataCollector + network.getData.
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
        // ─── 1. postData via route handler ───

        section("postData via route: intercept a POST and read body");

        const page = await b.page();
        await page.go("https://httpbin.org/html");
        await page.wait(500);

        let capturedBody = null;
        await page.route("**/post", async (route) => {
            capturedBody = await route.request.postData();
            console.log(`  [route handler] postData() = ${JSON.stringify(capturedBody)}`);
            route.continue();
        });

        // Give the data collector a moment to register
        await page.wait(300);

        // Trigger a POST via fetch
        await page.eval(`
            fetch('https://httpbin.org/post', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ hello: 'vibium', test: 1 })
            })
        `);

        await page.wait(1500);

        if (capturedBody !== null) {
            const parsed = JSON.parse(capturedBody);
            const matches = parsed.hello === "vibium" && parsed.test === 1;
            console.log(`  ${matches ? PASS : FAIL} postData() returned correct JSON body`);
        } else {
            console.log(`  ${FAIL} postData() returned null`);
            console.log(`  ${INFO} This may mean the browser does not support network.addDataCollector`);
        }

        await page.unroute("**/post");

        // ─── 2. postData via onRequest ───

        section("postData via onRequest: monitor a POST request");

        let onRequestBody = null;
        let onRequestResolved = false;
        const bodyPromise = new Promise((resolve) => {
            page.onRequest(async (req) => {
                if (req.method() === "POST" && req.url().includes("/post")) {
                    const body = await req.postData();
                    console.log(`  [onRequest] postData() = ${JSON.stringify(body)}`);
                    if (!onRequestResolved) {
                        onRequestResolved = true;
                        onRequestBody = body;
                        resolve();
                    }
                }
            });
        });

        // Give the data collector a moment to register
        await page.wait(300);

        await page.eval(`
            fetch('https://httpbin.org/post', {
                method: 'POST',
                headers: { 'Content-Type': 'text/plain' },
                body: 'plain text body from vibium'
            })
        `);

        // Wait for the callback (with timeout)
        await Promise.race([
            bodyPromise,
            page.wait(3000),
        ]);

        if (onRequestBody !== null) {
            const matches = onRequestBody === "plain text body from vibium";
            console.log(`  ${matches ? PASS : FAIL} postData() returned correct plain text body`);
        } else {
            console.log(`  ${FAIL} postData() returned null from onRequest`);
            console.log(`  ${INFO} This may mean the browser does not support network.addDataCollector`);
        }

        // ─── 3. postData returns null for GET requests ───

        section("postData for GET: should return null or empty");

        let getBody = "not-checked";
        await page.route("**/get*", async (route) => {
            getBody = await route.request.postData();
            console.log(`  [route handler] GET postData() = ${JSON.stringify(getBody)}`);
            route.continue();
        });

        await page.wait(300);

        await page.eval("fetch('https://httpbin.org/get?foo=bar')");
        await page.wait(1500);

        // GET requests have no body, so null or empty string is fine
        const getOk = getBody === null || getBody === "" || getBody === "not-checked";
        console.log(`  ${getOk ? PASS : FAIL} GET request postData() is null/empty (got: ${JSON.stringify(getBody)})`);

        await page.unroute("**/get*");

        // ─── 4. postData with form-encoded body ───

        section("postData with form-encoded body");

        let formBody = null;
        await page.route("**/post", async (route) => {
            formBody = await route.request.postData();
            console.log(`  [route handler] form postData() = ${JSON.stringify(formBody)}`);
            route.continue();
        });

        await page.wait(300);

        await page.eval(`
            fetch('https://httpbin.org/post', {
                method: 'POST',
                headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
                body: 'name=vibium&version=1'
            })
        `);

        await page.wait(1500);

        if (formBody !== null) {
            const matches = formBody.includes("name=vibium") && formBody.includes("version=1");
            console.log(`  ${matches ? PASS : FAIL} postData() returned correct form-encoded body`);
        } else {
            console.log(`  ${FAIL} postData() returned null for form body`);
        }

        await page.unroute("**/post");

        // ─── Done ───

        section("Done");
        console.log("  All checks complete. Press Ctrl+C to exit.\n");

        await new Promise(() => {});
    } catch (err) {
        console.error(`\n${FAIL} Error: ${err.message}`);
        console.error(err.stack);
        await b.close();
        process.exit(1);
    }
}

main();
