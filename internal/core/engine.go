package core

import (
	"fmt"
	"math"
	"strings"
)

// Mods contains the deterministic modifiers derived from the company's Lean Canvas.
type Mods struct {
	Alcance, Conversao, Recorrencia, SensibilidadePreco, ToleranciaPreco, Marketing, Capacidade float64
}

// DifficultyProfile contains the deterministic tuning parameters for each difficulty level.
type DifficultyProfile struct {
	Oscilacao         float64
	ChanceEvento      float64
	NegExtra          float64
	FornecedorBonus   float64
	Desperdicio       float64
	ConcorrenciaDrift float64
}

func Contains(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}

func CanvasMods(e *Empresa) Mods {
	m := Mods{1, 1, 1, 1, 1, 1, 1}
	c := e.Canvas
	if Contains(c.Segmentos, "sensivel_preco") {
		m.SensibilidadePreco *= 1.25
	}
	if Contains(c.Segmentos, "qualidade") {
		m.SensibilidadePreco *= .88
		m.ToleranciaPreco *= 1.08
		m.Recorrencia *= 1.05
	}
	if Contains(c.Segmentos, "conveniencia") {
		if e.Operacao == "hibrida" {
			m.Conversao *= 1.05
		}
		if e.Delivery {
			m.Conversao *= 1.05
			m.Recorrencia *= 1.04
		}
	}
	if Contains(c.Segmentos, "digital") && (e.Operacao == "digital" || e.Operacao == "hibrida") {
		m.Alcance *= 1.07
	}
	if Contains(c.Segmentos, "b2b") {
		m.Conversao *= .92
		m.Recorrencia *= 1.10
	}
	if Contains(c.Canais, "redes_sociais") {
		m.Marketing *= 1.14
		m.Alcance *= 1.03
	}
	if Contains(c.Canais, "busca_online") {
		m.Marketing *= 1.08
		m.Alcance *= 1.04
	}
	if Contains(c.Canais, "indicacao") {
		m.Conversao *= 1.05
		m.Recorrencia *= 1.08
	}
	if Contains(c.Canais, "loja_fisica") && (e.Operacao == "fisica" || e.Operacao == "hibrida") {
		m.Alcance *= 1.04
	}
	if Contains(c.Canais, "marketplace") {
		m.Alcance *= 1.12
		m.Conversao *= 1.03
	}
	if Contains(c.Canais, "prospeccao") && Contains(c.Segmentos, "b2b") {
		m.Conversao *= 1.08
	}
	switch c.PropostaValor {
	case "menor_preco":
		m.SensibilidadePreco *= 1.12
		if e.Preco <= e.PrecoReferencia {
			m.Conversao *= 1.10
		}
	case "qualidade":
		m.ToleranciaPreco *= 1.08
		if e.Reputacao >= 60 {
			m.Conversao *= 1.06
			m.Recorrencia *= 1.06
		}
	case "rapidez":
		if e.Delivery || e.Operacao == "digital" {
			m.Conversao *= 1.07
		}
	case "conveniencia":
		if e.Operacao == "hibrida" || e.Delivery {
			m.Conversao *= 1.09
			m.Recorrencia *= 1.04
		}
	case "personalizacao":
		m.SensibilidadePreco *= .92
		m.ToleranciaPreco *= 1.05
		m.Capacidade *= .94
		m.Recorrencia *= 1.06
	case "exclusividade":
		m.SensibilidadePreco *= .82
		m.ToleranciaPreco *= 1.15
		m.Alcance *= .94
	case "atendimento":
		if e.Reputacao >= 55 {
			m.Conversao *= 1.05
			m.Recorrencia *= 1.08
		}
	}
	switch c.ReceitaModelo {
	case "assinatura":
		m.Recorrencia *= 1.30
		m.Conversao *= .94
	case "projeto":
		m.Recorrencia *= .86
	case "comissao":
		m.Conversao *= 1.04
	}
	return m
}

func CanvasExplanations(e *Empresa) []string {
	c := e.Canvas
	out := []string{}
	if Contains(c.Segmentos, "sensivel_preco") {
		out = append(out, "Seu segmento reage mais fortemente a preços acima da referência.")
	}
	if Contains(c.Segmentos, "qualidade") {
		out = append(out, "Clientes orientados à qualidade toleram melhor preços superiores e tendem a retornar mais.")
	}
	if Contains(c.Segmentos, "digital") {
		out = append(out, "O segmento digital amplia o alcance quando a operação possui canal digital.")
	}
	if Contains(c.Canais, "redes_sociais") {
		out = append(out, "Redes sociais aumentam o retorno do investimento em marketing.")
	}
	if Contains(c.Canais, "indicacao") {
		out = append(out, "Indicação melhora conversão e recorrência.")
	}
	if Contains(c.Canais, "marketplace") {
		out = append(out, "Marketplace amplia alcance, mas cobra taxa sobre parte das vendas.")
	}
	return out
}

func Clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func Round2(v float64) float64 { return math.Round(v*100) / 100 }

func LocationFactor(e *Empresa) float64 {
	if e.Operacao == "digital" {
		return 1
	}
	switch e.QualidadeLocalizacao {
	case "baixa":
		return .88
	case "alta":
		return 1.14
	}
	return 1
}

func SeasonFactor(e *Empresa, week int) float64 {
	return 1 + e.SazonalidadeAmplitude*math.Sin(2*math.Pi*float64(week-1)/12+e.SazonalidadeFase)
}

func CompetitionFactor(e *Empresa) float64 {
	return Clamp(1-(e.ConcorrenciaIndice-1)*.30, .72, 1.18)
}

func EquipmentFactor(e *Empresa) float64 {
	ref := e.InvestimentosIniciais["equipamentos_referencia"]
	g := e.InvestimentosIniciais["equipamentos"]
	if ref <= 0 {
		return 1
	}
	r := g / ref
	if r < 1 {
		return Clamp(.68+.32*r, .68, 1)
	}
	return Clamp(1+(r-1)*.12, 1, 1.18)
}

func WeeklyInterest(e *Empresa) float64 {
	if e.Divida <= 0 || e.JurosMensal <= 0 {
		return 0
	}
	return e.Divida * (e.JurosMensal / 100) / 4.33
}

func PriceFactor(e *Empresa, m Mods, p float64) float64 {
	ref := math.Max(.01, e.PrecoReferencia*m.ToleranciaPreco)
	r := p / ref
	if r > 1 {
		return Clamp(1-(r-1)*.90*m.SensibilidadePreco, .38, 1)
	}
	return Clamp(1+(1-r)*.38*m.SensibilidadePreco, 1, 1.30)
}

func MarketingFactor(e *Empresa, m Mods) float64 {
	inc := math.Min(1, math.Sqrt(math.Max(0, e.MarketingSemanal)/700)*.55)
	return 1 + inc*m.Marketing
}

func LaunchFactor(e *Empresa) float64 {
	if e.Semana != 1 {
		return 1
	}
	return 1 + math.Min(.55, e.InvestimentosIniciais["marketing_lancamento"]/3000)
}

func RepReach(e *Empresa) float64 {
	imp := Clamp(e.ReputacaoImportancia, .5, 1.6)
	d := (e.Reputacao - 50) / 50
	return Clamp(1+d*.20*imp, .65, 1.35)
}

func RepConv(e *Empresa) float64 {
	imp := Clamp(e.ReputacaoImportancia, .5, 1.6)
	d := (e.Reputacao - 50) / 50
	return Clamp(1+d*.24*imp, .60, 1.40)
}

func RepRec(e *Empresa) float64 {
	imp := Clamp(e.ReputacaoImportancia, .5, 1.6)
	d := (e.Reputacao - 50) / 50
	return Clamp(1+d*.28*imp, .58, 1.45)
}

func NormalizeDifficulty(d string) string {
	switch strings.ToLower(strings.TrimSpace(d)) {
	case "iniciante", "intermediario", "avancado":
		return strings.ToLower(strings.TrimSpace(d))
	default:
		return "intermediario"
	}
}

func DifficultyProfileFor(e *Empresa) DifficultyProfile {
	d := NormalizeDifficulty(e.Dificuldade)
	if d == "iniciante" {
		return DifficultyProfile{.70, .28, -.08, .08, .80, .70}
	}
	if d == "avancado" {
		return DifficultyProfile{1.30, .42, .08, -.06, 1.18, 1.30}
	}
	return DifficultyProfile{1.00, .34, 0, 0, 1.00, 1.00}
}

func MaxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func MinInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func Capacity(e *Empresa, m Mods) int {
	c := float64(e.CapacidadeBase + e.Funcionarios*MaxInt(3, int(float64(e.CapacidadeBase)*.18)))
	c *= EquipmentFactor(e) * m.Capacidade
	return MaxInt(1, int(math.Round(c)))
}

func PaymentWeights(e *Empresa) map[string]float64 {
	base := map[string]float64{"dinheiro": .18, "pix": .30, "cartao": .52}
	if e.Operacao == "digital" {
		base = map[string]float64{"dinheiro": 0, "pix": .32, "cartao": .68}
	} else if e.Operacao == "hibrida" {
		base = map[string]float64{"dinheiro": .10, "pix": .30, "cartao": .60}
	}
	out := map[string]float64{}
	sum := 0.0
	for _, m := range e.MeiosPagamento {
		out[m] = base[m]
		sum += base[m]
	}
	if sum <= 0 {
		return map[string]float64{"pix": 1}
	}
	for k, v := range out {
		out[k] = v / sum
	}
	return out
}

func Liquidate(list []Conta, week int) (float64, []Conta) {
	v := 0.0
	f := []Conta{}
	for _, x := range list {
		if x.Semana <= week {
			v += x.Valor
		} else {
			f = append(f, x)
		}
	}
	return v, f
}

func SumAccounts(xs []Conta) float64 {
	s := 0.0
	for _, x := range xs {
		s += x.Valor
	}
	return s
}

func StockValueInputs(e *Empresa) float64 {
	t := 0.0
	for _, x := range e.Insumos {
		t += x.Quantidade * x.CustoMedio
	}
	return t
}

func FindInputIndex(e *Empresa, id string) int {
	for i := range e.Insumos {
		if e.Insumos[i].ID == id {
			return i
		}
	}
	return -1
}

func ReceiveInputOrders(e *Empresa) string {
	if len(e.PedidosInsumos) == 0 {
		return ""
	}
	left := make([]PedidoInsumo, 0, len(e.PedidosInsumos))
	notes := []string{}
	for _, o := range e.PedidosInsumos {
		if o.SemanaEntrega > e.Semana {
			left = append(left, o)
			continue
		}
		i := FindInputIndex(e, o.InsumoID)
		if i < 0 {
			left = append(left, o)
			continue
		}
		oldQ := e.Insumos[i].Quantidade
		oldV := oldQ * e.Insumos[i].CustoMedio
		e.Insumos[i].Quantidade += o.Quantidade
		if e.Insumos[i].Quantidade > 0 {
			e.Insumos[i].CustoMedio = (oldV + o.CustoTotal) / e.Insumos[i].Quantidade
		}
		notes = append(notes, fmt.Sprintf("%s: %.1f %s", o.InsumoNome, o.Quantidade, e.Insumos[i].Unidade))
	}
	e.PedidosInsumos = left
	e.EstoqueValor = StockValueInputs(e)
	return strings.Join(notes, "; ")
}

func MaxSalesByInputs(e *Empresa) (int, string) {
	if !e.UsaInsumos || len(e.Insumos) == 0 {
		return math.MaxInt32, ""
	}
	mx := math.MaxInt32
	limiting := ""
	for _, x := range e.Insumos {
		if !x.Critico || x.ConsumoPorVenda <= 0 {
			continue
		}
		n := int(math.Floor(x.Quantidade/x.ConsumoPorVenda + 1e-9))
		if n < mx {
			mx = n
			limiting = x.Nome
		}
	}
	if mx == math.MaxInt32 {
		return math.MaxInt32, ""
	}
	return MaxInt(0, mx), limiting
}

func ConsumeInputs(e *Empresa, sales int) float64 {
	if !e.UsaInsumos {
		return 0
	}
	cmv := 0.0
	for i := range e.Insumos {
		q := float64(sales) * e.Insumos[i].ConsumoPorVenda
		if q > e.Insumos[i].Quantidade {
			q = e.Insumos[i].Quantidade
		}
		cmv += q * e.Insumos[i].CustoMedio
		e.Insumos[i].Quantidade = math.Max(0, e.Insumos[i].Quantidade-q)
	}
	e.EstoqueValor = StockValueInputs(e)
	return cmv
}

func BuyStock(e *Empresa, q, term int) (bool, float64, string) {
	if !e.UsaEstoque || q <= 0 {
		return false, 0, "quantidade inválida"
	}
	if term < 0 {
		term = 0
	}
	if term > e.PrazoFornecedorMax {
		term = e.PrazoFornecedorMax
	}
	cost := float64(q) * e.CustoUnitario
	if term == 0 && cost > e.Caixa {
		return false, cost, "caixa insuficiente"
	}
	e.EstoqueUnidades += q
	e.EstoqueValor += cost
	if e.EstoqueUnidades > 0 {
		e.CustoMedioEstoque = e.EstoqueValor / float64(e.EstoqueUnidades)
	}
	if term == 0 {
		e.Caixa -= cost
		return true, cost, "compra à vista"
	}
	e.ContasPagar = append(e.ContasPagar, Conta{Semana: e.Semana + term, Valor: Round2(cost), Descricao: fmt.Sprintf("Fornecedor — %d unidades", q)})
	return true, cost, fmt.Sprintf("compra a prazo (%d semana(s))", term)
}

func AllocateSales(sales, newWant, recWant int) (int, int) {
	total := newWant + recWant
	if total <= 0 || sales <= 0 {
		return 0, 0
	}
	n := int(math.Round(float64(sales) * float64(newWant) / float64(total)))
	n = MinInt(n, MinInt(newWant, sales))
	r := MinInt(recWant, sales-n)
	left := sales - n - r
	if left > 0 {
		a := MinInt(left, newWant-n)
		n += a
		left -= a
	}
	if left > 0 {
		r += MinInt(left, recWant-r)
	}
	return n, r
}

func PersonaCompleteness(p Persona) float64 {
	vals := []string{p.Nome, p.Demografia, p.Rotinas, p.Objetivos, p.Desafios, p.Motivadores, p.Objecoes, p.Citacoes, p.PalavrasChave}
	filled := 0
	for _, v := range vals {
		v = strings.TrimSpace(strings.ToLower(v))
		if v != "" && v != "a definir" && v != "não definido" {
			filled++
		}
	}
	return float64(filled) / float64(len(vals))
}

func JourneyStep(e *Empresa) int {
	if e == nil {
		return 1
	}
	if PersonaCompleteness(e.Persona) < .67 || strings.TrimSpace(e.Canvas.Problema) == "" || strings.TrimSpace(e.Canvas.PropostaValor) == "" {
		return 1
	}
	if len(e.CanaisDigitais) == 0 {
		return 2
	}
	if e.Semana < 4 {
		return 3
	}
	return 4
}
