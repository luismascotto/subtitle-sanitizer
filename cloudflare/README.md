# Cloudflare (Pages / Workers) and WASM

## What runs where

- **Browser + static hosting (recommended first):** Build with `make wasm-pages` or `scripts/build-wasm.ps1`, then deploy the `web/wasm-demo/` folder (contains `index.html` + copied `wasm_exec.js` + `sanitize-go.wasm`). The sanitizer executes entirely in the visitor’s browser. **Cloudflare Pages** is a good fit: upload the directory or connect a Git build that runs the same copy step.

- **Workers executing Go’s `js/wasm` binary:** The official Go WASM runtime (`wasm_exec.js`) assumes a browser-like environment (e.g. `fetch`, timers). Workers are **not** a drop-in host for that stack; you would need a different compilation mode (e.g. WASI) or a tiny exported API built for Workers. Treat **TinyGo** + smaller WASM as the path to experiment with Worker bundle limits.

## Free-tier practical checks

- **Compressed size:** Upload limits and cold-start behavior depend on plan and product; compare `gzip -9 dist/sanitize-go.wasm` vs TinyGo output when testing.
- **CPU time:** Large ASS files increase work per request in the browser (same as local CLI complexity).
- **Memory:** Very large subtitles can stress mobile browsers before they stress Cloudflare.

## Pages deploy (example)

From repo root after a successful `make wasm-pages`:

```bash
npx wrangler pages deploy web/wasm-demo --project-name=subtitle-sanitizer-wasm-demo
```

Or use the dashboard: **Workers & Pages → Create → Pages → Direct Upload** and zip `web/wasm-demo/` (with WASM artifacts present).

## JSON API

The global `subtitleSanitizerProcess(jsonString)` contract matches `internal/wasmbridge` and `wasm/schema/*.json`.

## R2 / uploads (later)

Keeping binaries in R2 and returning signed download URLs is orthogonal: WASM still parses the subtitle bytes you pass in JSON (`subtitle` or `subtitleB64`).
