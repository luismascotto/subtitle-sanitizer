import { Capacitor } from "@capacitor/core";
import { Filesystem, Directory } from "@capacitor/filesystem";
import { Share } from "@capacitor/share";
import { defaultConfigText } from "./default-config.js";
import { isReady, loadWasm, processSubtitle } from "./wasm.js";

const statusEl = document.getElementById("status");
const changesEl = document.getElementById("changes");
const runBtn = document.getElementById("run");
const saveBtn = document.getElementById("save");
const fileNameEl = document.getElementById("fileName");
const fileInput = document.getElementById("subtitleFile");

let currentFileName = "";

document.getElementById("cfg").value = defaultConfigText;

function setStatus(text, isError = false) {
  statusEl.textContent = text;
  statusEl.classList.toggle("error", isError);
}

function outputFileName(sourceName) {
  if (!sourceName) return "subtitle-sanitized.srt";
  const dot = sourceName.lastIndexOf(".");
  const base = dot > 0 ? sourceName.slice(0, dot) : sourceName;
  return `${base}-sanitized.srt`;
}

async function readFile(file) {
  const text = await file.text();
  document.getElementById("sub").value = text;
  currentFileName = file.name;
  fileNameEl.textContent = file.name;
}

fileInput.addEventListener("change", async (event) => {
  const file = event.target.files?.[0];
  if (file) await readFile(file);
});

runBtn.addEventListener("click", () => {
  if (!isReady()) return;

  try {
    let cfgRaw = document.getElementById("cfg").value.trim();
    if (cfgRaw === "") cfgRaw = "{}";

    const resp = processSubtitle({
      subtitle: document.getElementById("sub").value,
      config: JSON.parse(cfgRaw),
    });

    if (!resp || resp.ok === false) {
      setStatus("Sanitizer returned an error", true);
      changesEl.textContent = JSON.stringify(resp, null, 2);
      saveBtn.disabled = true;
      return;
    }

    const srt = resp.srt || "";
    document.getElementById("srtOut").value = srt;
    saveBtn.disabled = !srt;
    changesEl.textContent = resp.changes
      ? JSON.stringify(resp.changes, null, 2)
      : "No changes.";
    setStatus(`Done (${resp.changes?.length ?? 0} changes)`);
  } catch (error) {
    setStatus(String(error), true);
    changesEl.textContent = String(error);
    saveBtn.disabled = true;
  }
});

async function saveOnNative(srt, filename) {
  const path = filename;
  await Filesystem.writeFile({
    path,
    data: srt,
    directory: Directory.Documents,
    encoding: "utf8",
  });

  const uri = (
    await Filesystem.getUri({
      path,
      directory: Directory.Documents,
    })
  ).uri;

  await Share.share({
    title: filename,
    text: "Sanitized subtitle",
    url: uri,
    dialogTitle: "Share sanitized subtitle",
  });
}

function saveInBrowser(srt, filename) {
  const blob = new Blob([srt], { type: "text/plain;charset=utf-8" });
  const url = URL.createObjectURL(blob);
  const anchor = document.createElement("a");
  anchor.href = url;
  anchor.download = filename;
  document.body.appendChild(anchor);
  anchor.click();
  anchor.remove();
  URL.revokeObjectURL(url);
}

saveBtn.addEventListener("click", async () => {
  const srt = document.getElementById("srtOut").value;
  if (!srt) return;

  const filename = outputFileName(currentFileName);

  try {
    if (Capacitor.isNativePlatform()) {
      await saveOnNative(srt, filename);
      setStatus(`Saved to Documents/${filename}`);
    } else {
      saveInBrowser(srt, filename);
      setStatus(`Downloaded ${filename}`);
    }
  } catch (error) {
    setStatus(`Save failed: ${error}`, true);
  }
});

(async () => {
  try {
    const wasmName = await loadWasm();
    runBtn.disabled = false;
    const platform = Capacitor.getPlatform();
    setStatus(`Ready (${wasmName}) · ${platform}`);
  } catch (error) {
    setStatus(`WASM load failed: ${error}`, true);
    changesEl.textContent = String(error);
  }
})();
