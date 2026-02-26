import { defineConfig } from "tsup";

export default defineConfig([
  // Main entry
  {
    entry: ["src/index.ts"],
    format: ["cjs", "esm"],
    dts: true,
    clean: true,
  },
  // Sync subpath entry
  {
    entry: { sync: "src/sync/index.ts" },
    format: ["cjs", "esm"],
    dts: true,
    outDir: "dist",
    clean: false,
  },
  // Worker entry (CJS only, bundled standalone)
  {
    entry: ["src/sync/worker.ts"],
    format: ["cjs"],
    outDir: "dist",
    clean: false, // Don't clean, main build already did
    noExternal: [/.*/], // Bundle all dependencies
  },
]);
