package core

// SupplyProfileFor returns the supply profile configured for a business model.
// If the supply catalog has no explicit profile but the model uses stock, a
// conservative single-input profile is derived from the model itself. Keeping
// this rule in Core lets native and web frontends initialize inventory in the
// same way without depending on filesystem or UI code.
func SupplyProfileFor(m Modelo, c CatalogoInsumos) PerfilInsumos {
	for _, p := range c.Perfis {
		if p.ModeloID == m.ID {
			return p
		}
	}
	if m.Estoque {
		loss := m.PerecibilidadeSemanal
		valid := 24
		if loss >= .10 {
			valid = 1
		} else if loss >= .05 {
			valid = 2
		} else if loss > 0 {
			valid = 6
		}
		return PerfilInsumos{ModeloID: m.ID, Insumos: []InsumoSpec{{
			ID:              "mercadoria",
			Nome:            "Mercadoria / insumo principal",
			Unidade:         "un",
			CustoBase:       m.CustoUnitario,
			ConsumoPorVenda: 1,
			EstoqueInicial:  float64(m.EstoqueInicialUn),
			ValidadeSemanas: valid,
			PerdaSemanal:    loss,
			Critico:         true,
		}}}
	}
	return PerfilInsumos{}
}

// FindSupplier finds a supplier by canonical ID.
func FindSupplier(c CatalogoInsumos, id string) (FornecedorSpec, bool) {
	for _, f := range c.Fornecedores {
		if f.ID == id {
			return f, true
		}
	}
	return FornecedorSpec{}, false
}

// CatalogCount returns the number of business specialties in the catalog.
func CatalogCount(c Catalogo) int {
	n := 0
	for _, s := range c.Setores {
		for _, t := range s.Tipos {
			n += len(t.Especialidades)
		}
	}
	return n
}

// FindModel finds a business model and enriches it with its containing sector
// and business-type metadata. These four context fields are intentionally not
// serialized in the source catalog and are derived on lookup.
func FindModel(c Catalogo, id string) (Modelo, bool) {
	for _, s := range c.Setores {
		for _, t := range s.Tipos {
			for _, m := range t.Especialidades {
				if m.ID == id {
					m.SetorID = s.ID
					m.SetorNome = s.Nome
					m.TipoID = t.ID
					m.TipoNome = t.Nome
					return m, true
				}
			}
		}
	}
	return Modelo{}, false
}
