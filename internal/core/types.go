// Package core contains the platform-independent domain model of JED Simulador.
//
// Domain types were extracted in W1.1. W1.2 begins moving deterministic
// simulation rules into this package while package main keeps compatibility
// wrappers for the existing Win32, Classic UI and server code.
package core

type LeanCanvas struct {
	Problema      string   `json:"problema"`
	Segmentos     []string `json:"segmentos"`
	PropostaValor string   `json:"proposta_valor"`
	Solucao       string   `json:"solucao"`
	Canais        []string `json:"canais"`
	ReceitaModelo string   `json:"receita_modelo"`
	CustosNotas   string   `json:"custos_notas"`
	Metricas      []string `json:"metricas"`
	Vantagem      string   `json:"vantagem"`
}
type Persona struct {
	Nome          string `json:"nome"`
	Demografia    string `json:"demografia"`
	Rotinas       string `json:"rotinas"`
	Objetivos     string `json:"objetivos"`
	Desafios      string `json:"desafios"`
	Motivadores   string `json:"motivadores"`
	Objecoes      string `json:"objecoes"`
	Citacoes      string `json:"citacoes"`
	PalavrasChave string `json:"palavras_chave"`
}
type CanalDigitalSpec struct {
	ID          string
	Nome        string
	Descricao   string
	Alcance     float64
	Conversao   float64
	Recorrencia float64
}
type FerramentaDigitalSpec struct {
	ID           string
	Nome         string
	Descricao    string
	CustoSemanal float64
	Alcance      float64
	Engajamento  float64
	Conversao    float64
	Recorrencia  float64
}
type Hipoteses struct {
	VendasSemanais          float64 `json:"vendas_semanais"`
	FaturamentoSemanal      float64 `json:"faturamento_semanal"`
	EsperaResultadoPositivo bool    `json:"espera_resultado_positivo"`
	ConversaoPct            float64 `json:"conversao_pct"`
	RecorrenciaPct          float64 `json:"recorrencia_pct"`
}
type Conta struct {
	Semana    int     `json:"semana"`
	Valor     float64 `json:"valor"`
	Descricao string  `json:"descricao"`
}
type InsumoSpec struct {
	ID              string  `json:"id"`
	Nome            string  `json:"nome"`
	Unidade         string  `json:"unidade"`
	CustoBase       float64 `json:"custo_base"`
	ConsumoPorVenda float64 `json:"consumo_por_venda"`
	EstoqueInicial  float64 `json:"estoque_inicial"`
	ValidadeSemanas int     `json:"validade_semanas"`
	PerdaSemanal    float64 `json:"perda_semanal"`
	Critico         bool    `json:"critico"`
}
type PerfilInsumos struct {
	ModeloID string       `json:"modelo_id"`
	Insumos  []InsumoSpec `json:"insumos"`
}
type FornecedorSpec struct {
	ID                  string  `json:"id"`
	Nome                string  `json:"nome"`
	MultiplicadorPreco  float64 `json:"multiplicador_preco"`
	PrazoEntregaSemanas int     `json:"prazo_entrega_semanas"`
	Confiabilidade      float64 `json:"confiabilidade"`
	PrazoPagamentoMax   int     `json:"prazo_pagamento_max"`
	Descricao           string  `json:"descricao"`
}
type CatalogoInsumos struct {
	Versao       string           `json:"versao"`
	Descricao    string           `json:"descricao"`
	Fornecedores []FornecedorSpec `json:"fornecedores"`
	Perfis       []PerfilInsumos  `json:"perfis"`
}
type InsumoEstoque struct {
	ID              string  `json:"id"`
	Nome            string  `json:"nome"`
	Unidade         string  `json:"unidade"`
	Quantidade      float64 `json:"quantidade"`
	CustoMedio      float64 `json:"custo_medio"`
	CustoReferencia float64 `json:"custo_referencia"`
	ConsumoPorVenda float64 `json:"consumo_por_venda"`
	ValidadeSemanas int     `json:"validade_semanas"`
	PerdaSemanal    float64 `json:"perda_semanal"`
	Critico         bool    `json:"critico"`
}
type PedidoInsumo struct {
	InsumoID      string  `json:"insumo_id"`
	InsumoNome    string  `json:"insumo_nome"`
	Quantidade    float64 `json:"quantidade"`
	CustoUnitario float64 `json:"custo_unitario"`
	CustoTotal    float64 `json:"custo_total"`
	Fornecedor    string  `json:"fornecedor"`
	SemanaEntrega int     `json:"semana_entrega"`
}
type Registro struct {
	Semana                  int      `json:"semana"`
	Alcance                 int      `json:"alcance"`
	Interacoes              int      `json:"interacoes"`
	Leads                   int      `json:"leads"`
	ConversaoTeoricaPct     float64  `json:"conversao_teorica_pct"`
	ConversaoObservadaPct   float64  `json:"conversao_observada_pct"`
	ClientesAtivosInicio    int      `json:"clientes_ativos_inicio"`
	NovosDesejados          int      `json:"novos_desejados"`
	RecorrentesDesejados    int      `json:"recorrentes_desejados"`
	NovosClientes           int      `json:"novos_clientes"`
	ClientesRecorrentes     int      `json:"clientes_recorrentes"`
	ClientesAtivosFinal     int      `json:"clientes_ativos_final"`
	RecorrenciaObservadaPct float64  `json:"recorrencia_observada_pct"`
	Demanda                 int      `json:"demanda"`
	Capacidade              int      `json:"capacidade"`
	Vendas                  int      `json:"vendas"`
	VendasPerdidas          int      `json:"vendas_perdidas"`
	Gargalo                 string   `json:"gargalo"`
	PrecoEfetivo            float64  `json:"preco_efetivo"`
	DescontoPct             float64  `json:"desconto_pct"`
	Receita                 float64  `json:"receita"`
	TicketMedio             float64  `json:"ticket_medio"`
	CAC                     *float64 `json:"cac"`
	CMV                     float64  `json:"cmv"`
	TaxaPagamento           float64  `json:"taxa_pagamento"`
	ComissaoDelivery        float64  `json:"comissao_delivery"`
	TaxaMarketplace         float64  `json:"taxa_marketplace"`
	CustosFixos             float64  `json:"custos_fixos"`
	Juros                   float64  `json:"juros"`
	Resultado               float64  `json:"resultado"`
	RecebimentosAnteriores  float64  `json:"recebimentos_anteriores"`
	PagamentosAnteriores    float64  `json:"pagamentos_anteriores"`
	ReceitaImediata         float64  `json:"receita_imediata"`
	FluxoCaixa              float64  `json:"fluxo_caixa"`
	Caixa                   float64  `json:"caixa"`
	ContasReceber           float64  `json:"contas_receber"`
	ContasPagar             float64  `json:"contas_pagar"`
	Reputacao               float64  `json:"reputacao"`
	EstoqueUnidades         int      `json:"estoque_unidades"`
	EstoqueValor            float64  `json:"estoque_valor"`
	DesperdicioUnidades     int      `json:"desperdicio_unidades"`
	DesperdicioValor        float64  `json:"desperdicio_valor"`
	ConcorrenciaIndice      float64  `json:"concorrencia_indice"`
	SazonalidadeFator       float64  `json:"sazonalidade_fator"`
	MargemContribuicaoUnit  float64  `json:"margem_contribuicao_unit"`
	PontoEquilibrioVendas   *int     `json:"ponto_equilibrio_vendas"`
	MarketingSemanal        float64  `json:"marketing_semanal"`
	InsumoLimitante         string   `json:"insumo_limitante"`
	EntregasRecebidas       string   `json:"entregas_recebidas"`
	DesperdicioInsumos      float64  `json:"desperdicio_insumos"`
	DesperdicioDetalhes     string   `json:"desperdicio_detalhes"`
	Evento                  string   `json:"evento"`
}
type Empresa struct {
	LocalID                  string             `json:"local_id"`
	CreatedAt                string             `json:"created_at"`
	UpdatedAt                string             `json:"updated_at"`
	Revision                 int                `json:"revision"`
	SyncState                string             `json:"sync_state"`
	Nome                     string             `json:"nome"`
	Responsavel              string             `json:"responsavel"`
	TurmaID                  string             `json:"turma_id"`
	ModeloBase               string             `json:"modelo_base"`
	Categoria                string             `json:"categoria"`
	ModeloID                 string             `json:"modelo_id"`
	Setor                    string             `json:"setor"`
	TipoNegocio              string             `json:"tipo_negocio"`
	Especialidade            string             `json:"especialidade"`
	PerecibilidadeSemanal    float64            `json:"perecibilidade_semanal"`
	Regulamentado            bool               `json:"regulamentado"`
	CustoRegulatorioMensal   float64            `json:"custo_regulatorio_mensal"`
	ReputacaoImportancia     float64            `json:"reputacao_importancia"`
	DeliveryAfinidade        float64            `json:"delivery_afinidade"`
	Dificuldade              string             `json:"dificuldade"`
	CapitalProprio           float64            `json:"capital_proprio"`
	EmprestimoInicial        float64            `json:"emprestimo_inicial"`
	JurosMensal              float64            `json:"juros_mensal"`
	Divida                   float64            `json:"divida"`
	Caixa                    float64            `json:"caixa"`
	Preco                    float64            `json:"preco"`
	PrecoReferencia          float64            `json:"preco_referencia"`
	CustoUnitario            float64            `json:"custo_unitario"`
	AlcanceBase              int                `json:"alcance_base"`
	ConversaoBase            float64            `json:"conversao_base"`
	RecorrenciaBase          float64            `json:"recorrencia_base"`
	CapacidadeBase           int                `json:"capacidade_base"`
	SazonalidadeAmplitude    float64            `json:"sazonalidade_amplitude"`
	SazonalidadeFase         float64            `json:"sazonalidade_fase"`
	UsaEstoque               bool               `json:"usa_estoque"`
	EstoqueUnidades          int                `json:"estoque_unidades"`
	EstoqueValor             float64            `json:"estoque_valor"`
	CustoMedioEstoque        float64            `json:"custo_medio_estoque"`
	UsaInsumos               bool               `json:"usa_insumos"`
	Insumos                  []InsumoEstoque    `json:"insumos"`
	PedidosInsumos           []PedidoInsumo     `json:"pedidos_insumos"`
	Operacao                 string             `json:"operacao"`
	Imovel                   string             `json:"imovel"`
	AluguelMensal            float64            `json:"aluguel_mensal"`
	QualidadeLocalizacao     string             `json:"qualidade_localizacao"`
	Delivery                 bool               `json:"delivery"`
	DeliveryTipo             string             `json:"delivery_tipo"`
	Funcionarios             int                `json:"funcionarios"`
	SalarioMedio             float64            `json:"salario_medio"`
	MarketingSemanal         float64            `json:"marketing_semanal"`
	MeiosPagamento           []string           `json:"meios_pagamento"`
	TaxaCartao               float64            `json:"taxa_cartao"`
	PrazoCartaoSemanas       int                `json:"prazo_cartao_semanas"`
	PrazoFornecedorMax       int                `json:"prazo_fornecedor_max"`
	Cenario                  string             `json:"cenario"`
	CenarioAlcance           float64            `json:"cenario_alcance"`
	CenarioConversao         float64            `json:"cenario_conversao"`
	CenarioOscilacao         float64            `json:"cenario_oscilacao"`
	CenarioEventoNegExtra    float64            `json:"cenario_evento_negativo_extra"`
	DuracaoSemanas           int                `json:"duracao_semanas"`
	ConcorrenciaNivel        string             `json:"concorrencia_nivel"`
	ConcorrenciaIndice       float64            `json:"concorrencia_indice"`
	Reputacao                float64            `json:"reputacao"`
	Semana                   int                `json:"semana"`
	ClientesAtivos           int                `json:"clientes_ativos"`
	ClientesTotais           int                `json:"clientes_totais"`
	ClientesRecorrentesTotal int                `json:"clientes_recorrentes_total"`
	PromocaoDesconto         float64            `json:"promocao_desconto"`
	Canvas                   LeanCanvas         `json:"canvas"`
	Persona                  Persona            `json:"persona"`
	CanaisDigitais           []string           `json:"canais_digitais"`
	ModeloDigital            string             `json:"modelo_digital,omitempty"`
	FerramentasDigitais      []string           `json:"ferramentas_digitais"`
	Hipoteses                Hipoteses          `json:"hipoteses"`
	InvestimentosIniciais    map[string]float64 `json:"investimentos_iniciais"`
	ContasPagar              []Conta            `json:"contas_pagar"`
	ContasReceber            []Conta            `json:"contas_receber"`
	Historico                []Registro         `json:"historico"`
	RevisoesCanvas           int                `json:"revisoes_canvas"`
	VersaoDados              string             `json:"versao_dados"`
}
type Modelo struct {
	ID                     string  `json:"id"`
	Nome                   string  `json:"nome"`
	PrecoRef               float64 `json:"preco_ref"`
	CustoUnitario          float64 `json:"custo_unitario"`
	AlcanceBase            int     `json:"alcance_base"`
	ConversaoBase          float64 `json:"conversao_base"`
	RecorrenciaBase        float64 `json:"recorrencia_base"`
	CapacidadeBase         int     `json:"capacidade_base"`
	Estoque                bool    `json:"estoque"`
	Funcionarios           int     `json:"funcionarios"`
	Aluguel                float64 `json:"aluguel"`
	Equipamentos           float64 `json:"equipamentos"`
	EstoqueInicialUn       int     `json:"estoque_inicial_un"`
	Sazonalidade           float64 `json:"sazonalidade"`
	Fase                   float64 `json:"fase"`
	PerecibilidadeSemanal  float64 `json:"perecibilidade_semanal"`
	Regulamentado          bool    `json:"regulamentado"`
	CustoRegulatorioMensal float64 `json:"custo_regulatorio_mensal"`
	ReputacaoImportancia   float64 `json:"reputacao_importancia"`
	DeliveryAfinidade      float64 `json:"delivery_afinidade"`
	ModeloDigital          string  `json:"modelo_digital,omitempty"`
	SetorID                string  `json:"-"`
	SetorNome              string  `json:"-"`
	TipoID                 string  `json:"-"`
	TipoNome               string  `json:"-"`
}
type TipoCatalogo struct {
	ID             string   `json:"id"`
	Nome           string   `json:"nome"`
	Especialidades []Modelo `json:"especialidades"`
}
type SetorCatalogo struct {
	ID    string         `json:"id"`
	Nome  string         `json:"nome"`
	Tipos []TipoCatalogo `json:"tipos"`
}
type Catalogo struct {
	Versao    string          `json:"versao"`
	Descricao string          `json:"descricao"`
	Setores   []SetorCatalogo `json:"setores"`
}
type Cenario struct {
	LocalID             string  `json:"local_id,omitempty"`
	CreatedAt           string  `json:"created_at,omitempty"`
	UpdatedAt           string  `json:"updated_at,omitempty"`
	Revision            int     `json:"revision,omitempty"`
	Nome                string  `json:"nome"`
	Alcance             float64 `json:"alcance"`
	Conversao           float64 `json:"conversao"`
	Oscilacao           float64 `json:"oscilacao"`
	EventoNegativoExtra float64 `json:"evento_negativo_extra"`
	Duracao             int     `json:"duracao"`
	ConcorrenciaNivel   string  `json:"concorrencia_nivel,omitempty"`
	ConcorrenciaIndice  float64 `json:"concorrencia_indice,omitempty"`
	Observacoes         string  `json:"observacoes,omitempty"`
	Dificuldade         string  `json:"dificuldade,omitempty"`
}
type Turma struct {
	LocalID   string  `json:"local_id,omitempty"`
	CreatedAt string  `json:"created_at,omitempty"`
	UpdatedAt string  `json:"updated_at,omitempty"`
	Revision  int     `json:"revision,omitempty"`
	ID        string  `json:"id"`
	Nome      string  `json:"nome"`
	Tutor     string  `json:"tutor"`
	Cenario   Cenario `json:"cenario"`
}
type Evento struct {
	Texto             string
	Alcance           float64
	Conversao         float64
	Reputacao         float64
	ConcorrenciaDelta float64
}
type Indicadores struct {
	Semanas, Vendas, Novos, Recorrentes, Perdidas, Alcance, ClientesAtivos                    int
	Receita, Resultado, ConversaoAcumuladaPct, CACAprox, TicketMedio, Caixa, AReceber, APagar float64
}
type Score struct {
	Total, Financeiro, Mercado, Operacao, Hipoteses, Gestao float64
	Observacoes                                             []string
}
type Review struct {
	MediaVendas, MetaVendas, MediaReceita, MetaReceita, MediaConversao, MetaConversao, MediaRecorrencia, MetaRecorrencia float64
	StatusVendas, StatusReceita, StatusConversao, StatusRecorrencia                                                      string
	SemanasPositivas                                                                                                     int
	EsperavaPositivo, PositivoObservado                                                                                  bool
}
