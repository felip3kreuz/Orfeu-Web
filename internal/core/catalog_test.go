package core

import "testing"

func TestCatalogHelpers(t *testing.T) {
	cat := Catalogo{Setores: []SetorCatalogo{{
		ID: "s", Nome: "Setor", Tipos: []TipoCatalogo{{
			ID: "t", Nome: "Tipo", Especialidades: []Modelo{{ID: "m", Nome: "Modelo", Estoque: true, CustoUnitario: 5, EstoqueInicialUn: 10, PerecibilidadeSemanal: .06}},
		}},
	}}}
	if CatalogCount(cat) != 1 {
		t.Fatalf("unexpected catalog count: %d", CatalogCount(cat))
	}
	m, ok := FindModel(cat, "m")
	if !ok || m.SetorID != "s" || m.TipoID != "t" {
		t.Fatalf("model context not enriched: %#v ok=%v", m, ok)
	}
	p := SupplyProfileFor(m, CatalogoInsumos{})
	if len(p.Insumos) != 1 || p.Insumos[0].ValidadeSemanas != 2 {
		t.Fatalf("unexpected derived supply profile: %#v", p)
	}
}
