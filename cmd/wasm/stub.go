//go:build !js || !wasm

package main

// Stub so `go build ./...` works on native targets. The real entrypoint is main_js_wasm.go.
func main() {}
