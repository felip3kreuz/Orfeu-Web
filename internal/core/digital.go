package core

import "math"

// digitalChannels and digitalTools are simulation parameters used by both the
// native UI and the platform-independent weekly engine. They were moved into
// Core in W1.4 so all frontends consume the same canonical rules.
var digitalChannels = []CanalDigitalSpec{
	{"instagram", "Instagram", "vitrine visual, conteúdo e relacionamento", 1.12, 1.03, 1.02},
	{"whatsapp", "WhatsApp Business", "atendimento, relacionamento e venda direta", 0.90, 1.18, 1.10},
	{"facebook", "Facebook", "presença social, comunidade e divulgação", 1.05, 1.02, 1.03},
	{"tiktok", "TikTok", "vídeos curtos, descoberta e alcance", 1.20, 0.96, 1.00},
	{"youtube", "YouTube", "conteúdo audiovisual, confiança e demonstração", 1.08, 1.04, 1.04},
	{"mercado_livre", "Mercado Livre", "marketplace com tráfego e venda estruturada", 1.15, 1.12, 1.02},
	{"ifood", "iFood", "plataforma de pedidos para alimentação e delivery", 1.10, 1.12, 1.06},
}

// Ferramentas estudadas nos kits do JED. Os coeficientes abaixo são parâmetros
// pedagógicos do simulador; não são estatísticas oficiais do programa.
var digitalTools = []FerramentaDigitalSpec{
	{"social_media", "Social Media", "planejamento de conteúdo, calendário e leitura de métricas", 35, 1.04, 1.22, 1.01, 1.03},
	{"copywriting", "Copywriting", "clareza, benefícios, storytelling, CTA e tratamento de objeções", 10, 1.00, 1.02, 1.08, 1.00},
	{"design", "Técnicas de Design", "identidade visual, hierarquia, composição e engajamento visual", 20, 1.01, 1.15, 1.02, 1.00},
	{"trafego_pago", "Tráfego Pago", "alcance segmentado com investimento, teste, medição e otimização", 0, 1.00, 1.00, 1.02, 1.00},
	{"uiux", "UI/UX", "interface clara, usabilidade, acessibilidade e jornada positiva", 20, 1.00, 1.03, 1.05, 1.04},
	{"ia", "Ferramentas de IA", "apoio à criação, organização, atendimento e produtividade", 12, 1.02, 1.04, 1.02, 1.01},
}

// DigitalChannelSpecs returns a copy of the canonical channel catalog.
func DigitalChannelSpecs() []CanalDigitalSpec {
	return append([]CanalDigitalSpec(nil), digitalChannels...)
}

// DigitalToolSpecs returns a copy of the canonical digital-tool catalog.
func DigitalToolSpecs() []FerramentaDigitalSpec {
	return append([]FerramentaDigitalSpec(nil), digitalTools...)
}

func DigitalToolSpec(id string) (FerramentaDigitalSpec, bool) {
	for _, f := range digitalTools {
		if f.ID == id {
			return f, true
		}
	}
	return FerramentaDigitalSpec{}, false
}

func HasDigitalTool(e *Empresa, id string) bool {
	return Contains(e.FerramentasDigitais, id)
}

func DigitalToolCost(e *Empresa) float64 {
	total := 0.0
	for _, id := range e.FerramentasDigitais {
		if f, ok := DigitalToolSpec(id); ok {
			total += f.CustoSemanal
		}
	}
	// IA é tratada como apoio de produtividade, reduzindo parte do custo de produção
	// das demais ferramentas; isso é uma regra da simulação, não um dado oficial.
	if HasDigitalTool(e, "ia") && total > 12 {
		total = 12 + (total-12)*0.88
	}
	return Round2(total)
}

func DigitalToolMods(e *Empresa) (reach, engagement, conv, rec float64) {
	reach, engagement, conv, rec = 1, 1, 1, 1
	for _, id := range e.FerramentasDigitais {
		f, ok := DigitalToolSpec(id)
		if !ok {
			continue
		}
		reach += (f.Alcance - 1) * 0.70
		engagement += (f.Engajamento - 1) * 0.75
		conv += (f.Conversao - 1) * 0.70
		rec += (f.Recorrencia - 1) * 0.70
	}
	// Tráfego pago só ganha força quando há verba de marketing.
	if HasDigitalTool(e, "trafego_pago") && e.MarketingSemanal > 0 {
		reach *= 1 + math.Min(.22, math.Sqrt(e.MarketingSemanal/900)*.14)
	}
	return Clamp(reach, .90, 1.45), Clamp(engagement, .90, 1.55), Clamp(conv, .90, 1.35), Clamp(rec, .90, 1.30)
}

func DigitalChannelMods(e *Empresa) (reach, conv, rec float64) {
	reach, conv, rec = 1, 1, 1
	if len(e.CanaisDigitais) == 0 {
		return
	}
	// Rendimentos decrescentes: combinar canais ajuda, mas não multiplica indefinidamente.
	for _, id := range e.CanaisDigitais {
		for _, c := range digitalChannels {
			if c.ID == id {
				reach += (c.Alcance - 1) * 0.65
				conv += (c.Conversao - 1) * 0.55
				rec += (c.Recorrencia - 1) * 0.55
				break
			}
		}
	}
	// Persona completa representa melhor direcionamento, não um "bônus oficial" do curso.
	pc := PersonaCompleteness(e.Persona)
	conv *= 0.97 + 0.06*pc
	return Clamp(reach, .85, 1.55), Clamp(conv, .85, 1.45), Clamp(rec, .85, 1.35)
}

func HasDigitalChannel(e *Empresa, id string) bool { return Contains(e.CanaisDigitais, id) }

func ChannelName(id string) string {
	for _, c := range digitalChannels {
		if c.ID == id {
			return c.Nome
		}
	}
	return id
}

func FixedWeekly(e *Empresa) float64 {
	aluguel := e.AluguelMensal / 4.33
	folha := float64(e.Funcionarios) * e.SalarioMedio / 4.33
	digital := 0.0
	if e.Operacao == "digital" || e.Operacao == "hibrida" {
		digital = 55
	}
	reg := 0.0
	if e.Regulamentado {
		reg = e.CustoRegulatorioMensal / 4.33
	}
	return aluguel + folha + digital + reg + e.MarketingSemanal + DigitalToolCost(e)
}
