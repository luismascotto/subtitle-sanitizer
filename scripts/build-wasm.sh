#!/usr/bin/env bash
# Build Go (+ optional TinyGo) WASM and stage files for static hosting in web/wasm-demo/.
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

if command -v make >/dev/null 2>&1; then
  exec make wasm-pages
fi

echo "make not found; running inline build"
mkdir -p dist
cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" dist/
GOOS=js GOARCH=wasm go build -trimpath -o dist/sanitize-go.wasm ./cmd/wasm
if command -v tinygo >/dev/null 2>&1; then
  tinygo build -o dist/sanitize-tinygo.wasm -target=wasm -scheduler=asyncify ./cmd/tinywasm || true
fi
mkdir -p web/wasm-demo
cp dist/wasm_exec.js dist/sanitize-go.wasm web/wasm-demo/
test -f dist/sanitize-tinygo.wasm && cp dist/sanitize-tinygo.wasm web/wasm-demo/ || true
ls -la dist/*.wasm
