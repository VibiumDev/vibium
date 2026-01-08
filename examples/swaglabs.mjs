// Simplified Vibium code snippet for a login test
import { browser } from "vibium";
async function testLogin() {
    const vibe = await browser.launch();
    await vibe.go("https://www.saucedemo.com/"); // Navigate
    await vibe.find('input[type="username"]').type("standard_user"); // Find and type
    await vibe.find('input[type="password"]').type("secret_sauce"); // Find and type
    await vibe.find('button[type="submit"]').click(); // Find and click

    // Vibium waits for the next expected state (e.g., dashboard)
    console.log('Login successful');
    await vibe.quit();
}
