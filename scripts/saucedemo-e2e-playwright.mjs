// Prereq: cd scripts && npm install playwright
import { chromium } from "./node_modules/playwright/index.mjs";
import path from "path";
import { fileURLToPath } from "url";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const outPath = path.resolve(__dirname, "..", "saucedemo-playwright-trace.zip");

const browser = await chromium.launch({ headless: false });
const context = await browser.newContext({
    viewport: { width: 1280, height: 720 },
});
const page = await context.newPage();

await context.tracing.start({
    name: "saucedemo-e2e",
    title: "Playwright - SauceDemo E2E Test",
    screenshots: true,
    snapshots: true,
});

// 1. Logging in
await context.tracing.group("Logging in");
await page.goto("https://www.saucedemo.com");
await page.locator("#user-name").fill("standard_user");
await page.locator("#password").fill("secret_sauce");
await page.locator("#login-button").click();
await page.waitForTimeout(500);
await context.tracing.groupEnd();

// 2. Selecting products
await context.tracing.group("Selecting products");
await page.locator("#add-to-cart-sauce-labs-backpack").click();
await page.locator("#add-to-cart-sauce-labs-bike-light").click();
await page.locator("#add-to-cart-sauce-labs-onesie").click();
const badge = await page.locator(".shopping_cart_badge").textContent();
if (badge !== "3") throw new Error(`Expected cart badge "3", got "${badge}"`);
console.log(`Cart badge: ${badge}`);
await context.tracing.groupEnd();

// 3. Reviewing cart
await context.tracing.group("Reviewing cart");
await page.locator(".shopping_cart_link").click();
await page.waitForTimeout(300);
await page.locator("#remove-sauce-labs-bike-light").click();
await context.tracing.groupEnd();

// 4. Checking out
await context.tracing.group("Checking out");
await page.locator("#checkout").click();
await page.locator("#first-name").fill("Test");
await page.locator("#last-name").fill("User");
await page.locator("#postal-code").fill("90210");
await page.locator("#continue").click();
await page.waitForTimeout(300);
await context.tracing.groupEnd();

// 5. Completing order
await context.tracing.group("Completing order");
await page.locator("#finish").click();
await page.waitForTimeout(500);
const confirmation = await page.locator(".complete-header").textContent();
if (!confirmation.includes("Thank you"))
    throw new Error(`Unexpected confirmation: "${confirmation}"`);
console.log(`Confirmation: ${confirmation}`);
await context.tracing.groupEnd();

// 6. Logging out
await context.tracing.group("Logging out");
await page.locator("#react-burger-menu-btn").click();
await page.waitForTimeout(400);
await page.locator("#logout_sidebar_link").click();
await page.waitForTimeout(300);
const loginBtn = await page.locator("#login-button").textContent();
console.log(`Back on login page: ${loginBtn}`);
await context.tracing.groupEnd();

// Stop tracing & save
await context.tracing.stop({ path: outPath });
console.log(`Trace saved → ${outPath}`);

await browser.close();
