# Subtitle Sanitizer v0.0.1-alpha

**First public alpha** of **Subtitle Sanitizer**, a small Go tool (and browser WASM demo) that cleans subtitle files by stripping Hearing Impaired–style text and other noise. Output is always **SRT** with a **`-his`** suffix so originals stay untouched (unless you choose to overwrite).

## Highlights

- **SRT pipeline** — Parse, transform, and write SRT with **cue renumbering** on save.
- **Sanitization rules** — Removes **ALL CAPS words** (2+ characters); **drops cues** that have no alphabetic text left after cleaning.
- **Interactive CLI** — Terminal UI built with **Bubble Tea** for a guided workflow (paths, overwrite, etc.).
- **Optional MKV path** — `--mkv-extract` / `-m` to try extracting subtitles from Matroska inputs (English preference, non-SDH when possible), then run the same sanitize flow.
- **ASS** — Recognized but **not implemented** yet; fails with a clear message.
- **JSON rules scaffold** — Present in-tree for future configurable rules (not the main story in this alpha).
- **WebAssembly demo** — Same sanitize logic exposed as **JSON in/out** (`wasm/schema/request.schema.json` and `response.schema.json`), with a static demo under `web/wasm-demo/`. Build via `make wasm-pages` (Unix) or `scripts/build-wasm.ps1` (Windows); serve over HTTP (not `file://`). **Cloudflare Pages**-ready; see `cloudflare/README.md`.
- **CI** — `go test ./...` and WASM demo staging on `main` (`.github/workflows/wasm.yml`).

## Install (from source)

```bash
go build -o subtitle-sanitizer ./cmd/sanitize
```

```bash
subtitle-sanitizer path/to/file1.srt [path/to/file2 ...] [--mkv-extract|-m]
```

Module path: `github.com/luismascotto/subtitle-sanitizer` (Go **1.25.6** per `go.mod`).

## Alpha disclaimer

This is an **early alpha**: behavior and flags may change, **ASS support is stubbed**, and features like richer JSON rules, encoding detection, directory batching, and deeper MKV/ffmpeg integration are **roadmap**, not guarantees for this tag. Feedback and issue reports are welcome.

## Thanks

Thanks for trying the project; issues and PRs on GitHub help shape the next releases.
