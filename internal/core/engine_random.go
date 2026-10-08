package core

import (
	"fmt"
	"math"
	"strings"
)

// RandomSource is the minimal random-number contract required by the JED Core.
// *math/rand.Rand satisfies this interface. Keeping the interface in the Core
// allows callers (native, WASM and tests) to inject a seeded source without the
// domain engine depending on a package-global RNG.
type RandomSource interface {
	Float64() float64
	Intn(n int) int
}

var positiveEvents = []Evento{
	{"Uma publicação sobre a empresa teve excelente alcance nas redes sociais.", 1.22, 1.00, 3, 0},
	{"Clientes recomendaram sua empresa para outras pessoas.", 1.08, 1.07, 4, 0},
	{"A procura pelo seu tipo de produto ou serviço aumentou nesta semana.", 1.12, 1.05, 0, 0},
	{"Um concorrente próximo encerrou as atividades.", 1.04, 1.08, 0, -0.10},
}

var negativeEvents = []Evento{
	{"Um concorrente iniciou uma promoção agressiva nesta semana.", 0.98, 0.90, 0, 0.10},
	{"Uma avaliação negativa ganhou visibilidade.", 0.98, 0.94, -4, 0},
	{"O movimento do mercado caiu inesperadamente nesta semana.", 0.88, 0.96, 0, 0},
	{"Um novo concorrente entrou no mercado.", 0.98, 0.92, 0, 0.12},
}

func RandRange(r RandomSource, a, b float64) float64 {
	return a + r.Float64()*(b-a)
}

func UpdateCompetition(e *Empresa, r RandomSource, delta float64) {
	dp := DifficultyProfileFor(e)
	e.ConcorrenciaIndice = Clamp(e.ConcorrenciaIndice+RandRange(r, -.025, .025)*dp.ConcorrenciaDrift+delta, .60, 1.50)
}

func SelectEvent(e *Empresa, r RandomSource) *Evento {
	dp := DifficultyProfileFor(e)
	if r.Float64() >= dp.ChanceEvento {
		return nil
	}
	neg := Clamp(.50+e.CenarioEventoNegExtra+dp.NegExtra, .15, .85)
	if r.Float64() < neg {
		x := negativeEvents[r.Intn(len(negativeEvents))]
		return &x
	}
	x := positiveEvents[r.Intn(len(positiveEvents))]
	return &x
}

func formatMoneyBR(v float64) string {
	neg := v < 0
	if neg {
		v = -v
	}
	s := fmt.Sprintf("%.2f", v)
	parts := strings.Split(s, ".")
	ip := parts[0]
	for i := len(ip) - 3; i > 0; i -= 3 {
		ip = ip[:i] + "." + ip[i:]
	}
	res := "R$ " + ip + "," + parts[1]
	if neg {
		res = "-" + res
	}
	return res
}

func WasteInputs(e *Empresa, r RandomSource) (float64, string) {
	if !e.UsaInsumos {
		return 0, ""
	}
	total := 0.0
	parts := []string{}
	for i := range e.Insumos {
		x := &e.Insumos[i]
		if x.PerdaSemanal <= 0 || x.Quantidade <= 0 {
			continue
		}
		rate := Clamp(x.PerdaSemanal*DifficultyProfileFor(e).Desperdicio*RandRange(r, .75, 1.25), 0, .50)
		q := x.Quantidade * rate
		if q < 0.01 {
			continue
		}
		v := q * x.CustoMedio
		x.Quantidade = math.Max(0, x.Quantidade-q)
		total += v
		parts = append(parts, fmt.Sprintf("%s %.1f %s (%s)", x.Nome, q, x.Unidade, formatMoneyBR(v)))
	}
	e.EstoqueValor = StockValueInputs(e)
	return total, strings.Join(parts, "; ")
}

func PlaceInputOrder(e *Empresa, inputID string, q float64, f FornecedorSpec, term int, r RandomSource) (bool, float64, string) {
	if q <= 0 {
		return false, 0, "quantidade inválida"
	}
	i := FindInputIndex(e, inputID)
	if i < 0 {
		return false, 0, "insumo não encontrado"
	}
	maxTerm := MinInt(e.PrazoFornecedorMax, f.PrazoPagamentoMax)
	if term < 0 {
		term = 0
	}
	if term > maxTerm {
		term = maxTerm
	}
	baseCost := e.Insumos[i].CustoReferencia
	if baseCost <= 0 {
		baseCost = e.Insumos[i].CustoMedio
	}
	if baseCost <= 0 {
		baseCost = 1
	}
	unit := baseCost * f.MultiplicadorPreco
	cost := q * unit
	if term == 0 && cost > e.Caixa {
		return false, cost, "caixa insuficiente"
	}
	if term == 0 {
		e.Caixa -= cost
	} else {
		e.ContasPagar = append(e.ContasPagar, Conta{Semana: e.Semana + term, Valor: Round2(cost), Descricao: fmt.Sprintf("%s — %s", f.Nome, e.Insumos[i].Nome)})
	}
	delay := f.PrazoEntregaSemanas
	delayed := false
	reliability := Clamp(f.Confiabilidade+DifficultyProfileFor(e).FornecedorBonus, .50, .995)
	if r.Float64() > reliability {
		delay++
		delayed = true
	}
	if delay == 0 {
		oldQ := e.Insumos[i].Quantidade
		oldV := oldQ * e.Insumos[i].CustoMedio
		e.Insumos[i].Quantidade += q
		e.Insumos[i].CustoMedio = (oldV + cost) / e.Insumos[i].Quantidade
		e.EstoqueValor = StockValueInputs(e)
		msg := "entrega imediata"
		if delayed {
			msg = "entrega imediata (fornecedor sofreu atraso, mas entregou no mesmo ciclo)"
		}
		return true, cost, msg
	}
	e.PedidosInsumos = append(e.PedidosInsumos, PedidoInsumo{InsumoID: inputID, InsumoNome: e.Insumos[i].Nome, Quantidade: q, CustoUnitario: unit, CustoTotal: Round2(cost), Fornecedor: f.Nome, SemanaEntrega: e.Semana + delay})
	msg := fmt.Sprintf("entrega prevista para a semana %d", e.Semana+delay)
	if delayed {
		msg += " — houve atraso do fornecedor"
	}
	return true, cost, msg
}
