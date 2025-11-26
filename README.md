# Subtitle Sanitizer (Go)

A small CLI tool to sanitize subtitle files by removing Hearing Impaired Text (HIT) and other noise. Initial target formats: `.srt` (implemented), `.ass` (stubbed for now). Always writes result as `.srt` with `-his` suffix to avoid damaging the original.

## Status
- SRT parse/write: ready
- ASS: stubbed (clear error)
- Rule: remove ALL UPPERCASE words (2+ chars)
- Drops cues that end up with no alphabetic content
- Renumbers cues on save
- JSON rules scaffold: in place (future work)

## Install
```bash
go build -o subtitle-sanitizer ./cmd/sanitize
```

## Usage
```bash
subtitle-sanitizer -input path/to/file.srt
```

Options:
- `-input` (required): file path to `.srt` or `.ass`
- `-encoding` (optional): defaults to `utf-8`
- `-verbose` (optional): extra logging
- `-ignoreErrors` (optional): continue best-effort on minor parse issues

Output:
- Saves to `path/to/file-his.srt`

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
