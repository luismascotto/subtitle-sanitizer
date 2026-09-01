const go = new Go();

let ready = false;
let currentWasm = "sanitize-go.wasm";

export function isReady() {
  return ready;
}

export async function loadWasm(wasmFile = "sanitize-go.wasm") {
  currentWasm = wasmFile;
  ready = false;

  const res = await fetch(`./${wasmFile}`);
  if (!res.ok) {
    throw new Error(`${res.status} ${res.statusText} loading ${wasmFile}`);
  }

  const result = await WebAssembly.instantiateStreaming(res, go.importObject);
  go.run(result.instance);
  ready = true;
  return currentWasm;
}

export function processSubtitle(body) {
  if (!ready) {
    throw new Error("WASM not loaded");
  }

  const raw = globalThis.subtitleSanitizerProcess(JSON.stringify(body));
  const text = typeof raw === "string" ? raw : String(raw);
  return JSON.parse(text);
}
