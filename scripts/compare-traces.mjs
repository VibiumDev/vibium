import { readFileSync } from "fs";
import { resolve, dirname, extname } from "path";
import { fileURLToPath } from "url";
import { execSync } from "child_process";
import { mkdtempSync, rmSync, readdirSync, statSync } from "fs";
import { tmpdir } from "os";
import { join } from "path";

const __dirname = dirname(fileURLToPath(import.meta.url));
const vibiumZip = resolve(__dirname, "..", "saucedemo-vibium-record.zip");
const playwrightZip = resolve(__dirname, "..", "saucedemo-playwright-trace.zip");

// --- Helpers ---

function unzip(zipPath, destDir) {
    execSync(`unzip -o -q "${zipPath}" -d "${destDir}"`);
}

function listFiles(dir, prefix = "") {
    const entries = [];
    for (const name of readdirSync(dir).sort()) {
        const full = join(dir, name);
        const rel = prefix ? `${prefix}/${name}` : name;
        if (statSync(full).isDirectory()) {
            entries.push(...listFiles(full, rel));
        } else {
            entries.push(rel);
        }
    }
    return entries;
}

function parseNDJSON(filePath) {
    try {
        const content = readFileSync(filePath, "utf-8").trim();
        if (!content) return [];
        return content.split("\n").map((line) => JSON.parse(line));
    } catch {
        return null;
    }
}

function groupBy(arr, fn) {
    const map = {};
    for (const item of arr) {
        const key = fn(item);
        (map[key] ??= []).push(item);
    }
    return map;
}

function deepKeys(obj, prefix = "") {
    if (obj === null || typeof obj !== "object") return [];
    const keys = [];
    for (const [k, v] of Object.entries(obj)) {
        const path = prefix ? `${prefix}.${k}` : k;
        keys.push(path);
        if (typeof v === "object" && v !== null && !Array.isArray(v)) {
            keys.push(...deepKeys(v, path));
        }
    }
    return keys;
}

// --- Main ---

const vibiumDir = mkdtempSync(join(tmpdir(), "vibium-trace-"));
const playwrightDir = mkdtempSync(join(tmpdir(), "playwright-trace-"));

try {
    unzip(vibiumZip, vibiumDir);
    unzip(playwrightZip, playwrightDir);

    const vibiumFiles = listFiles(vibiumDir);
    const playwrightFiles = listFiles(playwrightDir);

    // --- 1. File structure comparison ---
    console.log("=".repeat(80));
    console.log("1. FILE STRUCTURE COMPARISON");
    console.log("=".repeat(80));

    const vSet = new Set(vibiumFiles);
    const pSet = new Set(playwrightFiles);

    // Categorize files
    const vResources = vibiumFiles.filter((f) => f.startsWith("resources/"));
    const pResources = playwrightFiles.filter((f) => f.startsWith("resources/"));
    const vOther = vibiumFiles.filter((f) => !f.startsWith("resources/"));
    const pOther = playwrightFiles.filter((f) => !f.startsWith("resources/"));

    console.log(`\nVibium: ${vibiumFiles.length} files total (${vResources.length} resources, ${vOther.length} other)`);
    console.log(`Playwright: ${playwrightFiles.length} files total (${pResources.length} resources, ${pOther.length} other)`);

    console.log("\nVibium non-resource files:");
    for (const f of vOther) console.log(`  ${f}`);

    console.log("\nPlaywright non-resource files:");
    for (const f of pOther) console.log(`  ${f}`);

    // --- 2. Resource naming comparison ---
    console.log("\n" + "=".repeat(80));
    console.log("2. RESOURCE NAMING");
    console.log("=".repeat(80));

    const vExts = {};
    for (const f of vResources) {
        const ext = extname(f) || "(none)";
        vExts[ext] = (vExts[ext] || 0) + 1;
    }
    const pExts = {};
    for (const f of pResources) {
        const ext = extname(f) || "(none)";
        pExts[ext] = (pExts[ext] || 0) + 1;
    }

    console.log("\nVibium resource extensions:", JSON.stringify(vExts));
    console.log("Playwright resource extensions:", JSON.stringify(pExts));

    console.log("\nVibium resource samples (first 5):");
    for (const f of vResources.slice(0, 5)) console.log(`  ${f}`);

    console.log("\nPlaywright resource samples (first 5):");
    for (const f of pResources.slice(0, 5)) console.log(`  ${f}`);

    // --- 3. Trace file comparison ---
    console.log("\n" + "=".repeat(80));
    console.log("3. TRACE FILE (0-trace.trace) COMPARISON");
    console.log("=".repeat(80));

    // Find trace files (Vibium uses "0-trace.trace", Playwright uses "trace.trace")
    const vTraceFile = vibiumFiles.find((f) => f.endsWith(".trace") && !f.endsWith(".network") && !f.endsWith(".stacks"));
    const pTraceFile = playwrightFiles.find((f) => f.endsWith(".trace") && !f.endsWith(".network") && !f.endsWith(".stacks"));

    // Also check for alternative trace file patterns
    const vTraceFiles = vibiumFiles.filter((f) => f.includes("trace"));
    const pTraceFiles = playwrightFiles.filter((f) => f.includes("trace"));
    console.log("\nVibium trace-related files:", vTraceFiles);
    console.log("Playwright trace-related files:", pTraceFiles);

    const vTrace = vTraceFile ? parseNDJSON(join(vibiumDir, vTraceFile)) : null;
    const pTrace = pTraceFile ? parseNDJSON(join(playwrightDir, pTraceFile)) : null;

    if (vTrace && pTrace) {
        // Event types
        const vTypes = groupBy(vTrace, (e) => e.type);
        const pTypes = groupBy(pTrace, (e) => e.type);

        console.log("\nEvent types:");
        const allTypes = [...new Set([...Object.keys(vTypes), ...Object.keys(pTypes)])].sort();
        console.log(`${"Type".padEnd(30)} ${"Vibium".padEnd(10)} Playwright`);
        console.log("-".repeat(55));
        for (const t of allTypes) {
            const vc = vTypes[t]?.length ?? 0;
            const pc = pTypes[t]?.length ?? 0;
            const marker = vc === 0 ? " [PW only]" : pc === 0 ? " [V only]" : "";
            console.log(`${t.padEnd(30)} ${String(vc).padEnd(10)} ${pc}${marker}`);
        }

        // Compare context-options
        const vCtxOpts = vTrace.find((e) => e.type === "context-options");
        const pCtxOpts = pTrace.find((e) => e.type === "context-options");

        if (vCtxOpts && pCtxOpts) {
            console.log("\n--- context-options comparison ---");
            const vKeys = new Set(deepKeys(vCtxOpts));
            const pKeys = new Set(deepKeys(pCtxOpts));

            const onlyV = [...vKeys].filter((k) => !pKeys.has(k)).sort();
            const onlyP = [...pKeys].filter((k) => !vKeys.has(k)).sort();

            if (onlyV.length) console.log("Keys only in Vibium:", onlyV);
            if (onlyP.length) console.log("Keys only in Playwright:", onlyP);

            // Compare common top-level keys
            const commonTop = [...new Set([...Object.keys(vCtxOpts), ...Object.keys(pCtxOpts)])].sort();
            for (const k of commonTop) {
                const vv = JSON.stringify(vCtxOpts[k]);
                const pv = JSON.stringify(pCtxOpts[k]);
                if (vv !== pv) {
                    console.log(`  ${k}:`);
                    console.log(`    Vibium:     ${vv}`);
                    console.log(`    Playwright: ${pv}`);
                }
            }
        }

        // Compare before/after structure on action events
        const vActions = vTrace.filter((e) => e.type === "before" || e.type === "after");
        const pActions = pTrace.filter((e) => e.type === "before" || e.type === "after");

        if (vActions.length > 0 || pActions.length > 0) {
            console.log("\n--- before/after action events ---");
            console.log(`Vibium: ${vActions.length} events, Playwright: ${pActions.length} events`);

            // Compare first before event structure
            const vBefore = vTrace.find((e) => e.type === "before");
            const pBefore = pTrace.find((e) => e.type === "before");

            if (vBefore && pBefore) {
                const vbKeys = new Set(deepKeys(vBefore));
                const pbKeys = new Set(deepKeys(pBefore));
                const onlyV = [...vbKeys].filter((k) => !pbKeys.has(k)).sort();
                const onlyP = [...pbKeys].filter((k) => !vbKeys.has(k)).sort();
                if (onlyV.length) console.log("'before' keys only in Vibium:", onlyV);
                if (onlyP.length) console.log("'before' keys only in Playwright:", onlyP);

                console.log("\nSample Vibium 'before' event:");
                console.log(JSON.stringify(vBefore, null, 2));
                console.log("\nSample Playwright 'before' event:");
                console.log(JSON.stringify(pBefore, null, 2));
            }

            // Compare action method names
            const vMethods = [...new Set(vActions.filter((e) => e.apiName).map((e) => e.apiName))].sort();
            const pMethods = [...new Set(pActions.filter((e) => e.apiName).map((e) => e.apiName))].sort();

            if (vMethods.length || pMethods.length) {
                console.log("\nVibium apiNames:", vMethods);
                console.log("Playwright apiNames:", pMethods);
            }

            // Compare class values
            const vClasses = [...new Set(vActions.filter((e) => e.class).map((e) => e.class))].sort();
            const pClasses = [...new Set(pActions.filter((e) => e.class).map((e) => e.class))].sort();

            if (vClasses.length || pClasses.length) {
                console.log("\nVibium classes:", vClasses);
                console.log("Playwright classes:", pClasses);
            }
        }

        // Compare screencast frame events
        const vScreens = vTrace.filter((e) => e.type === "screencast-frame");
        const pScreens = pTrace.filter((e) => e.type === "screencast-frame");
        if (vScreens.length > 0 || pScreens.length > 0) {
            console.log("\n--- screencast-frame events ---");
            console.log(`Vibium: ${vScreens.length}, Playwright: ${pScreens.length}`);

            if (vScreens[0] && pScreens[0]) {
                const vsk = Object.keys(vScreens[0]).sort();
                const psk = Object.keys(pScreens[0]).sort();
                console.log("Vibium keys:", vsk);
                console.log("Playwright keys:", psk);

                // Check if sha1 references have extensions
                const vSha = vScreens[0].sha1;
                const pSha = pScreens[0].sha1;
                console.log(`Vibium sha1 sample: ${vSha}`);
                console.log(`Playwright sha1 sample: ${pSha}`);
            }
        }

        // Compare resource-snapshot events
        const vSnaps = vTrace.filter((e) => e.type === "resource-snapshot");
        const pSnaps = pTrace.filter((e) => e.type === "resource-snapshot");
        if (vSnaps.length > 0 || pSnaps.length > 0) {
            console.log("\n--- resource-snapshot events ---");
            console.log(`Vibium: ${vSnaps.length}, Playwright: ${pSnaps.length}`);
        }
    } else {
        if (!vTrace) console.log("Could not parse Vibium trace file");
        if (!pTrace) console.log("Could not parse Playwright trace file");
    }

    // --- 4. Network file comparison ---
    console.log("\n" + "=".repeat(80));
    console.log("4. NETWORK FILE COMPARISON");
    console.log("=".repeat(80));

    const vNetFile = vibiumFiles.find((f) => f.endsWith(".network"));
    const pNetFile = playwrightFiles.find((f) => f.endsWith(".network"));

    console.log(`Vibium network file: ${vNetFile || "(not found)"}`);
    console.log(`Playwright network file: ${pNetFile || "(not found)"}`);

    const vNet = vNetFile ? parseNDJSON(join(vibiumDir, vNetFile)) : null;
    const pNet = pNetFile ? parseNDJSON(join(playwrightDir, pNetFile)) : null;

    if (vNet && pNet) {
        console.log(`\nVibium: ${vNet.length} entries, Playwright: ${pNet.length} entries`);

        // Compare first entry structure
        if (vNet[0] && pNet[0]) {
            const vnKeys = new Set(deepKeys(vNet[0]));
            const pnKeys = new Set(deepKeys(pNet[0]));
            const onlyV = [...vnKeys].filter((k) => !pnKeys.has(k)).sort();
            const onlyP = [...pnKeys].filter((k) => !vnKeys.has(k)).sort();

            if (onlyV.length) console.log("Network entry keys only in Vibium:", onlyV);
            if (onlyP.length) console.log("Network entry keys only in Playwright:", onlyP);

            // Compare event types if present
            const vNetTypes = [...new Set(vNet.map((e) => e.type).filter(Boolean))].sort();
            const pNetTypes = [...new Set(pNet.map((e) => e.type).filter(Boolean))].sort();
            if (vNetTypes.length || pNetTypes.length) {
                console.log("\nVibium network event types:", vNetTypes);
                console.log("Playwright network event types:", pNetTypes);
            }
        }
    } else {
        if (!vNet) console.log("Could not parse Vibium network file");
        if (!pNet) console.log("Could not parse Playwright network file");
    }

    // --- 5. Stacks file comparison ---
    console.log("\n" + "=".repeat(80));
    console.log("5. STACKS FILE COMPARISON");
    console.log("=".repeat(80));

    const vStackFile = vibiumFiles.find((f) => f.endsWith(".stacks"));
    const pStackFile = playwrightFiles.find((f) => f.endsWith(".stacks"));

    console.log(`Vibium stacks file: ${vStackFile || "(not found)"}`);
    console.log(`Playwright stacks file: ${pStackFile || "(not found)"}`);

    // --- 6. Summary ---
    console.log("\n" + "=".repeat(80));
    console.log("6. SUMMARY");
    console.log("=".repeat(80));

    // Resource extension check
    const vHasExts = vResources.some((f) => extname(f) !== "");
    const pHasExts = pResources.some((f) => extname(f) !== "");
    console.log(`\nResource file extensions: Vibium=${vHasExts ? "YES" : "NO"}, Playwright=${pHasExts ? "YES" : "NO"}`);

    // Print all unique extensions in Playwright resources for detail
    if (pHasExts) {
        console.log("Playwright resource extensions breakdown:", pExts);
    }
    if (vHasExts) {
        console.log("Vibium resource extensions breakdown:", vExts);
    }

} finally {
    rmSync(vibiumDir, { recursive: true, force: true });
    rmSync(playwrightDir, { recursive: true, force: true });
}
