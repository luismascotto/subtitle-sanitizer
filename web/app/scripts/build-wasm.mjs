import { cpSync, existsSync, mkdirSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { execSync } from "node:child_process";

const __dirname = dirname(fileURLToPath(import.meta.url));
const appDir = join(__dirname, "..");
const publicDir = join(appDir, "public");
const repoRoot = join(appDir, "..", "..");
const distDir = join(repoRoot, "dist");

function runGoWasmBuild() {
  const goroot = execSync("go env GOROOT", { cwd: repoRoot, encoding: "utf8" }).trim();
  mkdirSync(distDir, { recursive: true });
  cpSync(join(goroot, "lib", "wasm", "wasm_exec.js"), join(distDir, "wasm_exec.js"));

  execSync("go build -trimpath -o dist/sanitize-go.wasm ./cmd/wasm", {
    cwd: repoRoot,
    env: { ...process.env, GOOS: "js", GOARCH: "wasm" },
    stdio: "inherit",
  });
}

function copyArtifacts() {
  mkdirSync(publicDir, { recursive: true });
  for (const name of ["wasm_exec.js", "sanitize-go.wasm", "sanitize-tinygo.wasm"]) {
    const src = join(distDir, name);
    if (existsSync(src)) {
      cpSync(src, join(publicDir, name));
    }
  }
}

console.log("Building Go WASM from repo root…");
runGoWasmBuild();
copyArtifacts();
console.log("WASM assets copied to web/app/public/");
