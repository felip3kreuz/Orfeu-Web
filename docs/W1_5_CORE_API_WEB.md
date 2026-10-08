# W1.5 — API do JED Core para frontends externos

## Objetivo

Encerrar a fase W1 deixando o motor do JED suficientemente independente da interface nativa para ser consumido por uma camada WebAssembly na fase W2.

A W1.5 não altera regras econômicas, protocolo online, formatos JSON ou a interface Win32. Ela move para `internal/core` funcionalidades puras que ainda estavam em `main.go` e cria uma fachada explícita para frontends não nativos.

## O que foi movido para o Core

### Análises agregadas

- cálculo de indicadores acumulados;
- cálculo do score JED;
- revisão quadrissemanal de hipóteses.

Arquivo: `internal/core/analytics.go`.

A interface existente continua usando `indicators`, `score` e `reviewHypotheses` por meio de wrappers em `engine_compat.go`.

### Catálogos

- contagem de especialidades;
- busca de modelo por ID;
- busca de fornecedor por ID;
- seleção/derivação do perfil de insumos.

Arquivo: `internal/core/catalog.go`.

Essas funções não dependem de filesystem. O carregamento dos arquivos JSON continua em `main.go` nesta etapa, mas, depois de carregados, os catálogos podem ser usados diretamente pelo Core em qualquer frontend.

## Nova fachada para WebAssembly

Foi criado `internal/core/api.go`, com o tipo:

```go
type Simulator struct { ... }
```

A API pública inicial é:

```go
core.NewSimulator(seed)
sim.ProcessWeek(empresa)
sim.Indicators(empresa)
sim.Score(empresa)
sim.Review(empresa)
sim.DigitalChannels()
sim.DigitalTools()
```

O `Simulator` mantém apenas a fonte pseudoaleatória. O estado do negócio permanece em `Empresa`, o que facilita transportar o estado como JSON entre Go e JavaScript.

A seed é explícita para que uma execução possa ser reproduzida em testes de paridade Windows/WebAssembly.

## Compatibilidade

`engine_compat.go` preserva os nomes esperados pela RC1.8. Portanto, a GUI Win32 e a interface Classic não precisam ser reescritas para consumir o Core nesta etapa.

Nenhuma mudança foi feita nos endpoints `/api/v1`, nos papéis Administrador/Mentor/Aluno ou no formato dos saves existentes.

## Testes adicionados

- estabilidade dos indicadores após histórico processado;
- score dentro do intervalo esperado;
- geração da revisão na quarta semana;
- replay determinístico pela nova fachada `Simulator`;
- funções de catálogo e derivação de perfil de insumos.

## Validação da W1.5

- `go test ./...` — OK
- `go run . --self-test` — OK
- build `windows/amd64` — OK
- build `js/wasm` — OK

## Estado após W1.5

A fase W1 é considerada encerrada.

O Core possui agora:

- tipos de domínio;
- cálculos determinísticos;
- aleatoriedade injetável;
- processamento semanal completo;
- parâmetros digitais;
- análises e score;
- regras puras de catálogo;
- uma fachada pequena para frontends externos.

O próximo marco é **W2 — ponte WebAssembly**, que deverá expor essa fachada ao JavaScript sem levar a interface Classic, filesystem local ou código Win32 para a API do navegador.
