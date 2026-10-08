"use client";

import { useEffect, useState } from "react";
import { createJEDClient } from "@/lib/jed-core";

export default function CoreStatus() {
  const [state, setState] = useState({ status: "loading" });

  useEffect(() => {
    let cancelled = false;
    let client;

    (async () => {
      try {
        client = await createJEDClient(20261007);
        const info = client.info();
        const channels = client.digitalChannels();
        const tools = client.digitalTools();

        if (!cancelled) {
          setState({
            status: "ready",
            version: info.version,
            protocol: info.protocol,
            channels: Array.isArray(channels) ? channels.length : 0,
            tools: Array.isArray(tools) ? tools.length : 0,
          });
        }
      } catch (error) {
        if (!cancelled) {
          setState({
            status: "error",
            message: error instanceof Error ? error.message : String(error),
          });
        }
      }
    })();

    return () => {
      cancelled = true;
      try {
        client?.close();
      } catch {
        // A desmontagem não deve derrubar a página.
      }
    };
  }, []);

  if (state.status === "loading") {
    return (
      <div className="core-status core-loading" role="status">
        <span className="status-dot" />
        Inicializando motor de simulação em WebAssembly…
      </div>
    );
  }

  if (state.status === "error") {
    return (
      <div className="core-status core-error" role="alert">
        <strong>Motor de simulação não carregou.</strong>
        <span>{state.message}</span>
      </div>
    );
  }

  return (
    <div className="core-status core-ready" role="status">
      <div className="status-heading">
        <span className="status-dot" />
        <strong>Motor Orfeu ativo no navegador</strong>
      </div>
      <div className="core-metrics">
        <span>Versão {state.version}</span>
        <span>Protocolo {state.protocol}</span>
        <span>{state.channels} canais digitais</span>
        <span>{state.tools} ferramentas digitais</span>
      </div>
    </div>
  );
}
