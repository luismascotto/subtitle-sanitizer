# Subtitle Sanitizer (Go)

A small CLI tool to sanitize subtitle files by removing Hearing Impaired Text (HIT) and other noise. Initial target formats: `.srt` (implemented), `.ass` (stubbed for now). Always writes result as `.srt` with `-his` suffix to avoid damaging the original.

## Status
- SRT parse/write: ready
- ASS: stubbed (clear error)
- Rule: remove ALL UPPERCASE words (2+ chars)
- Drops cues that end up with no alphabetic content
- Rewrite cues sequence at writing file to disk
- JSON rules scaffold: in place (future work)

## Install
```bash
go build -o subtitle-sanitizer ./cmd/sanitize
```

## Usage
```bash
subtitle-sanitizer path/to/file1.srt path/to/file2.srt 
```

Options:
- `PATH1 [PATH2] [--mkv-extract, -m]`, (default false): try to extract all subtitles from files, or try extract one (eng first) and try to sanitize it
- 

Output:
- Saves to `path/to/file-his.srt` or `path/to/file.srt`, depending on overwrite choice

## WebAssembly (browser)

The same sanitize pipeline is exposed as JSON in/out via `internal/wasmbridge` (used by `cmd/wasm` and `cmd/tinywasm`).

- **Build:** `make wasm-pages` (Unix) or `scripts/build-wasm.ps1` (Windows). Copies `wasm_exec.js` and `sanitize-go.wasm` into `web/wasm-demo/` next to `index.html`.
- **Try locally:** `npx serve web/wasm-demo` and open the URL shown (must be HTTP, not `file://`).
- **JSON shapes:** `wasm/schema/request.schema.json` and `response.schema.json`.
- **Cloudflare:** see `cloudflare/README.md`.
- **CI:** `.github/workflows/wasm.yml` runs tests and uploads a `wasm-demo` artifact.

## Roadmap
- Implement robust `.ass` parsing and conversion to SRT
- Expand rules via external JSON (regex-based, bracket text removal, etc.)
- Encoding detection & transcoding
- Batch processing directories
- Tests & CI

## Notes
Design emphasizes separation of concerns:
- `internal/io`: file reading/writing
- `internal/model`: core data structures
- `internal/subtitle`: format-specific parsers/printers
- `internal/transform`: content transformations
- `internal/rules`: rules config scaffold

# subtitle-sanitizer
Dotnet CLI tool to load ASS/SRT subtiles, apply rules to identify and remove text for hearing impaired and other little details. Inspired on  Subtitle Edit that I have to use GUI versinon for easch subtitle :)
