.PHONY: test wasm wasm-tinygo wasm-pages clean-wasm

# Native tests (includes wasmbridge golden fixture).
test:
	go test ./...

# Official Go toolchain: dist/sanitize-go.wasm + wasm_exec.js (+ optional TinyGo artifact).
wasm:
	@mkdir -p dist
	cp "$$(go env GOROOT)/lib/wasm/wasm_exec.js" dist/
	GOOS=js GOARCH=wasm go build -trimpath -o dist/sanitize-go.wasm ./cmd/wasm
	@if command -v tinygo >/dev/null 2>&1; then \
		tinygo build -o dist/sanitize-tinygo.wasm -target=wasm -scheduler=asyncify ./cmd/tinywasm && echo "tinygo: dist/sanitize-tinygo.wasm" || echo "tinygo: build failed (ignored)"; \
	else \
		echo "tinygo: not in PATH, skip"; \
	fi
	@ls -la dist/*.wasm 2>/dev/null || true

wasm-tinygo:
	@command -v tinygo >/dev/null 2>&1 || (echo "install tinygo: https://tinygo.org/getting-started/install/" && exit 1)
	@mkdir -p dist
	cp "$$(go env GOROOT)/lib/wasm/wasm_exec.js" dist/
	tinygo build -o dist/sanitize-tinygo.wasm -target=wasm -scheduler=asyncify ./cmd/tinywasm

# Copy WASM assets next to web/wasm-demo/index.html for static hosting (Pages, any CDN).
wasm-pages: wasm
	@mkdir -p web/wasm-demo
	cp dist/wasm_exec.js dist/sanitize-go.wasm web/wasm-demo/
	@test -f dist/sanitize-tinygo.wasm && cp dist/sanitize-tinygo.wasm web/wasm-demo/ || true
	@echo "Ready: npx --yes serve web/wasm-demo  then open http://localhost:3000"

clean-wasm:
	rm -f dist/*.wasm dist/wasm_exec.js
	rm -f web/wasm-demo/wasm_exec.js web/wasm-demo/sanitize-go.wasm web/wasm-demo/sanitize-tinygo.wasm
