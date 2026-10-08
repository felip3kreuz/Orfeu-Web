package wasmbridge

import (
	"encoding/json"
	"testing"

	core "jed-simulador/internal/core"
)

type testEnvelope struct {
	OK    bool            `json:"ok"`
	Data  json.RawMessage `json:"data"`
	Error string          `json:"error"`
}

func parseEnvelope(t *testing.T, raw string) testEnvelope {
	t.Helper()
	var env testEnvelope
	if err := json.Unmarshal([]byte(raw), &env); err != nil {
		t.Fatalf("invalid envelope: %v\n%s", err, raw)
	}
	return env
}

func bridgeFixture() core.Empresa {
	return core.Empresa{
		Nome: "Bridge", Dificuldade: "intermediario", Caixa: 10000,
		Preco: 80, PrecoReferencia: 80, CustoUnitario: 25,
		AlcanceBase: 500, ConversaoBase: .05, RecorrenciaBase: .20, CapacidadeBase: 50,
		Operacao: "presencial", Funcionarios: 1, SalarioMedio: 1500,
		MarketingSemanal: 250, CenarioAlcance: 1, CenarioConversao: 1,
		ConcorrenciaIndice: 1, Reputacao: 50, DuracaoSemanas: 12,
		InvestimentosIniciais: map[string]float64{},
	}
}

func TestProcessWeekReturnsUpdatedCompanyAndRecord(t *testing.T) {
	svc := NewService()
	created := parseEnvelope(t, svc.CreateSimulator("42"))
	if !created.OK {
		t.Fatalf("create failed: %s", created.Error)
	}
	var createdData struct {
		Handle int `json:"handle"`
	}
	if err := json.Unmarshal(created.Data, &createdData); err != nil {
		t.Fatal(err)
	}

	input, err := json.Marshal(bridgeFixture())
	if err != nil {
		t.Fatal(err)
	}
	out := parseEnvelope(t, svc.ProcessWeek(createdData.Handle, string(input)))
	if !out.OK {
		t.Fatalf("process failed: %s", out.Error)
	}

	var data struct {
		Empresa  core.Empresa  `json:"empresa"`
		Registro core.Registro `json:"registro"`
	}
	if err := json.Unmarshal(out.Data, &data); err != nil {
		t.Fatal(err)
	}
	if data.Empresa.Semana != 1 || data.Registro.Semana != 1 {
		t.Fatalf("expected week 1, company=%d record=%d", data.Empresa.Semana, data.Registro.Semana)
	}
	if len(data.Empresa.Historico) != 1 {
		t.Fatalf("expected one history item, got %d", len(data.Empresa.Historico))
	}
}

func TestSameSeedProducesSameJSON(t *testing.T) {
	a, b := NewService(), NewService()
	var ah, bh struct {
		Handle int `json:"handle"`
	}
	ea := parseEnvelope(t, a.CreateSimulator("123"))
	eb := parseEnvelope(t, b.CreateSimulator("123"))
	if err := json.Unmarshal(ea.Data, &ah); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(eb.Data, &bh); err != nil {
		t.Fatal(err)
	}
	input, _ := json.Marshal(bridgeFixture())
	ra := a.ProcessWeek(ah.Handle, string(input))
	rb := b.ProcessWeek(bh.Handle, string(input))
	if ra != rb {
		t.Fatalf("same seed produced different bridge JSON\nA=%s\nB=%s", ra, rb)
	}
}

func TestInvalidSimulatorHandleIsReported(t *testing.T) {
	svc := NewService()
	input, _ := json.Marshal(bridgeFixture())
	env := parseEnvelope(t, svc.ProcessWeek(999, string(input)))
	if env.OK || env.Error == "" {
		t.Fatalf("expected structured error, got %s", svc.ProcessWeek(999, string(input)))
	}
}

func TestBridgeStockAndSupplyOperations(t *testing.T) {
	svc := NewService()
	created := parseEnvelope(t, svc.CreateSimulator("77"))
	var createdData struct {
		Handle int `json:"handle"`
	}
	if err := json.Unmarshal(created.Data, &createdData); err != nil {
		t.Fatal(err)
	}

	company := bridgeFixture()
	company.UsaInsumos = true
	company.Insumos = []core.InsumoEstoque{{ID: "farinha", Nome: "Farinha", Unidade: "kg", Quantidade: 3, CustoMedio: 5, CustoReferencia: 5, ConsumoPorVenda: .2, Critico: true}}
	company.PrazoFornecedorMax = 2
	input, _ := json.Marshal(company)
	supplier, _ := json.Marshal(core.FornecedorSpec{ID: "padrao", Nome: "Fornecedor Padrão", MultiplicadorPreco: 1, PrazoEntregaSemanas: 1, Confiabilidade: 1, PrazoPagamentoMax: 2})
	env := parseEnvelope(t, svc.PlaceInputOrder(createdData.Handle, string(input), "farinha", "10", string(supplier), "0"))
	if !env.OK {
		t.Fatalf("pedido falhou: %s", env.Error)
	}
	var ordered struct {
		Empresa  core.Empresa `json:"empresa"`
		Accepted bool         `json:"accepted"`
	}
	if err := json.Unmarshal(env.Data, &ordered); err != nil {
		t.Fatal(err)
	}
	if !ordered.Accepted || len(ordered.Empresa.PedidosInsumos) != 1 {
		t.Fatalf("pedido não foi registrado: %#v", ordered)
	}

	stockCompany := bridgeFixture()
	stockCompany.UsaEstoque = true
	stockCompany.PrazoFornecedorMax = 2
	stockJSON, _ := json.Marshal(stockCompany)
	env = parseEnvelope(t, svc.BuyStock(string(stockJSON), "5", "0"))
	if !env.OK {
		t.Fatalf("compra de estoque falhou: %s", env.Error)
	}
	var bought struct {
		Empresa  core.Empresa `json:"empresa"`
		Accepted bool         `json:"accepted"`
	}
	if err := json.Unmarshal(env.Data, &bought); err != nil {
		t.Fatal(err)
	}
	if !bought.Accepted || bought.Empresa.EstoqueUnidades != 5 {
		t.Fatalf("estoque não atualizado: %#v", bought)
	}
}
