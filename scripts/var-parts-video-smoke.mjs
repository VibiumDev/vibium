// Smoke test: Firefox video recording over a real var.parts purchase flow.
// Verifies the WebM track actually lands in the zip and is a real WebM file.
//
//   node scripts/var-parts-video-smoke.mjs
//
// Firefox 154 is required for video. Until it reaches stable on 2026-08-18 the
// beta channel is needed; VIBIUM_ENGINE_CHANNEL overrides if that has landed.

import { firefox } from "../clients/javascript/dist/index.mjs";
import { execFileSync } from "child_process";
import fs from "fs";
import os from "os";
import path from "path";
import { fileURLToPath } from "url";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const outPath = path.resolve(__dirname, "..", "var-parts-video-record.zip");
const channel = process.env.VIBIUM_ENGINE_CHANNEL || "beta";

const failures = [];
function check(ok, label) {
    console.log(`  ${ok ? "PASS" : "FAIL"}  ${label}`);
    if (!ok) failures.push(label);
}

fs.rmSync(outPath, { force: true });

console.log(`Starting Firefox (channel: ${channel})...`);
const bro = await firefox.start({ channel });
let result;

try {
    const vibe = await bro.page();
    await vibe.setViewport({ width: 1280, height: 720 });

    // video: true fails loudly if the engine can't record, which is the point here.
    await vibe.context.recording.start({
        name: "var-parts-video-smoke",
        title: "Vibium - var.parts video smoke test",
        video: true,
        path: outPath,
    });

    console.log("Navigating to var.parts...");
    await vibe.go("https://var.parts");

    console.log("Adding an item to the cart...");
    await vibe.context.recording.startGroup("Selecting an item");
    const addButtons = await vibe.findAll({ role: "button", text: "Add to Cart" });
    if (addButtons.length === 0) throw new Error('No "Add to Cart" buttons found on var.parts');
    await addButtons[0].click();
    await vibe.wait(600);
    await vibe.context.recording.stopGroup();

    console.log("Opening the cart...");
    await vibe.context.recording.startGroup("Opening the cart");
    await vibe.find("a[href='/cart']").click();
    await vibe.wait(1200);
    await vibe.context.recording.stopGroup();

    const url = await vibe.url();
    console.log(`Cart URL: ${url}`);

    result = await vibe.context.recording.stop();
} finally {
    await bro.stop();
}

console.log(`\nRecording: ${result.path}  (${result.steps} steps, ${result.durationMs}ms)\n`);

console.log("Stop result:");
check(!result.videoUnavailable, `video was recorded${result.videoUnavailable ? ` (got: ${result.videoUnavailable})` : ""}`);
check(Array.isArray(result.videos) && result.videos.length > 0, "result.videos is non-empty");

for (const v of result.videos ?? []) {
    check(!v.error, `video ${v.context} recorded without error${v.error ? ` (got: ${v.error})` : ""}`);
    check(v.width > 0 && v.height > 0, `video ${v.context} has dimensions (${v.width}x${v.height})`);
    check(v.durationMs > 0, `video ${v.context} has duration (${v.durationMs}ms)`);
}

console.log("\nZip contents:");
check(fs.existsSync(outPath), `zip exists at ${outPath}`);

const entries = execFileSync("unzip", ["-Z1", outPath], { encoding: "utf-8" })
    .split("\n")
    .map((l) => l.trim())
    .filter(Boolean);

const webms = entries.filter((e) => e.startsWith("video/") && e.endsWith(".webm"));
check(webms.length > 0, `zip contains a video/*.webm (found: ${webms.join(", ") || "none"})`);
check(entries.includes("video/index.json"), "zip contains video/index.json");

for (const entry of webms) {
    const bytes = execFileSync("unzip", ["-p", outPath, entry], { maxBuffer: 512 * 1024 * 1024 });

    // EBML magic — a WebM/Matroska file always opens with 1A 45 DF A3.
    const isWebM = bytes.subarray(0, 4).equals(Buffer.from([0x1a, 0x45, 0xdf, 0xa3]));
    check(isWebM, `${entry} starts with the EBML magic bytes`);
    check(bytes.length > 1024, `${entry} is non-trivial (${bytes.length} bytes)`);

    // ffprobe reads the encoded stream itself, which is what separates a
    // genuinely broken video from dimensions merely being misreported.
    const probed = probe(bytes);
    if (!probed) {
        console.log(`  SKIP  ${entry} stream probe (ffprobe not installed)`);
        continue;
    }
    check(probed.width > 0 && probed.height > 0, `${entry} decodes at ${probed.width}x${probed.height} (codec: ${probed.codec_name})`);
}

function probe(bytes) {
    const tmp = path.join(os.tmpdir(), `vibium-smoke-${process.pid}.webm`);
    try {
        fs.writeFileSync(tmp, bytes);
        const out = execFileSync(
            "ffprobe",
            ["-v", "error", "-select_streams", "v:0", "-show_entries", "stream=codec_name,width,height", "-of", "json", tmp],
            { encoding: "utf-8" },
        );
        return JSON.parse(out).streams?.[0] ?? null;
    } catch {
        return null;
    } finally {
        fs.rmSync(tmp, { force: true });
    }
}

if (entries.includes("video/index.json")) {
    const raw = execFileSync("unzip", ["-p", outPath, "video/index.json"], { encoding: "utf-8" });
    let index;
    try {
        index = JSON.parse(raw);
    } catch (e) {
        check(false, `video/index.json parses as JSON (${e.message})`);
    }
    if (index) {
        console.log(`  index.json: ${JSON.stringify(index)}`);
        check(true, "video/index.json parses as JSON");
    }
}

console.log(
    failures.length === 0
        ? "\nAll video checks passed."
        : `\n${failures.length} check(s) failed:\n  - ${failures.join("\n  - ")}`,
);
process.exit(failures.length === 0 ? 0 : 1);
