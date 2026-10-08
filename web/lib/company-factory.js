let catalogPromise;

export const baseScenarios = [
  { id: "base-estavel", nome: "Mercado estável", alcance: 1, conversao: 1, oscilacao: 0.08, evento_negativo_extra: 0, duracao: 12, concorrencia_nivel: "media", concorrencia_indice: 1, dificuldade: "intermediario" },
  { id: "base-aquecido", nome: "Mercado aquecido", alcance: 1.10, conversao: 1.04, oscilacao: 0.10, evento_negativo_extra: -0.05, duracao: 12, concorrencia_nivel: "media", concorrencia_indice: 1, dificuldade: "intermediario" },
  { id: "base-desaceleracao", nome: "Desaceleração econômica", alcance: 0.90, conversao: 0.92, oscilacao: 0.12, evento_negativo_extra: 0.08, duracao: 12, concorrencia_nivel: "media", concorrencia_indice: 1, dificuldade: "intermediario" },
  { id: "base-concorrencia", nome: "Concorrência intensa", alcance: 0.96, conversao: 0.94, oscilacao: 0.10, evento_negativo_extra: 0.05, duracao: 12, concorrencia_nivel: "alta", concorrencia_indice: 1.12, dificuldade: "avancado" },
];

export async function loadCatalogs() {
  if (!catalogPromise) {
    catalogPromise = Promise.all([
      fetch("/data/catalogo_negocios.json", { cache: "force-cache" }).then((r) => {
        if (!r.ok) throw new Error("Catálogo de negócios indisponível.");
        return r.json();
      }),
      fetch("/data/catalogo_insumos.json", { cache: "force-cache" }).then((r) => {
        if (!r.ok) throw new Error("Catálogo de insumos indisponível.");
        return r.json();
      }),
    ]).then(([business, supplies]) => ({ business, supplies }));
  }
  return catalogPromise;
}

export function sectorOptions(catalog) {
  return Array.isArray(catalog?.setores) ? catalog.setores : [];
}

export function typeOptions(catalog, sectorID) {
  return sectorOptions(catalog).find((item) => item.id === sectorID)?.tipos || [];
}

export function specialtyOptions(catalog, sectorID, typeID) {
  return typeOptions(catalog, sectorID).find((item) => item.id === typeID)?.especialidades || [];
}

export function enrichedModel(catalog, sectorID, typeID, modelID) {
  const sector = sectorOptions(catalog).find((item) => item.id === sectorID);
  const type = sector?.tipos?.find((item) => item.id === typeID);
  const model = type?.especialidades?.find((item) => item.id === modelID);
  return model ? { ...model, setor_id: sector.id, setor_nome: sector.nome, tipo_id: type.id, tipo_nome: type.nome } : null;
}

function supplyProfile(model, supplies) {
  const configured = supplies?.perfis?.find((profile) => profile.modelo_id === model.id);
  if (configured) return configured;
  if (!model.estoque) return { modelo_id: model.id, insumos: [] };
  const loss = Number(model.perecibilidade_semanal || 0);
  const validity = loss >= 0.10 ? 1 : loss >= 0.05 ? 2 : loss > 0 ? 6 : 24;
  return {
    modelo_id: model.id,
    insumos: [{
      id: "mercadoria",
      nome: "Mercadoria / insumo principal",
      unidade: "un",
      custo_base: Number(model.custo_unitario || 0),
      consumo_por_venda: 1,
      estoque_inicial: Number(model.estoque_inicial_un || 0),
      validade_semanas: validity,
      perda_semanal: loss,
      critico: true,
    }],
  };
}

function slugID() {
  if (globalThis.crypto?.randomUUID) return `web-${globalThis.crypto.randomUUID()}`;
  return `web-${Date.now()}-${Math.random().toString(16).slice(2)}`;
}

export function createInitialCompany({ model, supplies, user, name, capital, difficulty, classInfo, scenario }) {
  if (!model) throw new Error("Escolha uma especialidade de negócio.");
  const cap = Math.max(0, Number(capital || 0));
  const profile = supplyProfile(model, supplies);
  const inputs = [];
  let inputValue = 0;
  for (const spec of profile.insumos || []) {
    const quantity = Number(spec.estoque_inicial || 0);
    const cost = Number(spec.custo_base || 0);
    inputValue += quantity * cost;
    inputs.push({
      id: spec.id,
      nome: spec.nome,
      unidade: spec.unidade,
      quantidade: quantity,
      custo_medio: cost,
      custo_referencia: cost,
      consumo_por_venda: Number(spec.consumo_por_venda || 0),
      validade_semanas: Number(spec.validade_semanas || 0),
      perda_semanal: Number(spec.perda_semanal || 0),
      critico: Boolean(spec.critico),
    });
  }

  const equipment = Number(model.equipamentos || 0);
  const licenses = 500;
  const launch = 600;
  let cash = cap - equipment - licenses - inputValue - launch;
  if (cash < 0) cash = Math.max(1000, cap * 0.20);

  const env = scenario || classInfo?.scenario || baseScenarios[0];
  const scenarioDuration = Math.max(1, Number(env?.duracao || (classInfo ? 12 : 24)));
  const weeklySales = Math.max(5, Number(model.alcance_base || 0) * Number(model.conversao_base || 0));
  const now = new Date().toISOString();
  const localID = slugID();

  return {
    local_id: localID,
    created_at: now,
    updated_at: now,
    revision: 0,
    sync_state: "local",
    nome: String(name || model.nome || "Novo empreendimento").trim(),
    responsavel: user?.name || "Equipe",
    turma_id: classInfo?.id || "",
    modelo_base: model.nome,
    categoria: model.setor_id,
    modelo_id: model.id,
    setor: model.setor_nome,
    tipo_negocio: model.tipo_nome,
    especialidade: model.nome,
    perecibilidade_semanal: Number(model.perecibilidade_semanal || 0),
    regulamentado: Boolean(model.regulamentado),
    custo_regulatorio_mensal: Number(model.custo_regulatorio_mensal || 0),
    reputacao_importancia: Number(model.reputacao_importancia || 1),
    delivery_afinidade: Number(model.delivery_afinidade || 1),
    dificuldade: String(env?.dificuldade || difficulty || "intermediario"),
    capital_proprio: cap,
    emprestimo_inicial: 0,
    juros_mensal: 0,
    divida: 0,
    caixa: cash,
    preco: Number(model.preco_ref || 0),
    preco_referencia: Number(model.preco_ref || 0),
    custo_unitario: Number(model.custo_unitario || 0),
    alcance_base: Number(model.alcance_base || 0),
    conversao_base: Number(model.conversao_base || 0),
    recorrencia_base: Number(model.recorrencia_base || 0),
    capacidade_base: Number(model.capacidade_base || 0),
    sazonalidade_amplitude: Number(model.sazonalidade || 0),
    sazonalidade_fase: Number(model.fase || 0),
    usa_estoque: Boolean(model.estoque),
    estoque_unidades: Number(model.estoque_inicial_un || 0),
    estoque_valor: inputValue || Number(model.estoque_inicial_un || 0) * Number(model.custo_unitario || 0),
    custo_medio_estoque: Number(model.custo_unitario || 0),
    usa_insumos: inputs.length > 0,
    insumos: inputs,
    pedidos_insumos: [],
    operacao: "hibrida",
    imovel: "alugado",
    aluguel_mensal: Number(model.aluguel || 0),
    qualidade_localizacao: "media",
    delivery: Number(model.delivery_afinidade || 1) > 1.05,
    delivery_tipo: "misto",
    funcionarios: Number(model.funcionarios || 0),
    salario_medio: 1800,
    marketing_semanal: 150,
    meios_pagamento: ["pix", "cartao"],
    taxa_cartao: 2.5,
    prazo_cartao_semanas: 1,
    prazo_fornecedor_max: 2,
    cenario: env?.nome || "Mercado estável",
    cenario_alcance: Number(env?.alcance || 1),
    cenario_conversao: Number(env?.conversao || 1),
    cenario_oscilacao: Number(env?.oscilacao ?? 0.08),
    cenario_evento_negativo_extra: Number(env?.evento_negativo_extra || 0),
    duracao_semanas: scenarioDuration,
    concorrencia_nivel: String(env?.concorrencia_nivel || "media"),
    concorrencia_indice: Number(env?.concorrencia_indice || 1),
    reputacao: 50,
    semana: 0,
    clientes_ativos: 0,
    clientes_totais: 0,
    clientes_recorrentes_total: 0,
    promocao_desconto: 0,
    canvas: {
      problema: "Validar necessidade do cliente",
      segmentos: ["geral"],
      proposta_valor: "qualidade",
      solucao: model.nome,
      canais: ["redes_sociais"],
      receita_modelo: "venda_unica",
      custos_notas: "Custos operacionais e aquisição",
      metricas: ["alcance", "interações", "leads", "vendas", "caixa", "conversão"],
      vantagem: "A desenvolver",
    },
    persona: {
      nome: "Cliente principal",
      demografia: "A definir",
      rotinas: "A definir",
      objetivos: "A definir",
      desafios: "A definir",
      motivadores: "A definir",
      objecoes: "A definir",
      citacoes: "A definir",
      palavras_chave: "A definir",
    },
    canais_digitais: [],
    modelo_digital: model.modelo_digital || "",
    ferramentas_digitais: [],
    hipoteses: {
      vendas_semanais: weeklySales,
      faturamento_semanal: weeklySales * Number(model.preco_ref || 0),
      espera_resultado_positivo: true,
      conversao_pct: Number(model.conversao_base || 0) * 100,
      recorrencia_pct: Number(model.recorrencia_base || 0) * 100,
    },
    investimentos_iniciais: {
      equipamentos: equipment,
      equipamentos_referencia: equipment,
      licencas: licenses,
      estoque_inicial: inputValue,
      marketing_lancamento: launch,
      capital_giro_inicial: cash,
    },
    contas_pagar: [],
    contas_receber: [],
    historico: [],
    revisoes_canvas: 0,
    versao_dados: "2.0.0-rc1.8",
  };
}
