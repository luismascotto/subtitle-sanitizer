# Cloudflare Pages — static WASM demo

This project’s WASM build is meant to run **only in the browser**. Cloudflare **Pages** serves `web/wasm-demo/` as plain static files (`index.html`, `wasm_exec.js`, `*.wasm`). **No Worker** is required for sanitize: all CPU work happens on the client.

## Prerequisite

From the repo root, produce the demo folder (WASM artifacts are gitignored until you build):

```bash
make wasm-pages
```

Windows:

```powershell
.\scripts\build-wasm.ps1
```

You should have `web/wasm-demo/index.html`, `wasm_exec.js`, and at least `sanitize-go.wasm`.

## Deploy with Wrangler

```bash
npx wrangler pages deploy web/wasm-demo --project-name=subtitle-sanitizer-wasm-demo
```

Or in the dashboard: **Workers & Pages → Create application → Pages → Direct upload**, and upload a zip of `web/wasm-demo/` **after** the build step above.

## Git-integrated Pages

Add a build command such as `make wasm-pages` (or `./scripts/build-wasm.sh`) and set the output directory to `web/wasm-demo`. Pages only needs to publish that directory; no Edge Worker step.

## Checks before production

- **Size:** Large `sanitize-go.wasm` benefits from gzip/brotli; Pages serves compressed assets when appropriate.
- **CORS:** Same-origin `fetch('./sanitize-go.wasm')` in `index.html` avoids cross-origin issues as long as HTML and WASM share the same site.
- **MIME:** Ensure `.wasm` is served as `application/wasm` (Wrangler/Pages defaults are fine for static uploads).

## JSON contract

The page calls `globalThis.subtitleSanitizerProcess(jsonString)`. Shapes are in `internal/wasmbridge` and `wasm/schema/*.json`.
