package core

import (
	"math"
	"reflect"
	"testing"
)

func almostEqual(a, b float64) bool { return math.Abs(a-b) < 1e-12 }

func testCompany() *Empresa {
	return &Empresa{
		Operacao:              "hibrida",
		Delivery:              true,
		Preco:                 90,
		PrecoReferencia:       100,
		Reputacao:             68,
		ReputacaoImportancia:  1.2,
		Dificuldade:           "avancado",
		QualidadeLocalizacao:  "alta",
		SazonalidadeAmplitude: .17,
		SazonalidadeFase:      .3,
		ConcorrenciaIndice:    1.13,
		CapacidadeBase:        40,
		Funcionarios:          3,
		MarketingSemanal:      525,
		Semana:                1,
		Divida:                12000,
		JurosMensal:           2.4,
		InvestimentosIniciais: map[string]float64{
			"equipamentos_referencia": 10000,
			"equipamentos":            12500,
			"marketing_lancamento":    900,
		},
		MeiosPagamento: []string{"pix", "cartao"},
		Canvas: LeanCanvas{
			Segmentos:     []string{"qualidade", "conveniencia", "digital", "b2b"},
			Canais:        []string{"redes_sociais", "indicacao", "marketplace", "prospeccao"},
			PropostaValor: "qualidade",
			ReceitaModelo: "assinatura",
			Problema:      "dor do cliente",
		},
		Persona: Persona{
			Nome: "Ana", Demografia: "x", Rotinas: "x", Objetivos: "x", Desafios: "x",
			Motivadores: "x", Objecoes: "x", Citacoes: "x", PalavrasChave: "x",
		},
		CanaisDigitais: []string{"instagram"},
		UsaInsumos:     true,
		Insumos: []InsumoEstoque{
			{ID: "a", Nome: "A", Unidade: "kg", Quantidade: 12, CustoMedio: 4, ConsumoPorVenda: 2, Critico: true},
			{ID: "b", Nome: "B", Unidade: "un", Quantidade: 20, CustoMedio: 1.5, ConsumoPorVenda: 1, Critico: true},
		},
		PedidosInsumos:     []PedidoInsumo{{InsumoID: "a", InsumoNome: "A", Quantidade: 3, CustoTotal: 15, SemanaEntrega: 1}},
		UsaEstoque:         true,
		PrazoFornecedorMax: 3,
		CustoUnitario:      7.25,
		Caixa:              1000,
	}
}

func TestDeterministicMarketFactors(t *testing.T) {
	e := testCompany()
	m := CanvasMods(e)
	if m.Alcance <= 1 || m.Conversao <= 1 || m.Recorrencia <= 1 {
		t.Fatalf("unexpected canvas modifiers: %#v", m)
	}
	if got := LocationFactor(e); got != 1.14 {
		t.Fatalf("LocationFactor = %v", got)
	}
	if got := CompetitionFactor(e); !almostEqual(got, .961) {
		t.Fatalf("CompetitionFactor = %v", got)
	}
	if got := EquipmentFactor(e); !almostEqual(got, 1.03) {
		t.Fatalf("EquipmentFactor = %v", got)
	}
	if got := WeeklyInterest(e); !almostEqual(got, 12000*.024/4.33) {
		t.Fatalf("WeeklyInterest = %v", got)
	}
	if got := LaunchFactor(e); !almostEqual(got, 1.3) {
		t.Fatalf("LaunchFactor = %v", got)
	}
	if got := Capacity(e, m); got < 1 {
		t.Fatalf("Capacity = %d", got)
	}
}

func TestDifficultyAndPayments(t *testing.T) {
	e := testCompany()
	if got := NormalizeDifficulty(" Avancado "); got != "avancado" {
		t.Fatalf("NormalizeDifficulty = %q", got)
	}
	dp := DifficultyProfileFor(e)
	if dp.Oscilacao != 1.30 || dp.ChanceEvento != .42 || dp.Desperdicio != 1.18 {
		t.Fatalf("DifficultyProfileFor = %#v", dp)
	}
	got := PaymentWeights(e)
	want := map[string]float64{"pix": 1.0 / 3.0, "cartao": 2.0 / 3.0}
	if !almostEqual(got["pix"], want["pix"]) || !almostEqual(got["cartao"], want["cartao"]) {
		t.Fatalf("PaymentWeights = %#v", got)
	}
}

func TestInputOperations(t *testing.T) {
	e := testCompany()
	note := ReceiveInputOrders(e)
	if note != "A: 3.0 kg" {
		t.Fatalf("ReceiveInputOrders = %q", note)
	}
	if len(e.PedidosInsumos) != 0 || e.Insumos[0].Quantidade != 15 {
		t.Fatalf("order not applied: %#v", e)
	}
	maxSales, limiting := MaxSalesByInputs(e)
	if maxSales != 7 || limiting != "A" {
		t.Fatalf("MaxSalesByInputs = %d %q", maxSales, limiting)
	}
	cmv := ConsumeInputs(e, 4)
	if !almostEqual(cmv, 39.6) {
		t.Fatalf("ConsumeInputs cost = %v", cmv)
	}
}

func TestStockPurchaseAndAllocation(t *testing.T) {
	e := testCompany()
	ok, cost, msg := BuyStock(e, 8, 2)
	if !ok || cost != 58 || msg != "compra a prazo (2 semana(s))" {
		t.Fatalf("BuyStock = %v %v %q", ok, cost, msg)
	}
	if len(e.ContasPagar) != 1 || e.ContasPagar[0].Semana != 3 || e.ContasPagar[0].Valor != 58 {
		t.Fatalf("unexpected accounts payable: %#v", e.ContasPagar)
	}
	n, r := AllocateSales(17, 13, 9)
	if n+r != 17 || n > 13 || r > 9 {
		t.Fatalf("AllocateSales = %d %d", n, r)
	}
}

func TestAccountsPersonaAndJourney(t *testing.T) {
	paid, future := Liquidate([]Conta{{Semana: 1, Valor: 10}, {Semana: 4, Valor: 20}}, 2)
	if paid != 10 || !reflect.DeepEqual(future, []Conta{{Semana: 4, Valor: 20}}) {
		t.Fatalf("Liquidate = %v %#v", paid, future)
	}
	e := testCompany()
	if got := PersonaCompleteness(e.Persona); got != 1 {
		t.Fatalf("PersonaCompleteness = %v", got)
	}
	if got := JourneyStep(e); got != 3 {
		t.Fatalf("JourneyStep = %d", got)
	}
	e.Semana = 4
	if got := JourneyStep(e); got != 4 {
		t.Fatalf("JourneyStep week 4 = %d", got)
	}
}
