# Subtitle Sanitizer (Go)

A small CLI tool to sanitize subtitles by removing configurable levels of Hearing Impaired Text (HIT) and other noises. Supports SRT and ASS (produces a SRT result) formats. Can also be used for raw subtitle extraction from MKV files with the ffmpeg tool ("sold separatedly")


## Install
```bash
go build -o subtitle-sanitizer ./cmd/sanitize
```

## Usage
```bash
subtitle-sanitizer /path/to/file1.srt /path/to/file2.ass /path/to/file3.mkv
```

Options:
- `PATH1 [PATH2] [--mkv-extract, -m]`, (default false): extract all subtitles from files only
    --mkv-extract, -m: skip sanitization and extract all subtitles

Checks for config.json, and when not found, saves a config.backup.json with default options
For sanitization, detects MKV arg and extracts one subtitle (english, no sdh first on language/description tags) and forwards to the workflow.
A list of all affected cues is presented with original and modified content, along with each triggered rule description.

Output:
- Saves as `/path/to/file-his.srt` or `/path/to/file.srt`, depending on format and saving options

## WebAssembly (browser)

The same sanitize pipeline is exposed as JSON in/out via `internal/wasmbridge` (used by `cmd/wasm` and `cmd/tinywasm`).

- **Build:** `make wasm-pages` (Unix) or `scripts/build-wasm.ps1` (Windows). Copies `wasm_exec.js` and `sanitize-go.wasm` into `web/wasm-demo/` next to `index.html`.
- **Try locally:** `npx serve web/wasm-demo` and open the URL shown (must be HTTP, not `file://`).
- **JSON shapes:** `wasm/schema/request.schema.json` and `response.schema.json`.
- **Cloudflare Pages (static only):** see `cloudflare/README.md` — no Worker; WASM runs in the browser.
- **CI:** `.github/workflows/wasm.yml` runs tests and uploads a `wasm-demo` artifact.

## Roadmap
- Implement robust `.ass` parsing and conversion to SRT
- Expand rules via external JSON (regex-based, bracket text removal, etc.)
- Encoding detection & transcoding
- Batch processing directories
- Tests & CI
- MKV subtitle extraction with ffmpeg
- WASM

## Notes
Design emphasizes separation of concerns:
- `internal/mkv`: subtitle extraction
- `internal/model`: core data structures
- `internal/view`: core bubble tea workflow
- `internal/subtitle`: format-specific parsers/printers
- `internal/transform`: content transformations
- `internal/rules`: transformation rules config
- `internal/wasmbridge`: WASM definitions


## TODO
- Keep '-' when dialogue starts with speaker reference. "-PERSON: Hello" to "-Hello", not "Hello" (tweak on regex, add/refactor tests)