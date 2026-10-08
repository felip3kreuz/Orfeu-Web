package core

import (
	"math/rand"
	"reflect"
	"testing"
)

type scriptedRandom struct {
	floats []float64
	ints   []int
	fi     int
	ii     int
}

func (s *scriptedRandom) Float64() float64 {
	v := s.floats[s.fi]
	s.fi++
	return v
}

func (s *scriptedRandom) Intn(n int) int {
	v := s.ints[s.ii]
	s.ii++
	if v < 0 {
		v = -v
	}
	return v % n
}

func randomEngineCompany() *Empresa {
	return &Empresa{
		Dificuldade:           "avancado",
		CenarioEventoNegExtra: .04,
		ConcorrenciaIndice:    1.07,
		UsaInsumos:            true,
		PrazoFornecedorMax:    3,
		Semana:                5,
		Caixa:                 2000,
		Insumos: []InsumoEstoque{
			{ID: "a", Nome: "A", Unidade: "kg", Quantidade: 10, CustoMedio: 8, CustoReferencia: 10, ConsumoPorVenda: 1, PerdaSemanal: .08, Critico: true},
			{ID: "b", Nome: "B", Unidade: "un", Quantidade: 20, CustoMedio: 3, CustoReferencia: 4, ConsumoPorVenda: 2, PerdaSemanal: .03, Critico: true},
		},
	}
}

func TestSelectEventWithInjectedRandomSource(t *testing.T) {
	e := randomEngineCompany()
	r := &scriptedRandom{floats: []float64{0, 0}, ints: []int{2}}
	ev := SelectEvent(e, r)
	if ev == nil {
		t.Fatal("expected an event")
	}
	if ev.Texto != "O movimento do mercado caiu inesperadamente nesta semana." || ev.Alcance != .88 || ev.Conversao != .96 {
		t.Fatalf("unexpected event: %#v", ev)
	}
}

func TestRandomOperationsRepeatWithSameSeed(t *testing.T) {
	seed := int64(99173)

	a := randomEngineCompany()
	b := randomEngineCompany()
	UpdateCompetition(a, rand.New(rand.NewSource(seed)), .06)
	UpdateCompetition(b, rand.New(rand.NewSource(seed)), .06)
	if a.ConcorrenciaIndice != b.ConcorrenciaIndice {
		t.Fatalf("competition drift is not repeatable: %v != %v", a.ConcorrenciaIndice, b.ConcorrenciaIndice)
	}

	a = randomEngineCompany()
	b = randomEngineCompany()
	av, ad := WasteInputs(a, rand.New(rand.NewSource(seed)))
	bv, bd := WasteInputs(b, rand.New(rand.NewSource(seed)))
	if av != bv || ad != bd || !reflect.DeepEqual(a.Insumos, b.Insumos) || a.EstoqueValor != b.EstoqueValor {
		t.Fatalf("waste is not repeatable")
	}

	f := FornecedorSpec{ID: "f", Nome: "Fornecedor", MultiplicadorPreco: .91, PrazoEntregaSemanas: 1, Confiabilidade: .72, PrazoPagamentoMax: 2}
	a = randomEngineCompany()
	b = randomEngineCompany()
	aok, acost, amsg := PlaceInputOrder(a, "a", 6, f, 2, rand.New(rand.NewSource(seed)))
	bok, bcost, bmsg := PlaceInputOrder(b, "a", 6, f, 2, rand.New(rand.NewSource(seed)))
	if aok != bok || acost != bcost || amsg != bmsg || !reflect.DeepEqual(a, b) {
		t.Fatalf("supplier order is not repeatable")
	}
}

func TestRandRangeUsesInjectedSource(t *testing.T) {
	r := &scriptedRandom{floats: []float64{.25}}
	if got := RandRange(r, -2, 6); got != 0 {
		t.Fatalf("RandRange = %v, want 0", got)
	}
}
