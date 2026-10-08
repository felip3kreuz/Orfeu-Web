# W1.3 — Aleatoriedade injetável no JED Core

A etapa W1.3 remove do motor extraído a dependência implícita de uma fonte global de aleatoriedade. O objetivo é permitir que o mesmo comportamento aleatório seja reproduzido por uma seed conhecida em testes, no executável nativo e, posteriormente, no WebAssembly.

## O que foi movido

As implementações canônicas das regras aleatórias agora vivem em `internal/core/engine_random.go`:

- intervalo aleatório usado pelo motor (`RandRange`);
- deriva semanal do índice de concorrência;
- seleção de eventos positivos e negativos;
- desperdício aleatório de insumos;
- confiabilidade/atraso aleatório de fornecedores ao fazer pedidos de insumos.

As tabelas de eventos positivos e negativos também deixaram `main.go` e passaram a pertencer ao Core.

## Fonte de aleatoriedade

O Core define a interface mínima:

```go
type RandomSource interface {
    Float64() float64
    Intn(n int) int
}
```

`*math/rand.Rand` atende a esse contrato. Assim, o Core não precisa criar nem conhecer um RNG global: a camada chamadora injeta a fonte desejada.

Nesta etapa, o pacote legado ainda mantém `rng` para preservar exatamente o ciclo de vida atual da RC1.8. A fachada `engine_compat.go` apenas encaminha esse RNG explicitamente para o Core. Quando `processWeek` for movido, ele poderá receber a mesma fonte diretamente.

## Compatibilidade

As assinaturas usadas pelo restante da RC1.8 foram preservadas por wrappers em `engine_compat.go`:

- `randRange`;
- `updateCompetition`;
- `selectEvent`;
- `wasteInputs`;
- `placeInputOrder`.

Nenhuma interface Win32, endpoint, arquivo JSON ou formato persistido foi alterado.

## Testes de equivalência

Antes de remover as implementações antigas de `main.go`, foi executado um teste temporário que comparou as versões legadas e as novas funções do Core com as mesmas empresas e as mesmas seeds. Foram comparados:

- valor de `randRange`;
- deriva de concorrência;
- evento escolhido;
- desperdício, detalhe textual e estoque resultante;
- pedido de fornecedor, custo, mensagem, contas a pagar e entrega prevista.

A comparação passou antes da remoção do código legado. O teste temporário foi então removido para que a suíte final valide somente a implementação canônica.

A suíte permanente `internal/core/engine_random_test.go` verifica injeção da fonte aleatória e repetibilidade com seed fixa.

Validações da W1.3:

```text
go test ./...                         OK
go run . --self-test                 OK
GOOS=windows GOARCH=amd64 go build   OK
GOOS=js GOARCH=wasm go build         OK
```

## Próxima etapa

Com cálculos determinísticos (W1.2) e aleatoriedade injetável (W1.3) já no Core, a W1.4 pode extrair o processamento semanal propriamente dito. A meta será transformar `processWeek` em uma operação do Core que recebe explicitamente a fonte de aleatoriedade, mantendo wrappers na aplicação legada até que Win32 e Web usem a nova API diretamente.
