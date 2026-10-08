import fs from "node:fs";
import path from "node:path";
import { pathToFileURL } from "node:url";

const root = process.cwd();
const out = path.join(root, "dist", "wasm");
await import(pathToFileURL(path.join(out, "wasm_exec.js")).href);

const go = new globalThis.Go();
const bytes = fs.readFileSync(path.join(out, "jed-core.wasm"));
const result = await WebAssembly.instantiate(bytes, go.importObject);
void go.run(result.instance);

for (let i = 0; i < 100 && !globalThis.JEDCore; i += 1) {
  await new Promise((resolve) => setTimeout(resolve, 0));
}
if (!globalThis.JEDCore) throw new Error("JEDCore global não foi registrado");

const info = JSON.parse(globalThis.JEDCore.info());
if (!info.ok || info.data?.protocol !== 1) throw new Error("info() inválido");
const created = JSON.parse(globalThis.JEDCore.createSimulator("42"));
if (!created.ok || created.data?.handle !== 1) throw new Error("createSimulator() inválido");

console.log(`WASM SMOKE OK | ${info.data.version} | protocol=${info.data.protocol}`);
process.exit(0);
