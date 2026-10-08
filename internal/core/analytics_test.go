package core

import (
	"math/rand"
	"reflect"
	"testing"
)

func TestAnalyticsAreStableForProcessedHistory(t *testing.T) {
	e := weekTestFixture()
	r := rand.New(rand.NewSource(20261007))
	for n := 0; n < 4; n++ {
		ProcessWeek(&e, r)
	}

	ind := Indicators(&e)
	if ind == nil || ind.Semanas != 4 {
		t.Fatalf("expected four-week indicators, got %#v", ind)
	}
	if ind.Vendas <= 0 || ind.Receita <= 0 {
		t.Fatalf("expected non-empty aggregate indicators, got %#v", ind)
	}

	s := CalculateScore(&e)
	if s.Total < 0 || s.Total > 100 {
		t.Fatalf("score outside 0-100 range: %#v", s)
	}
	if len(s.Observacoes) == 0 {
		t.Fatal("score must include at least one observation")
	}

	review := ReviewHypotheses(&e)
	if review == nil {
		t.Fatal("week 4 must produce a hypothesis review")
	}
}

func TestSimulatorFacadeReplaysSameSeed(t *testing.T) {
	a := weekTestFixture()
	b := cloneWeekTestCompany(t, a)
	sa := NewSimulator(99)
	sb := NewSimulator(99)

	for n := 0; n < 4; n++ {
		ra := sa.ProcessWeek(&a)
		rb := sb.ProcessWeek(&b)
		if !reflect.DeepEqual(ra, rb) {
			t.Fatalf("facade diverged with same seed on week %d", n+1)
		}
	}
	if !reflect.DeepEqual(sa.Indicators(&a), sb.Indicators(&b)) {
		t.Fatal("facade indicators diverged with same seed")
	}
	if !reflect.DeepEqual(sa.Score(&a), sb.Score(&b)) {
		t.Fatal("facade score diverged with same seed")
	}
	if !reflect.DeepEqual(sa.Review(&a), sb.Review(&b)) {
		t.Fatal("facade review diverged with same seed")
	}
}
