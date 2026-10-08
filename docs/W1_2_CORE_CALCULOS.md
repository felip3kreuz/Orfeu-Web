# W1.2 — Cálculos determinísticos no JED Core

A etapa W1.2 inicia a extração efetiva do motor da RC1.8 para `internal/core`, sem alterar formatos JSON, protocolo do servidor ou comportamento das interfaces existentes.

## O que foi movido

As implementações canônicas das seguintes áreas agora vivem em `internal/core/engine.go`:

- modificadores do Lean Canvas;
- utilitários numéricos (`Clamp`, `Round2`, mínimo e máximo);
- fatores de localização, sazonalidade, concorrência e equipamentos;
- juros semanais, preço, marketing, lançamento e reputação;
- normalização e perfil de dificuldade;
- capacidade produtiva e pesos dos meios de pagamento;
- liquidação e soma de contas;
- avaliação e consumo determinístico de insumos;
- recebimento de pedidos de insumos;
- compra de estoque sem componente aleatório;
- alocação de vendas;
- completude de Persona e etapa da jornada digital.

Funções que dependem da fonte global de aleatoriedade, catálogos ainda mantidos em `main`, filesystem, terminal, Win32 ou rede permanecem fora do Core nesta etapa. Isso inclui, entre outras, seleção de eventos, variação de concorrência, desperdício aleatório, atraso de fornecedor e `processWeek`.

## Fachada de compatibilidade

`engine_compat.go` mantém os nomes usados pela RC1.8 (`canvasMods`, `clamp`, `capacity`, `consumeInputs` etc.) e encaminha cada chamada para `internal/core`. Assim, a interface Win32 e o restante do programa não precisam ser reescritos simultaneamente.

## Testes

Foi adicionada uma suíte própria em `internal/core/engine_test.go` para validar fatores determinísticos, dificuldade, pagamentos, insumos, estoque, contas, Persona e jornada.

Antes da remoção das implementações antigas, uma comparação temporária executou as versões legadas e as novas funções do Core com os mesmos dados e confirmou equivalência nos casos cobertos. O teste temporário não faz parte do pacote final porque, após a extração, os wrappers apontam diretamente para o Core.

Validações da W1.2:

```text
go test ./...                         OK
go run . --self-test                 OK
GOOS=windows GOARCH=amd64 go build   OK
GOOS=js GOARCH=wasm go build         OK
```

## Próxima etapa

W1.3 deve isolar a fonte de aleatoriedade do motor. O objetivo é substituir a dependência direta do `math/rand.Rand` global por uma fonte injetável/seedável, permitindo testes determinísticos de eventos, concorrência, fornecedores, desperdício e, posteriormente, de `processWeek` completo em Go nativo e WebAssembly.
