# sanitize-subripr

Herdr plugin that prompts for an input `.srt` and an output `.srt`, then runs `sanitize --auto` to declutter subtitles (HIT noise, speech-to-text clutter, etc.).

## Prerequisites

```bash
go install github.com/luismascotto/subtitle-sanitizer/cmd/sanitize@latest
```

Ensure the Go bin directory is on your `PATH` so `sanitize` resolves.

## Link (local)

From the repo root (or this directory):

```bash
herdr plugin link ./plugins/sanitize-subripr
herdr plugin action list --plugin sanitize-subripr
herdr plugin action invoke sanitize-subripr.declutter
```

## Behavior

1. Action opens a popup pane.
2. Enter input and output `.srt` paths.
3. Runs `sanitize --auto --output <output> <input>` (input file is left unchanged).
