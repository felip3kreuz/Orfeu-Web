const parseEnvelope = (raw) => {
  const value = JSON.parse(raw);
  if (!value.ok) throw new Error(value.error || "Erro desconhecido no JED Core");
  return value.data;
};

async function instantiateGoWasm(go, wasmUrl) {
  const response = await fetch(wasmUrl);
  if (!response.ok) throw new Error(`Falha ao carregar ${wasmUrl}: HTTP ${response.status}`);

  if (WebAssembly.instantiateStreaming) {
    try {
      return await WebAssembly.instantiateStreaming(response.clone(), go.importObject);
    } catch (_) {
      // Some development servers send .wasm with the wrong MIME type.
    }
  }
  const bytes = await response.arrayBuffer();
  return WebAssembly.instantiate(bytes, go.importObject);
}

export async function loadJEDCore({ wasmUrl = "/wasm/jed-core.wasm", seed = Date.now() } = {}) {
  if (!globalThis.Go) {
    await import("./wasm_exec.js");
  }
  const go = new globalThis.Go();
  const result = await instantiateGoWasm(go, wasmUrl);
  void go.run(result.instance);

  for (let i = 0; i < 100 && !globalThis.JEDCore; i += 1) {
    await new Promise((resolve) => setTimeout(resolve, 0));
  }
  if (!globalThis.JEDCore) throw new Error("JED Core WASM não inicializou");

  const api = globalThis.JEDCore;
  const created = parseEnvelope(api.createSimulator(String(Math.trunc(seed))));
  const handle = created.handle;

  return {
    version: api.version,
    info: () => parseEnvelope(api.info()),
    processWeek: (empresa) => parseEnvelope(api.processWeek(handle, JSON.stringify(empresa))),
    indicators: (empresa) => parseEnvelope(api.indicators(JSON.stringify(empresa))),
    score: (empresa) => parseEnvelope(api.score(JSON.stringify(empresa))),
    review: (empresa) => parseEnvelope(api.review(JSON.stringify(empresa))),
    placeInputOrder: (empresa, inputID, quantity, supplier, term = 0) => parseEnvelope(api.placeInputOrder(handle, JSON.stringify(empresa), String(inputID), String(quantity), JSON.stringify(supplier), String(term))),
    buyStock: (empresa, quantity, term = 0) => parseEnvelope(api.buyStock(JSON.stringify(empresa), String(quantity), String(term))),
    journeyStep: (empresa) => parseEnvelope(api.journeyStep(JSON.stringify(empresa))),
    canvasExplanations: (empresa) => parseEnvelope(api.canvasExplanations(JSON.stringify(empresa))),
    digitalChannels: () => parseEnvelope(api.digitalChannels()),
    digitalTools: () => parseEnvelope(api.digitalTools()),
    close: () => parseEnvelope(api.destroySimulator(handle)),
  };
}
