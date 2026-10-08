const parseEnvelope = (raw) => {
  const value = JSON.parse(raw);
  if (!value.ok) {
    throw new Error(value.error || "Erro desconhecido no JED Core");
  }
  return value.data;
};

let runtimePromise;

function loadClassicScript(src) {
  if (src.endsWith("wasm_exec.js") && globalThis.Go) {
    return Promise.resolve();
  }

  return new Promise((resolve, reject) => {
    const existing = document.querySelector(`script[src="${src}"]`);
    if (existing) {
      if (globalThis.Go) resolve();
      else {
        existing.addEventListener("load", resolve, { once: true });
        existing.addEventListener("error", reject, { once: true });
      }
      return;
    }

    const script = document.createElement("script");
    script.src = src;
    script.async = true;
    script.onload = () => resolve();
    script.onerror = () => reject(new Error(`Falha ao carregar ${src}`));
    document.head.appendChild(script);
  });
}

async function instantiateGoWasm(go, wasmUrl) {
  const response = await fetch(wasmUrl, { cache: "no-store" });
  if (!response.ok) {
    throw new Error(`Falha ao carregar ${wasmUrl}: HTTP ${response.status}`);
  }

  if (WebAssembly.instantiateStreaming) {
    try {
      return await WebAssembly.instantiateStreaming(response.clone(), go.importObject);
    } catch {
      // Fallback para servidores que não enviam application/wasm.
    }
  }

  const bytes = await response.arrayBuffer();
  return WebAssembly.instantiate(bytes, go.importObject);
}

async function ensureRuntime() {
  if (globalThis.JEDCore) return globalThis.JEDCore;
  if (runtimePromise) return runtimePromise;

  runtimePromise = (async () => {
    await loadClassicScript("/wasm/wasm_exec.js");
    if (!globalThis.Go) {
      throw new Error("Runtime Go WebAssembly não foi registrado");
    }

    const go = new globalThis.Go();
    const result = await instantiateGoWasm(go, "/wasm/jed-core.wasm");
    void go.run(result.instance);

    for (let i = 0; i < 100 && !globalThis.JEDCore; i += 1) {
      await new Promise((resolve) => setTimeout(resolve, 0));
    }

    if (!globalThis.JEDCore) {
      throw new Error("JED Core WASM não inicializou");
    }
    return globalThis.JEDCore;
  })();

  try {
    return await runtimePromise;
  } catch (error) {
    runtimePromise = undefined;
    throw error;
  }
}

export async function createJEDClient(seed = Date.now()) {
  const api = await ensureRuntime();
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
