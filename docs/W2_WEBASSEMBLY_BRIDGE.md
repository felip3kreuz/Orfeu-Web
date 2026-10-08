# W2 — Ponte Go → WebAssembly

A fase W2 expõe o JED Core ao navegador sem duplicar as regras de negócio em JavaScript.

## Arquitetura

```
React / Next.js (W3)
        |
        | objetos JS / JSON
        v
web/wasm/jed-core.js
        |
        | strings JSON
        v
cmd/jed-wasm
        |
        v
internal/wasmbridge
        |
        v
internal/core
```

`internal/wasmbridge` é deliberadamente testável em Go nativo. `cmd/jed-wasm` contém somente a adaptação para `syscall/js`.

## API exposta no navegador

Após o módulo WASM inicializar, `globalThis.JEDCore` oferece:

- `info()`
- `createSimulator(seed)`
- `destroySimulator(handle)`
- `processWeek(handle, empresaJSON)`
- `indicators(empresaJSON)`
- `score(empresaJSON)`
- `review(empresaJSON)`
- `digitalChannels()`
- `digitalTools()`

Todas as chamadas retornam um envelope JSON:

```json
{"ok":true,"data":{}}
```

ou

```json
{"ok":false,"error":"mensagem"}
```

A camada `web/wasm/jed-core.js` esconde handles e envelopes do frontend. Em W3, o frontend usará `loadJEDCore()` e trabalhará com objetos JavaScript normais.

## Processamento de semana

`processWeek` recebe a empresa serializada e devolve, no mesmo retorno, a empresa mutada e o `Registro` da semana. Isso evita manter estado de negócio dentro do WASM e preserva a arquitetura definida em W1.5: o `Simulator` possui somente a fonte pseudoaleatória.

## Build

```bash
bash scripts/build-wasm.sh
```

Saída:

```
dist/wasm/jed-core.wasm
dist/wasm/wasm_exec.js
dist/wasm/jed-core.js
```

O workflow `Build WebAssembly` executa os testes, compila o módulo e publica esses três arquivos como artifact do GitHub Actions.

## Próxima fase

W3 criará a aplicação Next.js e copiará/servirá o artifact WASM no frontend da Vercel. Nessa fase surgirá o primeiro endereço web navegável do JED.
