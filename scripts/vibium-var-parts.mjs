import { browser } from "../clients/javascript/dist/index.mjs";

const bro = await browser.start();
const page = await bro.page();

await page.setViewport({ width: 1280, height: 720 });

// 1. Navigate to VAR Parts
console.log("Navigating to var.parts...");
await page.go("https://var.parts");

// 2. Add first 3 products to cart (Battery Pack, Gripper, Sticker Pack)
const buttons = await page.findAll({ role: "button", text: "Add to Cart" });
const productNames = ["Vibium Battery Pack", "Gripper End-Effector", "Vibium Sticker Pack"];

for (let i = 0; i < 3; i++) {
    console.log(`Adding ${productNames[i]}...`);
    await buttons[i].click();
    await page.wait(600);
}

// 3. Go to cart via the cart icon link
console.log("Opening cart...");
await page.find("a[href='/cart']").click();
await page.wait(1000);

const url = await page.url();
console.log(`URL: ${url}`);

// 4. Proceed to checkout
console.log("Proceeding to checkout...");
await page.find({ text: "Proceed to Checkout" }).click();
await page.wait(1000);

// 5. Fill delivery address (using form field IDs)
console.log("Filling delivery address...");
await page.find("#name").fill("VAR-742");
await page.find("#bay").fill("Bay 7-A");
await page.find("#sector").fill("East Quadrant");
await page.find("#station").fill("Armstrong Base");

// 6. Proceed to payment
console.log("Proceeding to payment...");
await page.find({ text: "Proceed to Payment" }).click();
await page.wait(1000);

// 7. Complete payment
console.log("Completing payment...");
await page.find({ text: "Pay Now" }).click();
await page.wait(1500);

// 8. Verify confirmation
await page.wait(2000);
const confirmation = await page.find({ text: "Payment Received!" }).text();
console.log(`Result: ${confirmation}`);

console.log("Checkout complete!");
await bro.stop();
