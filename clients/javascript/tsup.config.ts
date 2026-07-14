import { defineConfig } from "tsup";

export default defineConfig([
  // Main entry
  {
    entry: ["src/index.ts"],
    format: ["cjs", "esm"],
    dts: true,
    clean: true,
    // Inject a real `require` (via createRequire) into the ESM bundle so
    // `require.resolve` works when the package is loaded as ESM. Without it,
    // esbuild's stub `require` has no `.resolve`, throwing "require.resolve is
    // not a function" so the vibium binary can never be located (issue #62).
    shims: true,
  },
  // Sync subpath entry
  {
    entry: { sync: "src/sync/index.ts" },
    format: ["cjs", "esm"],
    dts: true,
    outDir: "dist",
    clean: false,
    shims: true,
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
