package core

import (
	"encoding/json"
	"math/rand"
	"reflect"
	"testing"
)

func weekTestFixture() Empresa {
	return Empresa{
		Nome: "Paridade", Dificuldade: "intermediario", Caixa: 15000,
		Preco: 85, PrecoReferencia: 80, CustoUnitario: 29,
		AlcanceBase: 760, ConversaoBase: .065, RecorrenciaBase: .21, CapacidadeBase: 65,
		SazonalidadeAmplitude: .08, SazonalidadeFase: .35,
		Operacao: "hibrida", AluguelMensal: 2200, QualidadeLocalizacao: "alta",
		Delivery: true, DeliveryTipo: "misto", DeliveryAfinidade: 1.05,
		Funcionarios: 2, SalarioMedio: 1800, MarketingSemanal: 420,
		MeiosPagamento: []string{"dinheiro", "pix", "cartao"}, TaxaCartao: 3.4, PrazoCartaoSemanas: 1,
		CenarioAlcance: 1.04, CenarioConversao: .98, CenarioOscilacao: .10,
		CenarioEventoNegExtra: .02, ConcorrenciaIndice: 1.03, Reputacao: 58,
		ReputacaoImportancia: 1.1, ClientesAtivos: 34, PromocaoDesconto: 8,
		Regulamentado: true, CustoRegulatorioMensal: 260, JurosMensal: 2.2, Divida: 5000,
		Canvas:              LeanCanvas{Segmentos: []string{"qualidade", "conveniencia"}, Canais: []string{"redes_sociais", "marketplace", "indicacao"}, PropostaValor: "qualidade", ReceitaModelo: "assinatura"},
		Persona:             Persona{Nome: "Ana", Demografia: "adultos", Rotinas: "correria", Objetivos: "ganhar tempo", Desafios: "preço", Motivadores: "qualidade", Objecoes: "prazo", Citacoes: "quero praticidade", PalavrasChave: "rápido"},
		CanaisDigitais:      []string{"instagram", "whatsapp", "mercado_livre", "ifood"},
		FerramentasDigitais: []string{"social_media", "copywriting", "design", "trafego_pago", "ia"},
		InvestimentosIniciais: map[string]float64{
			"equipamentos_referencia": 5000,
			"equipamentos":            5600,
			"marketing_lancamento":    900,
		},
		ContasReceber: []Conta{{Semana: 1, Valor: 320, Descricao: "antiga"}, {Semana: 3, Valor: 180, Descricao: "futura"}},
		ContasPagar:   []Conta{{Semana: 1, Valor: 140, Descricao: "antiga"}, {Semana: 4, Valor: 90, Descricao: "futura"}},
	}
}

func cloneWeekTestCompany(t *testing.T, e Empresa) Empresa {
	t.Helper()
	b, err := json.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}
	var out Empresa
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestProcessWeekDeterministicWithSameSeed(t *testing.T) {
	a := weekTestFixture()
	b := cloneWeekTestCompany(t, a)

	ra := ProcessWeek(&a, rand.New(rand.NewSource(42)))
	rb := ProcessWeek(&b, rand.New(rand.NewSource(42)))

	if !reflect.DeepEqual(ra, rb) {
		t.Fatalf("same seed produced different records\nA=%#v\nB=%#v", ra, rb)
	}
	if !reflect.DeepEqual(a, b) {
		t.Fatalf("same seed produced different company states\nA=%#v\nB=%#v", a, b)
	}
}

func TestProcessWeekAdvancesStateAndResetsPromotion(t *testing.T) {
	e := weekTestFixture()
	r := ProcessWeek(&e, rand.New(rand.NewSource(7)))

	if e.Semana != 1 || r.Semana != 1 {
		t.Fatalf("expected week 1, company=%d record=%d", e.Semana, r.Semana)
	}
	if len(e.Historico) != 1 || !reflect.DeepEqual(e.Historico[0], r) {
		t.Fatalf("weekly record was not appended to history")
	}
	if e.PromocaoDesconto != 0 {
		t.Fatalf("promotion discount should reset after processing, got %.2f", e.PromocaoDesconto)
	}
	if r.RecebimentosAnteriores != 320 || r.PagamentosAnteriores != 140 {
		t.Fatalf("matured accounts were not liquidated as expected: recv=%.2f pay=%.2f", r.RecebimentosAnteriores, r.PagamentosAnteriores)
	}
}

func TestDigitalCatalogAccessorsReturnCopies(t *testing.T) {
	channels := DigitalChannelSpecs()
	tools := DigitalToolSpecs()
	if len(channels) == 0 || len(tools) == 0 {
		t.Fatal("digital catalogs must not be empty")
	}
	originalChannel := channels[0].Nome
	originalTool := tools[0].Nome
	channels[0].Nome = "alterado"
	tools[0].Nome = "alterado"
	if DigitalChannelSpecs()[0].Nome != originalChannel {
		t.Fatal("channel accessor exposed mutable canonical slice")
	}
	if DigitalToolSpecs()[0].Nome != originalTool {
		t.Fatal("tool accessor exposed mutable canonical slice")
	}
}
