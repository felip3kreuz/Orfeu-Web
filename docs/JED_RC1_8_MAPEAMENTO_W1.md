# JED Simulador v2.0 RC1.8 — Mapeamento W1

Base analisada: `JED_Simulador_v2_0_RC1_8_EmailConvites_Fonte(1).zip`

## Estado atual

- Módulo Go: `jed-simulador`, Go 1.23.
- Aproximadamente 10.177 linhas Go.
- Quase todo o produto ainda está no pacote `main`.
- `main.go`: 4.290 linhas — domínio, motor, persistência, CLI e UI Classic misturados.
- `native_windows.go`: 2.673 linhas — UI Win32, chamando diretamente tipos/funções de `main.go`.
- `online_server.go`: 1.671 linhas — servidor HTTP, autenticação, papéis e persistência do servidor.
- `online_client.go`: 454 linhas — cliente HTTP e DTOs do protocolo online.
- `email_server.go`: 494 linhas — SMTP e fluxo de credenciais Mentor.
- `go test ./...` passa na RC1.8 original.
- O conteúdo Go do pacote Fonte e do pacote GitHub Ready é idêntico.
- A aplicação inteira compila atualmente com `GOOS=js GOARCH=wasm`, gerando ~11 MB, mas ainda não é uma aplicação web utilizável.

## Objetivo da W1

Separar o motor e os contratos compartilhados sem alterar:

1. resultados da simulação;
2. formato JSON das empresas/cenários/turmas;
3. endpoints `/api/v1/...`;
4. papéis `admin`, `mentor`, `aluno` e compatibilidade `tutor`;
5. comportamento do JED Windows;
6. dados persistidos pela RC1.8.

## Estrutura-alvo

```text
JED-Simulador/
├─ cmd/
│  ├─ jed-windows/
│  │  └─ main.go
│  └─ jed-server/
│     └─ main.go
├─ internal/
│  ├─ core/
│  │  ├─ model.go
│  │  ├─ engine.go
│  │  ├─ difficulty.go
│  │  ├─ inventory.go
│  │  ├─ digital.go
│  │  ├─ scoring.go
│  │  ├─ migration.go
│  │  └─ random.go
│  ├─ catalog/
│  │  ├─ catalog.go
│  │  ├─ catalogo_negocios.json
│  │  └─ catalogo_insumos.json
│  ├─ storage/
│  │  ├─ json.go
│  │  ├─ company.go
│  │  ├─ scenario.go
│  │  ├─ class.go
│  │  └─ backup.go
│  ├─ protocol/
│  │  ├─ types.go
│  │  └─ roles.go
│  ├─ onlineclient/
│  │  └─ client.go
│  └─ server/
│     ├─ state.go
│     ├─ auth.go
│     ├─ handler.go
│     ├─ email.go
│     └─ cli.go
├─ ui/
│  ├─ classic/
│  └─ win32/
├─ web/
│  └─ (Next.js, criado depois da W1/W2)
├─ catalogo_negocios.json  (temporariamente, se necessário)
├─ catalogo_insumos.json   (temporariamente, se necessário)
└─ go.mod
```

## Mapeamento do `main.go`

### Vai para `internal/core`

Tipos de domínio:
- `LeanCanvas`
- `Persona`
- `Hipoteses`
- `Conta`
- `InsumoSpec`
- `PerfilInsumos`
- `FornecedorSpec`
- `CatalogoInsumos`
- `InsumoEstoque`
- `PedidoInsumo`
- `Registro`
- `Empresa`
- `Modelo`
- `TipoCatalogo`
- `SetorCatalogo`
- `Catalogo`
- `Cenario`
- `Turma`
- `Evento`
- `Mods`
- `DifficultyProfile`
- `Indicadores`
- `Score`
- `Review`

Regras puras ou quase puras:
- canais e ferramentas digitais;
- `canvasMods` e explicações;
- dificuldade;
- localização, sazonalidade, concorrência e capacidade;
- preço, marketing e lançamento;
- contas a pagar/receber;
- estoque e insumos;
- pedidos e desperdício;
- jornada digital;
- `processWeek`;
- indicadores;
- score;
- revisão de hipóteses.

### Vai para `internal/catalog`

- `loadCatalog`
- `loadSupplyCatalog`
- `supplyProfileFor`
- `findSupplier`
- `catalogCount`
- `findModel`
- arquivos JSON embutidos.

### Vai para `internal/storage`

- caminhos de dados;
- `saveJSON` / `loadJSON`;
- geração de IDs locais e timestamps;
- identidade/revisão da empresa;
- backup;
- `saveCompany` / carregamento;
- salvamento/listagem de cenários;
- salvamento/listagem de turmas;
- carregamento das empresas da turma;
- exportação de relatório/comparativo CSV.

### Migração

A lógica hoje contida em `loadCompany` que converte empresas antigas para o modelo de insumos da v2.0 deve sair para uma função explícita, por exemplo:

```go
func MigrateCompany(e *core.Empresa, catalogs catalog.Provider) error
```

O formato JSON não deve mudar nesta etapa.

### Permanece em UI Classic

Tudo que usa diretamente:
- `fmt.Print*` para apresentar tela;
- `readLine`, `askText`, `askFloat`, `askInt`, `askOption`, `askMultiple`;
- cabeçalhos, cores, barras e menus;
- wizards interativos;
- painel Classic;
- modo Tutor textual;
- tutorial e aparência.

## `native_windows.go`

Todo o arquivo é camada de apresentação e permanece fora do Core.

Na primeira refatoração, não deve ser reescrito. A forma mais segura é manter uma fachada de compatibilidade no pacote `main`, usando aliases e wrappers para os novos pacotes, de modo que a UI Win32 continue compilando com o mínimo de alterações.

Exemplo:

```go
type Empresa = core.Empresa
type Registro = core.Registro
type Cenario = core.Cenario

func processWeek(e *Empresa) Registro {
    return engine.ProcessWeek(e)
}
```

Depois que os testes confirmarem paridade, os aliases podem ser removidos gradualmente e a UI pode importar os pacotes diretamente.

## `online_client.go`

Dividir em:

- `internal/protocol/types.go`: `OnlineUser`, `OnlineClass`, `RemoteCompany`, convites e respostas;
- `internal/onlineclient/client.go`: chamadas HTTP;
- persistência de `OnlineConfig`: camada desktop, não protocolo.

É essencial preservar nomes dos campos JSON e endpoints da RC1.8.

## `online_server.go`

Dividir em:

- estado/persistência;
- autenticação e sessão;
- autorização/papéis;
- convites;
- administração;
- turmas;
- empresas/sincronização;
- handlers HTTP;
- CLI do servidor.

O servidor atual continuará sendo o backend canônico durante a primeira versão web.

## `email_server.go`

Fica com o servidor, não com o Core. Separar configuração SMTP, composição da mensagem e entrega. Os comandos interativos `--configure-email` e `--test-email` permanecem na CLI do servidor.

## Pontos que exigem cuidado

### 1. Aleatoriedade

O motor usa atualmente um `math/rand.Rand` global inicializado com `time.Now().UnixNano()`.

Para testes de paridade Windows/Web é melhor tornar a fonte aleatória uma dependência do motor, mantendo o comportamento padrão aleatório, mas permitindo seed fixa nos testes.

Não mudar a distribuição nem a sequência de chamadas na W1.

### 2. Catálogos embutidos

Hoje `main.go` usa `//go:embed` diretamente. O embed deve pertencer ao pacote `catalog`, para que Windows e WASM usem exatamente os mesmos dados.

### 3. Servidor usa tipos de domínio

`OnlineClass` contém `Cenario` e `RemoteCompany` contém `Empresa`. Portanto `protocol` deve depender de `core`, não o contrário.

Direção correta:

```text
core <- catalog
core <- storage
core <- protocol <- onlineclient
core <- protocol <- server
core <- Win32/Classic
core <- WASM
```

O `core` não deve importar servidor, HTTP, filesystem, Win32 ou UI.

### 4. Navegador e CORS

O servidor RC1.8 não implementa CORS. Isso não é um problema se a versão Vercel utilizar um BFF/proxy e o navegador falar apenas com `/api/...` no mesmo domínio da aplicação web.

### 5. Sessões

O desktop persiste Bearer Token em `conexao_online.json`. A aplicação web não deve copiar esse comportamento para `localStorage`; o BFF deverá manter a sessão em cookie HttpOnly/Secure e anexar o Bearer Token ao chamar o JED Servidor.

## Estratégia de refatoração segura

### W1.0 — baseline
- manter ZIP original intacto;
- `go test ./...`;
- `go run . --self-test`;
- registrar hashes dos catálogos.

### W1.1 — domínio
- criar `internal/core`;
- mover tipos de domínio;
- usar aliases temporários no pacote `main`;
- nenhuma mudança de JSON.

### W1.2 — motor
- mover regras matemáticas e `processWeek`;
- encapsular RNG;
- criar testes determinísticos com seed fixa.

### W1.3 — catálogo
- mover JSONs e embed para `internal/catalog`;
- garantir que os mesmos modelos sejam carregados.

### W1.4 — persistência/migração
- extrair filesystem;
- preservar arquivos existentes da RC1.8;
- testar carregamento de empresas antigas.

### W1.5 — protocolo online
- extrair DTOs e cliente HTTP;
- manter endpoints e payloads byte/semanticamente compatíveis.

### W1.6 — servidor
- separar `cmd/jed-server` e `internal/server`;
- manter o mesmo `JED_Servidor.exe` e dados `jed_server_data.json`.

### W2 — Windows sobre Core novo
- compilar todos os executáveis Windows;
- rodar testes e self-test;
- validar GUI Win32 sem regressões.

### W3 — WASM
- criar um pacote de exportação muito pequeno sobre `core`;
- expor JSON in/out ao frontend;
- não exportar filesystem, servidor ou UI.

## API WASM inicial sugerida

```text
jed.version()
jed.catalog()
jed.createCompany(inputJSON)
jed.loadCompany(companyJSON)
jed.processWeek(companyJSON, randomSeed?)
jed.indicators(companyJSON)
jed.score(companyJSON)
jed.review(companyJSON)
jed.migrateCompany(companyJSON)
```

Na aplicação real, o estado pode permanecer no JavaScript/IndexedDB e cada chamada ao WASM receber/retornar JSON. Isso mantém a fronteira simples e facilita testes.

## Critério de sucesso W1/W2

Uma refatoração só é aceita se:

1. `go test ./...` continuar verde;
2. `--self-test` continuar verde;
3. executáveis Windows continuarem sendo produzidos;
4. arquivos de empresa RC1.8 abrirem sem perda;
5. servidor antigo e cliente refatorado continuarem conversando;
6. um cenário com RNG fixa produzir os mesmos resultados antes e depois da extração.
