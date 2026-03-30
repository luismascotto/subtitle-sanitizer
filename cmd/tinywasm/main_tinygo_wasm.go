//go:build tinygo

package main

import (
	"syscall/js"

	"github.com/luismascotto/subtitle-sanitizer/internal/wasmbridge"
)

func main() {
	js.Global().Set("subtitleSanitizerProcess", js.FuncOf(process))
	select {}
}

func process(this js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return `{"ok":false,"error":"expected one argument: JSON request string"}`
	}
	out := wasmbridge.Process([]byte(args[0].String()))
	return string(out)
}
