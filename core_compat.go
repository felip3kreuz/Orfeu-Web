package main

// W1.1 compatibility facade.
//
// The Win32/classic code still belongs to package main in RC1.8. Aliases keep
// its existing source API intact while the canonical domain types now live in
// internal/core. Remove these aliases gradually as UI and engine packages are
// extracted in subsequent W1 steps.

import core "jed-simulador/internal/core"

type LeanCanvas = core.LeanCanvas
type Persona = core.Persona
type CanalDigitalSpec = core.CanalDigitalSpec
type FerramentaDigitalSpec = core.FerramentaDigitalSpec
type Hipoteses = core.Hipoteses
type Conta = core.Conta
type InsumoSpec = core.InsumoSpec
type PerfilInsumos = core.PerfilInsumos
type FornecedorSpec = core.FornecedorSpec
type CatalogoInsumos = core.CatalogoInsumos
type InsumoEstoque = core.InsumoEstoque
type PedidoInsumo = core.PedidoInsumo
type Registro = core.Registro
type Empresa = core.Empresa
type Modelo = core.Modelo
type TipoCatalogo = core.TipoCatalogo
type SetorCatalogo = core.SetorCatalogo
type Catalogo = core.Catalogo
type Cenario = core.Cenario
type Turma = core.Turma
type Evento = core.Evento
type Indicadores = core.Indicadores
type Score = core.Score
type Review = core.Review
