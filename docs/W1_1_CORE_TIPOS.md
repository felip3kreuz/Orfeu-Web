# W1.1 — Extração dos tipos de domínio

Base: JED Simulador v2.0.0-rc1.8.

## Objetivo

Criar o primeiro limite arquitetural do JED Core sem alterar regras de simulação, protocolo online ou comportamento das interfaces existentes.

## Alterações

- criado `internal/core/types.go` como fonte canônica dos principais tipos de domínio;
- criado `core_compat.go` no pacote `main`, com aliases temporários para preservar a API usada pela interface Win32, interface Classic, servidor e testes existentes;
- removidas de `main.go` as declarações duplicadas dos tipos agora pertencentes ao Core;
- nenhuma regra de cálculo foi movida nesta etapa;
- nenhuma versão de dados, endpoint ou formato JSON foi alterado.

## Tipos transferidos

`LeanCanvas`, `Persona`, `CanalDigitalSpec`, `FerramentaDigitalSpec`, `Hipoteses`, `Conta`, `InsumoSpec`, `PerfilInsumos`, `FornecedorSpec`, `CatalogoInsumos`, `InsumoEstoque`, `PedidoInsumo`, `Registro`, `Empresa`, `Modelo`, `TipoCatalogo`, `SetorCatalogo`, `Catalogo`, `Cenario`, `Turma`, `Evento`, `Indicadores`, `Score` e `Review`.

## Verificações executadas

```text
go test ./...                         OK
GOOS=windows GOARCH=amd64 go build   OK
GOOS=js GOARCH=wasm go build         OK
```

## Próxima etapa

W1.2: mover regras puras do motor para `internal/core`, começando por utilitários matemáticos e cálculo de indicadores/score, antes de migrar `processWeek`.
