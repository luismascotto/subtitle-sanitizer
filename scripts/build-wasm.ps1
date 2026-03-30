# Build standard Go wasm (cmd/wasm) and optionally TinyGo wasm (cmd/tinywasm).
# Serve dist/ over HTTP and open wasm-smoke.html to smoke-test in a browser (file:// often blocks WASM).

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
Set-Location $root

New-Item -ItemType Directory -Force -Path dist | Out-Null

$goroot = (& go env GOROOT).Trim()
Copy-Item -Force "$goroot\lib\wasm\wasm_exec.js" dist\wasm_exec.js

$env:GOOS = "js"
$env:GOARCH = "wasm"
try {
    go build -trimpath -o dist/sanitize-go.wasm ./cmd/wasm
} finally {
    Remove-Item Env:GOOS -ErrorAction SilentlyContinue
    Remove-Item Env:GOARCH -ErrorAction SilentlyContinue
}

$tiny = Get-Command tinygo -ErrorAction SilentlyContinue
if ($tiny) {
    tinygo build -o dist/sanitize-tinygo.wasm -target=wasm -scheduler=asyncify ./cmd/tinywasm
} else {
    Write-Host "tinygo not in PATH; skip sanitize-tinygo.wasm"
}

$distDir = Join-Path $root "dist"
$demo = Join-Path $root "web\wasm-demo"
New-Item -ItemType Directory -Force -Path $demo | Out-Null
Copy-Item -Force (Join-Path $distDir "wasm_exec.js"), (Join-Path $distDir "sanitize-go.wasm") $demo
if (Test-Path (Join-Path $distDir "sanitize-tinygo.wasm")) {
    Copy-Item -Force (Join-Path $distDir "sanitize-tinygo.wasm") $demo
}
Write-Host "wasm-demo: $demo  (e.g. npx --yes serve web/wasm-demo)"

Get-ChildItem dist\*.wasm | ForEach-Object { Write-Host ("{0}`t{1:N0} bytes" -f $_.Name, $_.Length) }
