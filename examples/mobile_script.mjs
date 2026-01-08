// Example of using the new mobile client in Vibium
import { mobile } from "../clients/javascript/dist/index.mjs"; // In a real project: import { mobile } from "vibium";

async function main() {
    console.log("Connecting to Appium...");

    // 1. Launch/Connect to session
    const device = await mobile.connect({
        url: "http://localhost:4723",
        capabilities: {
            platformName: "iOS",
            "appium:platformVersion": "26.1",
            "appium:automationName": "XCUITest",
            'appium:app': '/Users/stuart.minchington@saucelabs.com/Developer/Features.zip',
            'appium:deviceName': 'iPhone 17 Pro'
        }
    });

    console.log("Connected!");

    try {
        // 2. Interact
        // Note: You need a running Appium server and a simulator/device for this to work.

        // Example: Print page source
        console.log("Getting source...");
        const source = await device.source();
        console.log("Source length:", source.length);

        // Tap Geolocation from the main screen
        await device.tap("Apple Pay");

        // Optional: Handle permission flow if it appears
        try {
            console.log("Tapping Buy with Apple Pay button...");
            // Notice: The label has a non-breaking space (U+00A0) between Apple and Pay
            const applePaySelector = "Buy with Apple\u00A0Pay";
            await device.waitFor(applePaySelector, 10000);
            await device.tap(applePaySelector);

            console.log("Waiting for Passcode prompt...");
            // Try robust selector first, fallback to coordinates
            try {
                const passcodeSelector = "**/XCUIElementTypeButton[`label CONTAINS 'Passcode'`]";
                await device.waitFor(passcodeSelector, 5000, "-ios class chain");
                await device.tap(passcodeSelector, "-ios class chain");
            } catch (err) {
                console.log("Could not find Passcode button via selector (likely secure window).");
                console.log("Attempting blind tap at coordinates (200, 800)...");
                await device.tapCoordinates(200, 800);
            }

            console.log("Waiting for purchase to complete...");
            await new Promise(r => setTimeout(r, 3000));

            console.log("Job done!");
        } catch (e) {
            console.log("Job failed:", e.message);
            console.log("Dumping source to error_source.xml...");
            const source = await device.source();
            const fs = await import("fs");
            fs.writeFileSync("error_source.xml", source);
        }

        console.log("Done!");

    } catch (err) {
        console.error("Error:", err);
    } finally {
        // 3. Cleanup
        console.log("Quitting...");
        await device.quit();
    }
}

main();
