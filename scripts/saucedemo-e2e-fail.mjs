//import { browser } from 'vibium-client'
import { browser } from "../clients/javascript/dist/index.mjs";
import path from "path";
import { fileURLToPath } from "url";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const outPath = path.resolve(__dirname, "..", "saucedemo-vibium-record-fail.zip");

const bro = await browser.start();
const vibe = await bro.page();

await vibe.setViewport({ width: 1280, height: 720 });

await vibe.context.recording.start({
    name: "saucedemo-e2e",
    title: "Vibium - SauceDemo E2E Test",
    screenshots: true,
    snapshots: true,
    format: "jpeg",
    quality: 0.1,
});

// 1. Logging in
await vibe.context.recording.startGroup("Logging in");
await vibe.go("https://www.saucedemo.com");
await vibe.find("#user-name").fill("standard_user");
await vibe.find("#password").fill("secret_sauce");
await vibe.find("#login-button").click();
await vibe.wait(500);
await vibe.context.recording.stopGroup();

// 2. Selecting products
await vibe.context.recording.startGroup("Selecting products");
await vibe.find("#add-to-cart-sauce-labs-backpack").click();
await vibe.find("#add-to-cart-sauce-labs-bike-light").click();
//await vibe.find("#add-to-cart-sauce-labs-onesie").click();
const badge = await vibe.find(".shopping_cart_badge").text();
if (badge !== "3") throw new Error(`Expected cart badge "3", got "${badge}"`);
console.log(`Cart badge: ${badge}`);
await vibe.context.recording.stopGroup();

// 3. Reviewing cart
await vibe.context.recording.startGroup("Reviewing cart");
await vibe.find(".shopping_cart_link").click();
await vibe.wait(300);
await vibe.find("#remove-sauce-labs-bike-light").click();
await vibe.context.recording.stopGroup();

// 4. Checking out
await vibe.context.recording.startGroup("Checking out");
await vibe.find("#checkout").click();
await vibe.find("#first-name").fill("Test");
await vibe.find("#last-name").fill("User");
await vibe.find("#postal-code").fill("90210");
await vibe.find("#continue").click();
await vibe.wait(300);
await vibe.context.recording.stopGroup();

// 5. Completing order
await vibe.context.recording.startGroup("Completing order");
await vibe.find("#finish").click();
await vibe.wait(500);
const confirmation = await vibe.find(".complete-header").text();
if (!confirmation.includes("Thank you"))
    throw new Error(`Unexpected confirmation: "${confirmation}"`);
console.log(`Confirmation: ${confirmation}`);
await vibe.context.recording.stopGroup();

// 6. Logging out
await vibe.context.recording.startGroup("Logging out");
await vibe.find("#react-burger-menu-btn").click();
await vibe.wait(400);
await vibe.find("#logout_sidebar_link").click();
await vibe.wait(300);
const loginBtn = await vibe.find("#login-button").text();
console.log(`Back on login page: ${loginBtn}`);
await vibe.context.recording.stopGroup();

// Stop recording & save
await vibe.context.recording.stop({ path: outPath });
console.log(`Recording saved → ${outPath}`);

await bro.stop();
