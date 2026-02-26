/**
 * Manual Clock Test — pinthing.com
 *
 * Run: node tests/manual/clock-pinthing.mjs
 *
 * This opens pinthing.com in a visible browser and manipulates the clock
 * so you can see the effects visually. Each step pauses for you to observe.
 */

import { browser } from "../../clients/javascript/dist/index.mjs";

function sleep(ms) {
    return new Promise((r) => setTimeout(r, ms));
}

async function main() {
    const b = await browser.launch();
    const page = await b.page();

    console.log("1. Navigating to pinthing.com...");
    await page.go("https://www.pinthing.com");
    await page.waitForLoad();
    await sleep(2000);

    console.log("2. Installing fake clock...");
    await page.clock.install();
    console.log(
        "   ✓ Clock installed. Date/setTimeout/setInterval are now fake.",
    );
    await sleep(2000);

    console.log("3. Freezing time with setFixedTime (Jan 1, 2000 noon UTC)...");
    await page.clock.setFixedTime(new Date("2000-01-01T12:00:00Z"));
    const frozenYear = await page.eval("new Date().getFullYear()");
    console.log(`   ✓ Date.now() year = ${frozenYear} (should be 2000)`);
    console.log("   → Look at the page — any live clocks should be frozen.");
    await sleep(3000);

    console.log("4. Changing to setFixedTime (Dec 31, 2030 23:59:50 UTC)...");
    await page.clock.setFixedTime(new Date("2030-12-31T23:59:50Z"));
    const newYear = await page.eval("new Date().toISOString()");
    console.log(`   ✓ Date.now() = ${newYear}`);
    console.log("   → Page should show a time near midnight 2030.");
    await sleep(3000);

    console.log("5. Using setSystemTime to jump to July 4, 2025...");
    await page.clock.setSystemTime(new Date("2025-07-04T15:30:00Z"));
    const sysTime = await page.eval("new Date().toISOString()");
    console.log(`   ✓ Date.now() = ${sysTime}`);
    await sleep(3000);

    console.log("6. Fast-forwarding 60 seconds...");
    await page.clock.fastForward(60_000);
    const afterFF = await page.eval("new Date().toISOString()");
    console.log(`   ✓ After fastForward(60000): ${afterFF}`);
    await sleep(2000);

    console.log("7. Running clock for 5 minutes (300,000 ms) with runFor...");
    await page.clock.runFor(300_000);
    const afterRun = await page.eval("new Date().toISOString()");
    console.log(`   ✓ After runFor(300000): ${afterRun}`);
    await sleep(2000);

    console.log("8. Pausing at a specific time...");
    await page.clock.pauseAt(new Date("2025-12-25T00:00:00Z"));
    const paused = await page.eval("new Date().toISOString()");
    console.log(`   ✓ Paused at: ${paused}`);
    console.log("   → Time is frozen. Nothing should tick.");
    await sleep(3000);

    console.log("9. Resuming real-time flow from paused time...");
    await page.clock.resume();
    console.log(
        "   ✓ Resumed. Clock should now tick forward from Dec 25, 2025.",
    );
    await sleep(3000);

    const resumed = await page.eval("new Date().toISOString()");
    console.log(`   → Current time after ~3s of real-time: ${resumed}`);

    console.log(
        "\n✅ All clock methods exercised. Browser stays open for inspection.",
    );
    console.log("   Press Ctrl+C to exit.\n");

    // Keep process alive so user can inspect
    await new Promise(() => {});
}

main().catch((err) => {
    console.error("Error:", err.message);
    process.exit(1);
});
