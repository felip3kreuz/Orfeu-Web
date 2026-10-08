package core

import (
	"fmt"
	"math"
)

// ProcessWeek advances one simulation week and mutates e with the resulting
// financial, operational and customer state. All randomness is supplied by r,
// which makes the same week reproducible across native and WebAssembly builds.
func ProcessWeek(e *Empresa, r RandomSource) Registro {
	e.Semana++
	deliveries := ReceiveInputOrders(e)
	recv, cr := Liquidate(e.ContasReceber, e.Semana)
	pay, cp := Liquidate(e.ContasPagar, e.Semana)
	e.ContasReceber = cr
	e.ContasPagar = cp
	e.Caixa += recv - pay
	ev := SelectEvent(e, r)
	evReach, evConv, delta := 1.0, 1.0, 0.0
	evText := ""
	if ev != nil {
		evReach = ev.Alcance
		evConv = ev.Conversao
		delta = ev.ConcorrenciaDelta
		e.Reputacao = Clamp(e.Reputacao+ev.Reputacao, 0, 100)
		evText = ev.Texto
	}
	UpdateCompetition(e, r, delta)
	m := CanvasMods(e)
	digitalReach, digitalConv, digitalRec := DigitalChannelMods(e)
	toolReach, toolEngagement, toolConv, toolRec := DigitalToolMods(e)
	disc := Clamp(e.PromocaoDesconto, 0, 35)
	effective := e.Preco * (1 - disc/100)
	promo := 1 + math.Min(.35, disc/100*1.20)
	dp := DifficultyProfileFor(e)
	oscRange := e.CenarioOscilacao * dp.Oscilacao
	osc := RandRange(r, 1-oscRange, 1+oscRange)
	reachF := float64(e.AlcanceBase) * e.CenarioAlcance * LocationFactor(e) * SeasonFactor(e, e.Semana) * MarketingFactor(e, m) * LaunchFactor(e) * RepReach(e) * m.Alcance * digitalReach * toolReach * evReach * osc
	reach := MaxInt(0, int(math.Round(reachF)))
	deliveryF := 1.0
	if e.Delivery {
		deliveryF = e.DeliveryAfinidade
	}
	conv := e.ConversaoBase * e.CenarioConversao * PriceFactor(e, m, effective) * RepConv(e) * CompetitionFactor(e) * m.Conversao * digitalConv * toolConv * evConv * promo * deliveryF
	conv = Clamp(conv, .005, .65)
	engagementRate := Clamp((.035+.015*PersonaCompleteness(e.Persona)+.004*float64(len(e.CanaisDigitais)))*toolEngagement, .02, .28)
	interactions := MaxInt(0, int(math.Round(float64(reach)*engagementRate)))
	leads := MaxInt(0, int(math.Round(float64(interactions)*Clamp(.18+conv*.8, .12, .55))))
	newWant := MaxInt(0, int(math.Round(float64(reach)*conv)))
	activeStart := e.ClientesAtivos
	recRate := e.RecorrenciaBase * RepRec(e) * m.Recorrencia * digitalRec * toolRec * PriceFactor(e, m, effective)
	recRate = Clamp(recRate, 0, .70)
	recWant := MaxInt(0, int(math.Round(float64(activeStart)*recRate)))
	demand := newWant + recWant
	cap := Capacity(e, m)
	sales := MinInt(demand, cap)
	gargalo := ""
	if demand > cap {
		gargalo = "capacidade"
	}
	inputLimiting := ""
	if e.UsaInsumos {
		mx, lim := MaxSalesByInputs(e)
		if mx < sales {
			sales = mx
			gargalo = "insumos"
			inputLimiting = lim
		}
	} else if e.UsaEstoque && e.EstoqueUnidades < sales {
		sales = e.EstoqueUnidades
		gargalo = "estoque"
	}
	sales = MaxInt(0, sales)
	newGot, recGot := AllocateSales(sales, newWant, recWant)
	lost := MaxInt(0, demand-sales)
	revenue := float64(sales) * effective
	weights := PaymentWeights(e)
	card := revenue * weights["cartao"]
	immediate := revenue - card
	cardFee := card * (e.TaxaCartao / 100)
	cardNet := math.Max(0, card-cardFee)
	cardNow := 0.0
	if cardNet > 0 {
		if e.PrazoCartaoSemanas <= 0 {
			cardNow = cardNet
		} else {
			e.ContasReceber = append(e.ContasReceber, Conta{e.Semana + e.PrazoCartaoSemanas, Round2(cardNet), fmt.Sprintf("Cartão — vendas da semana %d", e.Semana)})
		}
	}
	deliveryFee := 0.0
	if e.Delivery && (e.DeliveryTipo == "plataforma" || e.DeliveryTipo == "misto") {
		share := .55
		if e.DeliveryTipo == "misto" {
			share = .30
		}
		deliveryFee = revenue * share * .18
	}
	marketFee := 0.0
	if Contains(e.Canvas.Canais, "marketplace") || HasDigitalChannel(e, "mercado_livre") {
		marketFee += revenue * .20 * .12
	}
	if HasDigitalChannel(e, "ifood") && e.Delivery {
		marketFee += revenue * .18 * .12
	}
	fixed := FixedWeekly(e)
	interest := WeeklyInterest(e)
	cmv := 0.0
	varCash := 0.0
	if e.UsaInsumos {
		cmv = ConsumeInputs(e, sales)
	} else if e.UsaEstoque {
		cmv = float64(sales) * e.CustoMedioEstoque
		e.EstoqueUnidades -= sales
		e.EstoqueValor = math.Max(0, e.EstoqueValor-cmv)
	} else {
		cmv = float64(sales) * e.CustoUnitario
		varCash = cmv
	}
	wasteUnits := 0
	wasteValue := 0.0
	wasteDetails := ""
	if e.UsaInsumos {
		wasteValue, wasteDetails = WasteInputs(e, r)
	} else if e.UsaEstoque && e.PerecibilidadeSemanal > 0 && e.EstoqueUnidades > 0 {
		loss := Clamp(e.PerecibilidadeSemanal*DifficultyProfileFor(e).Desperdicio*RandRange(r, .75, 1.25), 0, .50)
		wasteUnits = MinInt(e.EstoqueUnidades, int(math.Round(float64(e.EstoqueUnidades)*loss)))
		wasteValue = float64(wasteUnits) * e.CustoMedioEstoque
		e.EstoqueUnidades -= wasteUnits
		e.EstoqueValor = math.Max(0, e.EstoqueValor-wasteValue)
	}
	result := revenue - cmv - wasteValue - cardFee - deliveryFee - marketFee - fixed - interest
	entries := recv + immediate + cardNow
	exits := pay + deliveryFee + marketFee + fixed + interest + varCash
	cashFlow := entries - exits
	e.Caixa += immediate + cardNow - deliveryFee - marketFee - fixed - interest - varCash
	retention := Clamp(.92+(e.Reputacao-50)/1000, .86, .97)
	e.ClientesAtivos = MaxInt(0, int(math.Round(float64(activeStart)*retention))+newGot)
	e.ClientesTotais += newGot
	e.ClientesRecorrentesTotal += recGot
	service := 1.0
	if demand > 0 {
		service = float64(sales) / float64(demand)
	}
	if service >= .92 && sales > 0 {
		e.Reputacao = Clamp(e.Reputacao+1, 0, 100)
	} else if service < .65 && demand > 0 {
		e.Reputacao = Clamp(e.Reputacao-2, 0, 100)
	}
	obsConv := 0.0
	if reach > 0 {
		obsConv = float64(newGot) / float64(reach)
	}
	obsRec := 0.0
	if activeStart > 0 {
		obsRec = float64(recGot) / float64(activeStart)
	}
	var cac *float64
	if newGot > 0 {
		x := Round2(e.MarketingSemanal / float64(newGot))
		cac = &x
	}
	ticket := 0.0
	if sales > 0 {
		ticket = revenue / float64(sales)
	}
	varUnit := 0.0
	if sales > 0 {
		varUnit = (cmv + cardFee + deliveryFee + marketFee) / float64(sales)
	}
	margin := math.Max(0, effective-varUnit)
	var breakeven *int
	if margin > 0 {
		x := int(math.Ceil((fixed + interest) / margin))
		breakeven = &x
	}
	reg := Registro{Semana: e.Semana, Alcance: reach, Interacoes: interactions, Leads: leads, ConversaoTeoricaPct: Round2(conv * 100), ConversaoObservadaPct: Round2(obsConv * 100), ClientesAtivosInicio: activeStart, NovosDesejados: newWant, RecorrentesDesejados: recWant, NovosClientes: newGot, ClientesRecorrentes: recGot, ClientesAtivosFinal: e.ClientesAtivos, RecorrenciaObservadaPct: Round2(obsRec * 100), Demanda: demand, Capacidade: cap, Vendas: sales, VendasPerdidas: lost, Gargalo: gargalo, PrecoEfetivo: Round2(effective), DescontoPct: Round2(disc), Receita: Round2(revenue), TicketMedio: Round2(ticket), CAC: cac, CMV: Round2(cmv), TaxaPagamento: Round2(cardFee), ComissaoDelivery: Round2(deliveryFee), TaxaMarketplace: Round2(marketFee), CustosFixos: Round2(fixed), Juros: Round2(interest), Resultado: Round2(result), RecebimentosAnteriores: Round2(recv), PagamentosAnteriores: Round2(pay), ReceitaImediata: Round2(immediate + cardNow), FluxoCaixa: Round2(cashFlow), Caixa: Round2(e.Caixa), ContasReceber: Round2(SumAccounts(e.ContasReceber)), ContasPagar: Round2(SumAccounts(e.ContasPagar)), Reputacao: Round2(e.Reputacao), EstoqueUnidades: e.EstoqueUnidades, EstoqueValor: Round2(e.EstoqueValor), DesperdicioUnidades: wasteUnits, DesperdicioValor: Round2(wasteValue), ConcorrenciaIndice: Round2(e.ConcorrenciaIndice), SazonalidadeFator: math.Round(SeasonFactor(e, e.Semana)*1000) / 1000, MargemContribuicaoUnit: Round2(margin), PontoEquilibrioVendas: breakeven, MarketingSemanal: Round2(e.MarketingSemanal), InsumoLimitante: inputLimiting, EntregasRecebidas: deliveries, DesperdicioInsumos: Round2(wasteValue), DesperdicioDetalhes: wasteDetails, Evento: evText}
	e.Historico = append(e.Historico, reg)
	e.PromocaoDesconto = 0
	return reg
}
