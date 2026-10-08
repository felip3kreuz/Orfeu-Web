# W1.4 — Processamento semanal no JED Core

A etapa W1.4 transfere para `internal/core` a transição semanal completa da empresa. Com isso, o motor que altera caixa, demanda, vendas, estoque, clientes, reputação e histórico deixa de pertencer ao pacote `main` e passa a ser uma operação de domínio reutilizável por Windows e, posteriormente, WebAssembly.

## O que foi movido

A implementação canônica de `processWeek` agora é:

```go
func ProcessWeek(e *Empresa, r RandomSource) Registro
```

em `internal/core/week.go`.

A função recebe explicitamente a fonte de aleatoriedade introduzida na W1.3. Portanto, uma mesma empresa processada com a mesma seed produz o mesmo resultado independentemente da camada de interface.

Também foram movidas para `internal/core/digital.go` as regras e os parâmetros digitais usados pelo processamento semanal:

- catálogo de canais digitais;
- catálogo de ferramentas digitais;
- custo semanal das ferramentas;
- modificadores de alcance, engajamento, conversão e recorrência;
- modificadores dos canais digitais;
- identificação de canal/ferramenta;
- custo fixo semanal da empresa.

A UI Win32 continua usando os nomes legados através da fachada `engine_compat.go`.

## Compatibilidade

O restante da RC1.8 ainda chama:

```go
processWeek(e)
```

A fachada mantém essa assinatura e encaminha para:

```go
core.ProcessWeek(e, rng)
```

O RNG global continua existindo apenas na aplicação legada para preservar seu ciclo de vida atual. O Core não depende dele.

Da mesma forma, `canaisDigitais`, `ferramentasDigitais`, `digitalToolCost`, `digitalToolMods`, `digitalChannelMods`, `fixedWeekly` e auxiliares continuam disponíveis ao código Win32 por wrappers, mas suas regras canônicas agora pertencem ao Core.

Nenhum formato JSON, endpoint HTTP, papel de usuário ou fluxo da interface foi alterado.

## Teste de paridade com W1.3

Antes da remoção da implementação antiga, foi executado um teste temporário comparando `processWeek` legado com `core.ProcessWeek` para as mesmas empresas e as mesmas seeds.

Foram cobertos três perfis:

- empresa de serviço sem estoque;
- empresa com estoque e perecibilidade;
- empresa com insumos e pedido pendente.

Cada perfil foi executado com múltiplas seeds. Tanto o `Registro` retornado quanto o estado final completo de `Empresa` foram comparados com igualdade estrutural. A paridade foi confirmada antes da remoção do código legado.

O teste temporário não faz parte do pacote final porque a implementação antiga deixou de existir. A suíte permanente em `internal/core/week_test.go` verifica:

- repetibilidade com a mesma seed;
- avanço da semana e inclusão no histórico;
- liquidação de contas vencidas;
- reset da promoção após o processamento;
- proteção contra mutação acidental dos catálogos digitais canônicos.

## Validações da W1.4

```text
go test ./...                         OK
go run . --self-test                 OK
GOOS=windows GOARCH=amd64 go build   OK
GOOS=js GOARCH=wasm go build         OK
```

## Estado arquitetural após W1.4

```text
Win32 / Classic
      │
      ▼
engine_compat.go
      │
      ▼
internal/core
  ├─ types.go
  ├─ engine.go
  ├─ engine_random.go
  ├─ digital.go
  └─ week.go
```

O processamento semanal já não depende de terminal, Win32, filesystem, HTTP ou servidor.

## Próxima etapa

A W1.5 pode extrair indicadores, score e revisão de hipóteses para o Core. Isso remove outro bloco grande de lógica de domínio de `main.go` e aproxima o pacote `main` de uma camada apenas de aplicação/interface.
