package core

import (
	"fmt"
	"math"
)

// Indicators aggregates all completed weeks into the dashboard metrics used by
// every JED frontend. It returns nil while the company has no processed week.
func Indicators(e *Empresa) *Indicadores {
	if len(e.Historico) == 0 {
		return nil
	}
	i := Indicadores{
		Semanas:        len(e.Historico),
		ClientesAtivos: e.ClientesAtivos,
		Caixa:          Round2(e.Caixa),
		AReceber:       Round2(SumAccounts(e.ContasReceber)),
		APagar:         Round2(SumAccounts(e.ContasPagar)),
	}
	marketing := 0.0
	for _, r := range e.Historico {
		i.Vendas += r.Vendas
		i.Receita += r.Receita
		i.Resultado += r.Resultado
		i.Novos += r.NovosClientes
		i.Recorrentes += r.ClientesRecorrentes
		i.Perdidas += r.VendasPerdidas
		i.Alcance += r.Alcance
		marketing += r.MarketingSemanal
	}
	if i.Alcance > 0 {
		i.ConversaoAcumuladaPct = Round2(float64(i.Novos) / float64(i.Alcance) * 100)
	}
	if i.Novos > 0 {
		i.CACAprox = Round2(marketing / float64(i.Novos))
	}
	if i.Vendas > 0 {
		i.TicketMedio = Round2(i.Receita / float64(i.Vendas))
	}
	i.Receita = Round2(i.Receita)
	i.Resultado = Round2(i.Resultado)
	return &i
}

func proximity(obs, exp float64) float64 {
	if exp <= 0 {
		return .5
	}
	err := math.Abs(obs-exp) / math.Max(math.Abs(exp), 1e-9)
	return Clamp(1-err, 0, 1)
}

// CalculateScore computes the canonical 0-100 JED score and its five
// components from the current company state and history.
func CalculateScore(e *Empresa) Score {
	i := Indicators(e)
	if i == nil {
		return Score{Observacoes: []string{"A empresa ainda não concluiu nenhuma semana."}}
	}
	weeks := len(e.Historico)
	cashNeg := 0
	for _, r := range e.Historico {
		if r.Caixa < 0 {
			cashNeg++
		}
	}
	liquidity := e.Caixa + i.AReceber - i.APagar
	fin := 0.0
	if i.Resultado > 0 {
		fin += 10
	} else {
		loss := math.Abs(i.Resultado) / math.Max(i.Receita, 1)
		fin += math.Max(0, 10*(1-loss*2))
	}
	if liquidity >= 0 {
		fin += 10
	} else {
		fin += math.Max(0, 10+liquidity/1000)
	}
	fin += 5 * (1 - float64(cashNeg)/float64(MaxInt(weeks, 1)))
	fin = Clamp(fin, 0, 25)
	baseConv := e.ConversaoBase * 100
	ratio := i.ConversaoAcumuladaPct / math.Max(baseConv, .01)
	market := math.Min(8, 8*Clamp(ratio, 0, 1.25)/1.25)
	growth := math.Min(1, float64(i.ClientesAtivos)/float64(MaxInt(10, weeks*2)))
	market += 7 * growth
	if i.Novos > 0 {
		marginRef := math.Max(1, e.Preco-e.CustoUnitario)
		cs := math.Max(0, 1-i.CACAprox/math.Max(marginRef, 1))
		market += 5 * cs
	}
	market = Clamp(market, 0, 20)
	dem := 0
	lost := 0
	for _, r := range e.Historico {
		dem += r.Demanda
		lost += r.VendasPerdidas
	}
	service := 1.0
	if dem > 0 {
		service = 1 - float64(lost)/float64(dem)
	}
	op := 15 * Clamp(service, 0, 1)
	avgSales := float64(i.Vendas) / float64(weeks)
	avgRev := i.Receita / float64(weeks)
	avgConv := i.ConversaoAcumuladaPct
	recVals := []float64{}
	for _, r := range e.Historico {
		if r.ClientesAtivosInicio > 0 {
			recVals = append(recVals, r.RecorrenciaObservadaPct)
		}
	}
	avgRec := 0.0
	if len(recVals) > 0 {
		for _, x := range recVals {
			avgRec += x
		}
		avgRec /= float64(len(recVals))
	}
	ps := []float64{
		proximity(avgSales, e.Hipoteses.VendasSemanais),
		proximity(avgRev, e.Hipoteses.FaturamentoSemanal),
		proximity(avgConv, e.Hipoteses.ConversaoPct),
		proximity(avgRec, e.Hipoteses.RecorrenciaPct),
	}
	hyp := 25 * (ps[0] + ps[1] + ps[2] + ps[3]) / 4
	progress := math.Min(1, float64(weeks)/float64(MaxInt(e.DuracaoSemanas, 1)))
	cycles := math.Min(1, float64(weeks/4)/float64(MaxInt(1, e.DuracaoSemanas/4)))
	rev := math.Min(1, float64(e.RevisoesCanvas)/float64(MaxInt(1, weeks/8)))
	mgmt := Clamp(7*progress+5*cycles+3*rev, 0, 15)
	obs := []string{}
	if liquidity < 0 {
		obs = append(obs, "Liquidez final negativa: contas a pagar superam caixa e recebíveis.")
	}
	if service < .80 {
		obs = append(obs, "Muitas vendas foram perdidas por capacidade ou estoque.")
	}
	if i.ConversaoAcumuladaPct < baseConv*.8 {
		obs = append(obs, "Conversão abaixo do parâmetro-base do negócio.")
	}
	if cashNeg > 0 {
		obs = append(obs, fmt.Sprintf("Caixa ficou negativo em %d semana(s).", cashNeg))
	}
	if len(obs) == 0 {
		obs = append(obs, "Não foram detectados alertas graves nos indicadores agregados.")
	}
	return Score{
		Round2(fin + market + op + hyp + mgmt),
		Round2(fin),
		Round2(market),
		Round2(op),
		Round2(hyp),
		Round2(mgmt),
		obs,
	}
}

func hypothesisStatus(obs, exp float64) string {
	if exp <= 0 {
		return "sem meta"
	}
	r := obs / exp
	if r >= .85 && r <= 1.15 {
		return "VALIDADA"
	}
	if r > 1.15 {
		return "SUPERADA"
	}
	return "NÃO VALIDADA"
}

// ReviewHypotheses returns the four-week hypothesis review on review weeks. It
// is nil on all other weeks, preserving the RC1.8 behavior.
func ReviewHypotheses(e *Empresa) *Review {
	if e.Semana < 4 || e.Semana%4 != 0 {
		return nil
	}
	h := e.Historico[len(e.Historico)-4:]
	sales, rev, conv := 0.0, 0.0, 0.0
	rec := []float64{}
	pos := 0
	for _, r := range h {
		sales += float64(r.Vendas)
		rev += r.Receita
		conv += r.ConversaoObservadaPct
		if r.ClientesAtivosInicio > 0 {
			rec = append(rec, r.RecorrenciaObservadaPct)
		}
		if r.Resultado > 0 {
			pos++
		}
	}
	sales /= 4
	rev /= 4
	conv /= 4
	avgRec := 0.0
	if len(rec) > 0 {
		for _, x := range rec {
			avgRec += x
		}
		avgRec /= float64(len(rec))
	}
	return &Review{
		Round2(sales), e.Hipoteses.VendasSemanais,
		Round2(rev), e.Hipoteses.FaturamentoSemanal,
		Round2(conv), e.Hipoteses.ConversaoPct,
		Round2(avgRec), e.Hipoteses.RecorrenciaPct,
		hypothesisStatus(sales, e.Hipoteses.VendasSemanais),
		hypothesisStatus(rev, e.Hipoteses.FaturamentoSemanal),
		hypothesisStatus(conv, e.Hipoteses.ConversaoPct),
		hypothesisStatus(avgRec, e.Hipoteses.RecorrenciaPct),
		pos, e.Hipoteses.EsperaResultadoPositivo, pos >= 3,
	}
}
