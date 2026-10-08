"use client";

import { useEffect, useMemo, useRef, useState } from "react";
import { createJEDClient } from "@/lib/jed-core";
import { baseScenarios, createInitialCompany, enrichedModel, loadCatalogs, sectorOptions, specialtyOptions, typeOptions } from "@/lib/company-factory";
import Modal from "@/components/modal";

const money = new Intl.NumberFormat("pt-BR", { style: "currency", currency: "BRL" });
const number = new Intl.NumberFormat("pt-BR", { maximumFractionDigits: 2 });
const integer = new Intl.NumberFormat("pt-BR", { maximumFractionDigits: 0 });

function clone(value) { return JSON.parse(JSON.stringify(value ?? {})); }
function csv(value) { return Array.isArray(value) ? value.join(", ") : ""; }
function list(value) { return String(value || "").split(",").map((item) => item.trim()).filter(Boolean); }
function fmtDate(value) { if (!value) return "—"; const d = new Date(value); return Number.isNaN(d.getTime()) ? value : new Intl.DateTimeFormat("pt-BR", { dateStyle: "short", timeStyle: "short" }).format(d); }

function Metric({ label, value, detail }) {
  return <article className="metric-card"><span>{label}</span><strong>{value}</strong>{detail ? <small>{detail}</small> : null}</article>;
}

function DataRow({ label, value }) { return <div><dt>{label}</dt><dd>{value ?? "—"}</dd></div>; }

function CompanyPicker({ companies, selectedID, onChange }) {
  return <label className="company-picker"><span>EMPRESA ATIVA</span><select value={selectedID || ""} onChange={(e) => onChange(e.target.value)}>{companies.map((remote) => <option value={remote.id} key={remote.id}>{remote.company?.nome || remote.id}</option>)}</select></label>;
}

function Checklist({ title, items, selected, onToggle, disabled }) {
  return <div className="checklist-block"><div className="workspace-section-head compact"><div><p className="eyebrow">{title}</p></div><span className="section-count">{selected?.length || 0} ATIVOS</span></div><div className="checklist-grid">{items.map((item) => <label className={selected?.includes(item.ID ?? item.id) ? "choice-card selected" : "choice-card"} key={item.ID ?? item.id}><input type="checkbox" checked={selected?.includes(item.ID ?? item.id) || false} onChange={() => onToggle(item.ID ?? item.id)} disabled={disabled} /><strong>{item.Nome ?? item.nome}</strong><span>{item.Descricao ?? item.descricao ?? ""}</span>{item.CustoSemanal ? <small>{money.format(item.CustoSemanal)}/semana</small> : null}</label>)}</div></div>;
}

function RequireCompany({ onCreate }) {
  return <section className="workspace-card empty-module"><p className="eyebrow">EMPRESA NECESSÁRIA</p><h2>Crie seu empreendimento para liberar este módulo.</h2><p className="student-muted">A versão Web agora cria a empresa diretamente pelo catálogo do Orfeu. Não é necessário voltar ao programa Windows.</p><button type="button" className="primary-button" onClick={onCreate}>NOVO EMPREENDIMENTO</button></section>;
}

export default function StudentWorkspace({ user }) {
  const coreRef = useRef(null);
  const [activeView, setActiveView] = useState("empresa");
  const [companies, setCompanies] = useState([]);
  const [classes, setClasses] = useState([]);
  const [selectedID, setSelectedID] = useState("");
  const [draft, setDraft] = useState(null);
  const [catalogs, setCatalogs] = useState(null);
  const [core, setCore] = useState({ state: "loading", channels: [], tools: [], indicators: null, score: null, journey: 1, canvasNotes: [] });
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);
  const [dirty, setDirty] = useState(false);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const [dialog, setDialog] = useState("");
  const [deletePhrase, setDeletePhrase] = useState("");
  const [newCompany, setNewCompany] = useState({ sector: "", type: "", model: "", name: "", capital: 20000, difficulty: "intermediario", class_id: "", scenario_id: "base-estavel" });
  const [order, setOrder] = useState({ input_id: "", quantity: 10, supplier_id: "padrao", term: 0 });
  const [stockOrder, setStockOrder] = useState({ quantity: 10, term: 0 });

  useEffect(() => {
    const onNavigate = (event) => {
      if (event?.detail?.role !== "aluno") return;
      const view = String(event.detail.view || "empresa");
      setActiveView(view);
      window.dispatchEvent(new CustomEvent("jed:view", { detail: { role: "aluno", view } }));
    };
    window.addEventListener("jed:navigate", onNavigate);
    return () => window.removeEventListener("jed:navigate", onNavigate);
  }, []);

  useEffect(() => {
    let cancelled = false;
    async function boot() {
      setLoading(true); setError("");
      try {
        const [companiesResponse, classesResponse, loadedCatalogs, client] = await Promise.all([
          fetch("/api/student/companies", { cache: "no-store" }),
          fetch("/api/student/classes", { cache: "no-store" }),
          loadCatalogs(),
          createJEDClient(Date.now()),
        ]);
        const companiesPayload = await companiesResponse.json();
        const classesPayload = await classesResponse.json();
        if (!companiesResponse.ok) throw new Error(companiesPayload?.error || "Falha ao carregar empresas.");
        if (!classesResponse.ok) throw new Error(classesPayload?.error || "Falha ao carregar turmas.");
        if (cancelled) { try { client.close(); } catch {} return; }
        coreRef.current = client;
        const channels = client.digitalChannels();
        const tools = client.digitalTools();
        const nextCompanies = Array.isArray(companiesPayload.companies) ? companiesPayload.companies : [];
        const nextClasses = Array.isArray(classesPayload.classes) ? classesPayload.classes : [];
        setCatalogs(loadedCatalogs);
        setCompanies(nextCompanies);
        setClasses(nextClasses);
        setNewCompany((current) => ({ ...current, class_id: nextClasses[0]?.id || "" }));
        setSelectedID(nextCompanies[0]?.id || "");
        setCore((current) => ({ ...current, state: "ready", channels, tools }));
      } catch (caught) {
        setCore((current) => ({ ...current, state: "error" }));
        setError(caught instanceof Error ? caught.message : "Falha ao iniciar a versão Web.");
      } finally { if (!cancelled) setLoading(false); }
    }
    void boot();
    return () => { cancelled = true; try { coreRef.current?.close(); } catch {} coreRef.current = null; };
  }, []);

  const selected = useMemo(() => companies.find((item) => item.id === selectedID) || null, [companies, selectedID]);

  useEffect(() => {
    if (!selected) { setDraft(null); setDirty(false); return; }
    setDraft(clone(selected.company)); setDirty(false);
  }, [selectedID, selected?.revision]);

  useEffect(() => {
    const client = coreRef.current;
    if (!client || !draft || core.state !== "ready") return;
    try {
      const indicators = client.indicators(draft);
      const score = client.score(draft);
      const journey = client.journeyStep(draft)?.step || 1;
      const canvasNotes = client.canvasExplanations(draft) || [];
      setCore((current) => ({ ...current, indicators, score, journey, canvasNotes }));
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : "Falha ao calcular indicadores.");
    }
  }, [draft, core.state]);

  useEffect(() => {
    if (!catalogs?.business) return;
    const sectors = sectorOptions(catalogs.business);
    if (!newCompany.sector && sectors[0]) {
      const types = sectors[0].tipos || [];
      const models = types[0]?.especialidades || [];
      setNewCompany((current) => ({ ...current, sector: sectors[0].id, type: types[0]?.id || "", model: models[0]?.id || "" }));
    }
  }, [catalogs, newCompany.sector]);

  function activate(view) {
    setActiveView(view);
    window.dispatchEvent(new CustomEvent("jed:view", { detail: { role: "aluno", view } }));
  }

  function edit(field, value) { setDraft((current) => ({ ...current, [field]: value })); setDirty(true); setNotice(""); }
  function editNested(group, field, value) { setDraft((current) => ({ ...current, [group]: { ...(current?.[group] || {}), [field]: value } })); setDirty(true); setNotice(""); }
  function toggleList(field, id) { setDraft((current) => { const items = new Set(current?.[field] || []); if (items.has(id)) items.delete(id); else items.add(id); return { ...current, [field]: [...items] }; }); setDirty(true); }

  async function syncCompany(company = draft, classID = selected?.class_id || company?.turma_id || "", silent = false) {
    if (!company?.local_id) throw new Error("Empresa sem identificador local.");
    const response = await fetch("/api/student/companies", { method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ local_id: company.local_id, class_id: classID, company }) });
    const payload = await response.json();
    if (!response.ok) throw new Error(payload?.error || "Falha ao sincronizar empresa.");
    const remote = payload.company;
    setCompanies((current) => {
      const exists = current.some((item) => item.id === remote.id);
      return exists ? current.map((item) => item.id === remote.id ? remote : item) : [...current, remote];
    });
    setSelectedID(remote.id);
    setDraft(clone(remote.company));
    setDirty(false);
    if (!silent) setNotice("Empresa salva e sincronizada com o servidor de simulação.");
    return remote;
  }

  async function saveDraft() {
    if (!draft) return;
    setBusy(true); setError(""); setNotice("");
    try { await syncCompany(draft, selected?.class_id || draft.turma_id || classes[0]?.id || ""); }
    catch (caught) { setError(caught instanceof Error ? caught.message : "Falha ao salvar empresa."); }
    finally { setBusy(false); }
  }

  async function deleteSelectedCompany(event) {
    event.preventDefault();
    if (!selected || deletePhrase.trim().toUpperCase() !== "EXCLUIR" || busy) return;
    const localID = String(selected.company?.local_id || "");
    if (!localID) { setError("Empreendimento sem identificador."); return; }
    setBusy(true); setError(""); setNotice("");
    try {
      const response = await fetch("/api/student/companies", {
        method: "DELETE",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ local_id: localID }),
      });
      const payload = await response.json();
      if (!response.ok) throw new Error(payload?.error || "Não foi possível excluir o empreendimento.");
      const next = companies.filter((entry) => entry.id !== selected.id);
      setCompanies(next);
      setSelectedID(next[0]?.id || "");
      setDraft(next.length ? clone(next[0].company) : null);
      setDirty(false);
      setDialog(""); setDeletePhrase("");
      setNotice("Empreendimento excluído do servidor. As demais empresas foram preservadas.");
    } catch (caught) { setError(caught instanceof Error ? caught.message : "Falha ao excluir empreendimento."); }
    finally { setBusy(false); }
  }

  async function createCompany(event) {
    event.preventDefault();
    if (!catalogs) return;
    const model = enrichedModel(catalogs.business, newCompany.sector, newCompany.type, newCompany.model);
    const classInfo = classes.find((item) => item.id === newCompany.class_id) || null;
    const scenario = classInfo?.scenario || baseScenarios.find((item) => item.id === newCompany.scenario_id) || baseScenarios[0];
    setBusy(true); setError(""); setNotice("");
    try {
      const company = createInitialCompany({ model, supplies: catalogs.supplies, user, name: newCompany.name, capital: newCompany.capital, difficulty: newCompany.difficulty, classInfo, scenario });
      await syncCompany(company, classInfo?.id || "", true);
      setDialog(""); setNotice(`Empresa ${company.nome} criada. Complete a Persona e o Lean Canvas antes da primeira rodada.`); activate("persona");
      setNewCompany((current) => ({ ...current, name: "" }));
    } catch (caught) { setError(caught instanceof Error ? caught.message : "Falha ao criar empresa."); }
    finally { setBusy(false); }
  }

  async function processWeek() {
    const client = coreRef.current;
    if (!client || !draft) return;
    setBusy(true); setError(""); setNotice("");
    try {
      const result = client.processWeek(draft);
      const next = result.empresa;
      await syncCompany(next, selected?.class_id || next.turma_id || "", true);
      setNotice(`Semana ${result.registro?.semana || next.semana} processada pelo motor de simulação e sincronizada.`);
    } catch (caught) { setError(caught instanceof Error ? caught.message : "Falha ao processar semana."); }
    finally { setBusy(false); }
  }

  async function placeOrder(event) {
    event.preventDefault();
    const client = coreRef.current;
    if (!client || !draft || !catalogs) return;
    const supplier = catalogs.supplies?.fornecedores?.find((item) => item.id === order.supplier_id);
    if (!supplier) { setError("Fornecedor não encontrado."); return; }
    setBusy(true); setError(""); setNotice("");
    try {
      const result = client.placeInputOrder(draft, order.input_id || draft.insumos?.[0]?.id, Number(order.quantity), supplier, Number(order.term));
      if (!result.accepted) throw new Error(result.note || "Pedido não aceito.");
      await syncCompany(result.empresa, selected?.class_id || result.empresa.turma_id || "", true);
      setNotice(`Pedido registrado: ${result.note}. Custo ${money.format(Number(result.cost || 0))}.`);
    } catch (caught) { setError(caught instanceof Error ? caught.message : "Falha ao registrar pedido."); }
    finally { setBusy(false); }
  }

  async function buyStock(event) {
    event.preventDefault();
    const client = coreRef.current;
    if (!client || !draft) return;
    setBusy(true); setError(""); setNotice("");
    try {
      const result = client.buyStock(draft, Number(stockOrder.quantity), Number(stockOrder.term));
      if (!result.accepted) throw new Error(result.note || "Compra não aceita.");
      await syncCompany(result.empresa, selected?.class_id || result.empresa.turma_id || "", true);
      setNotice(`Compra registrada: ${result.note}. Custo ${money.format(Number(result.cost || 0))}.`);
    } catch (caught) { setError(caught instanceof Error ? caught.message : "Falha ao comprar estoque."); }
    finally { setBusy(false); }
  }

  if (loading) return <section className="student-loading">CARREGANDO MOTOR ORFEU, CATÁLOGOS E SERVIDOR…</section>;

  const company = draft;
  const sectors = sectorOptions(catalogs?.business);
  const types = typeOptions(catalogs?.business, newCompany.sector);
  const models = specialtyOptions(catalogs?.business, newCompany.sector, newCompany.type);
  const currentModel = enrichedModel(catalogs?.business, newCompany.sector, newCompany.type, newCompany.model);
  const indicators = core.indicators;
  const score = core.score;
  const last = company?.historico?.length ? company.historico[company.historico.length - 1] : null;

  function companyToolbar() {
    return <div className="orbit-toolbar sticky-module-toolbar"><button type="button" className="primary-button" onClick={() => setDialog("company")}>NOVO EMPREENDIMENTO</button>{companies.length ? <CompanyPicker companies={companies} selectedID={selectedID} onChange={(id) => { if (!dirty || window.confirm("Trocar de empresa e descartar alterações não salvas?")) setSelectedID(id); }} /> : null}{company ? <button type="button" className="secondary-button" onClick={saveDraft} disabled={!dirty || busy}>{busy ? "SALVANDO…" : dirty ? "SALVAR NO SERVIDOR" : "SINCRONIZADO"}</button> : null}{company ? <button type="button" className="danger-button" disabled={busy} onClick={() => { setDeletePhrase(""); setDialog("delete-company"); }}>EXCLUIR EMPREENDIMENTO</button> : null}</div>;
  }

  function renderEmpresa() {
    if (!company) return <><section className="orbit-overview-strip"><div><span>EMPRESA</span><strong>NENHUM EMPREENDIMENTO CRIADO</strong><small>Crie sua empresa diretamente no navegador usando o catálogo oficial do Orfeu.</small></div><div className="orbit-stat-row"><div><span>TURMAS</span><strong>{classes.length}</strong></div><div><span>CORE</span><strong>{core.state === "ready" ? "OK" : "—"}</strong></div></div></section>{companyToolbar()}<RequireCompany onCreate={() => setDialog("company")} /></>;
    return <><section className="student-company-head"><div><p className="eyebrow">MINHA EMPRESA</p><h1>{company.nome}</h1><p className="lede">{company.setor} → {company.tipo_negocio} → {company.especialidade}</p></div><div><span className="technical-label">SEMANA</span><strong className="big-week">{company.semana || 0}/{company.duracao_semanas || "—"}</strong></div></section>{companyToolbar()}<section className="student-metrics-grid"><Metric label="Caixa" value={money.format(Number(company.caixa || 0))} /><Metric label="Preço" value={money.format(Number(company.preco || 0))} /><Metric label="Clientes ativos" value={integer.format(Number(company.clientes_ativos || 0))} /><Metric label="Reputação" value={number.format(Number(company.reputacao || 0))} /><Metric label="Receita acumulada" value={indicators ? money.format(Number(indicators.Receita || 0)) : "…"} /><Metric label="Resultado acumulado" value={indicators ? money.format(Number(indicators.Resultado || 0)) : "…"} /><Metric label="Índice Orfeu" value={score ? number.format(Number(score.Total || 0)) : "…"} detail="motor Go/WASM" /><Metric label="Revisão" value={`#${selected?.revision || company.revision || 0}`} detail={fmtDate(selected?.updated_at || company.updated_at)} /></section><section className="workspace-two-col"><article className="workspace-card"><p className="eyebrow">IDENTIDADE</p><h2>Configuração do empreendimento</h2><dl className="student-data-list"><DataRow label="Responsável" value={company.responsavel} /><DataRow label="Cenário" value={company.cenario} /><DataRow label="Dificuldade" value={company.dificuldade} /><DataRow label="Capital próprio" value={money.format(Number(company.capital_proprio || 0))} /><DataRow label="Capacidade base" value={integer.format(Number(company.capacidade_base || 0))} /><DataRow label="Concorrência" value={company.concorrencia_nivel} /></dl></article><article className="workspace-card"><p className="eyebrow">ÚLTIMA RODADA</p><h2>{last ? `Semana ${last.semana}` : "Aguardando primeira semana"}</h2>{last ? <dl className="student-data-list"><DataRow label="Vendas" value={last.vendas} /><DataRow label="Receita" value={money.format(Number(last.receita || 0))} /><DataRow label="Resultado" value={money.format(Number(last.resultado || 0))} /><DataRow label="Evento" value={last.evento || "Nenhum"} /></dl> : <p className="student-muted">Complete Persona, Lean Canvas e decisões e então processe a primeira semana.</p>}</article></section><article className="workspace-card mentor-evaluation-card"><div className="evaluation-card-head"><div><p className="eyebrow">AVALIAÇÃO DO MENTOR</p><h2>Parecer sobre o empreendimento</h2></div>{selected?.approval_status === "aprovado" ? <span className="state-label approved">APROVADO</span> : selected?.approval_status === "reprovado" ? <span className="state-label rejected">REPROVADO</span> : <span className="state-label used">NÃO AVALIADO</span>}</div>{selected?.approval_status ? <><p className="mentor-evaluation-comment">{selected?.mentor_comment || "O Mentor não registrou comentários adicionais."}</p><small className="student-muted">{selected?.evaluated_by_name || "Mentor"} · {fmtDate(selected?.evaluated_at)}</small></> : <p className="student-muted">O Mentor ainda não classificou este empreendimento.</p>}</article></>;
  }

  function renderPersona() {
    if (!company) return <RequireCompany onCreate={() => setDialog("company")} />;
    const fields = [["nome", "Nome / identidade"],["demografia", "Demografia"],["rotinas", "Rotinas"],["objetivos", "Objetivos"],["desafios", "Desafios"],["motivadores", "Motivadores"],["objecoes", "Objeções"],["citacoes", "Citações"],["palavras_chave", "Palavras-chave"]];
    return <><div className="workspace-section-head"><div><p className="eyebrow">PERSONA</p><h2>Cliente principal</h2></div><button className="primary-button" type="button" onClick={saveDraft} disabled={!dirty || busy}>SALVAR PERSONA</button></div><p className="section-help">Descreva uma pessoa reconhecível, com necessidades, hábitos, objetivos e objeções. Estes dados influenciam o acompanhamento pedagógico e o Canvas.</p><div className="form-card-grid">{fields.map(([field,label]) => <label className="workspace-card edit-card" key={field}><span className="technical-label">{label}</span><textarea rows={field === "nome" ? 2 : 4} value={company.persona?.[field] || ""} onChange={(e) => editNested("persona", field, e.target.value)} /></label>)}</div></>;
  }

  function renderCanvas() {
    if (!company) return <RequireCompany onCreate={() => setDialog("company")} />;
    const c = company.canvas || {};
    return <><div className="workspace-section-head"><div><p className="eyebrow">LEAN CANVAS</p><h2>Hipóteses do modelo de negócio</h2></div><button className="primary-button" type="button" onClick={saveDraft} disabled={!dirty || busy}>SALVAR CANVAS</button></div>{core.canvasNotes?.length ? <div className="workspace-alert workspace-alert-success">EFEITOS ATIVOS · {core.canvasNotes.join(" · ")}</div> : null}<div className="canvas-grid"><label className="workspace-card edit-card"><span className="technical-label">PROBLEMA</span><textarea rows="5" value={c.problema || ""} onChange={(e) => editNested("canvas","problema",e.target.value)} /></label><label className="workspace-card edit-card"><span className="technical-label">SEGMENTOS</span><textarea rows="5" value={csv(c.segmentos)} onChange={(e) => editNested("canvas","segmentos",list(e.target.value))} /></label><label className="workspace-card edit-card"><span className="technical-label">PROPOSTA DE VALOR</span><textarea rows="5" value={c.proposta_valor || ""} onChange={(e) => editNested("canvas","proposta_valor",e.target.value)} /></label><label className="workspace-card edit-card"><span className="technical-label">SOLUÇÃO</span><textarea rows="5" value={c.solucao || ""} onChange={(e) => editNested("canvas","solucao",e.target.value)} /></label><label className="workspace-card edit-card"><span className="technical-label">CANAIS</span><textarea rows="5" value={csv(c.canais)} onChange={(e) => editNested("canvas","canais",list(e.target.value))} /></label><label className="workspace-card edit-card"><span className="technical-label">RECEITAS</span><textarea rows="5" value={c.receita_modelo || ""} onChange={(e) => editNested("canvas","receita_modelo",e.target.value)} /></label><label className="workspace-card edit-card"><span className="technical-label">ESTRUTURA DE CUSTOS</span><textarea rows="5" value={c.custos_notas || ""} onChange={(e) => editNested("canvas","custos_notas",e.target.value)} /></label><label className="workspace-card edit-card"><span className="technical-label">MÉTRICAS</span><textarea rows="5" value={csv(c.metricas)} onChange={(e) => editNested("canvas","metricas",list(e.target.value))} /></label><label className="workspace-card edit-card"><span className="technical-label">VANTAGEM</span><textarea rows="5" value={c.vantagem || ""} onChange={(e) => editNested("canvas","vantagem",e.target.value)} /></label></div></>;
  }

  function renderDigital() {
    if (!company) return <RequireCompany onCreate={() => setDialog("company")} />;
    return <><div className="workspace-section-head"><div><p className="eyebrow">PRESENÇA DIGITAL</p><h2>Canais e ferramentas</h2></div><button className="primary-button" type="button" onClick={saveDraft} disabled={!dirty || busy}>SALVAR SELEÇÃO</button></div><Checklist title="Canais digitais" items={core.channels} selected={company.canais_digitais || []} onToggle={(id) => toggleList("canais_digitais", id)} /><Checklist title="Ferramentas digitais" items={core.tools} selected={company.ferramentas_digitais || []} onToggle={(id) => toggleList("ferramentas_digitais", id)} /></>;
  }

  function renderDecisoes() {
    if (!company) return <RequireCompany onCreate={() => setDialog("company")} />;
    const finished = Number(company.duracao_semanas || 0) > 0 && Number(company.semana || 0) >= Number(company.duracao_semanas || 0);
    return <><div className="workspace-section-head"><div><p className="eyebrow">DECISÕES</p><h2>Semana {Number(company.semana || 0) + 1}</h2></div><span className="section-count">{finished ? "SIMULAÇÃO CONCLUÍDA" : "PRONTA PARA PROCESSAR"}</span></div><div className="workspace-two-col"><article className="workspace-card"><div className="decision-grid"><label>PREÇO<input type="number" min="0" step="0.01" value={company.preco ?? 0} onChange={(e) => edit("preco", Number(e.target.value))} /></label><label>MARKETING SEMANAL<input type="number" min="0" step="1" value={company.marketing_semanal ?? 0} onChange={(e) => edit("marketing_semanal", Number(e.target.value))} /></label><label>DESCONTO PROMOCIONAL (%)<input type="number" min="0" max="35" step="1" value={company.promocao_desconto ?? 0} onChange={(e) => edit("promocao_desconto", Number(e.target.value))} /></label><label>FUNCIONÁRIOS<input type="number" min="0" step="1" value={company.funcionarios ?? 0} onChange={(e) => edit("funcionarios", Math.max(0, Math.trunc(Number(e.target.value))))} /></label><label>SALÁRIO MÉDIO<input type="number" min="0" step="10" value={company.salario_medio ?? 0} onChange={(e) => edit("salario_medio", Number(e.target.value))} /></label><label>LOCALIZAÇÃO<select value={company.qualidade_localizacao || "media"} onChange={(e) => edit("qualidade_localizacao", e.target.value)}><option value="baixa">Baixa</option><option value="media">Média</option><option value="alta">Alta</option></select></label><label>OPERAÇÃO<select value={company.operacao || "hibrida"} onChange={(e) => edit("operacao", e.target.value)}><option value="fisica">Física</option><option value="digital">Digital</option><option value="hibrida">Híbrida</option></select></label><label className="toggle-row"><input type="checkbox" checked={Boolean(company.delivery)} onChange={(e) => edit("delivery", e.target.checked)} /><span>Usar delivery</span></label></div></article><article className="workspace-card process-card"><p className="eyebrow">MOTOR GO</p><h2>Processar semana</h2><p className="student-muted">A rodada é calculada pelo motor de simulação WebAssembly e sincronizada automaticamente com o servidor Oracle.</p><dl className="student-data-list"><DataRow label="Cenário" value={company.cenario} /><DataRow label="Concorrência" value={number.format(Number(company.concorrencia_indice || 1))} /><DataRow label="Caixa antes" value={money.format(Number(company.caixa || 0))} /><DataRow label="Estoque" value={money.format(Number(company.estoque_valor || 0))} /></dl><div className="process-actions"><button className="primary-button" type="button" onClick={processWeek} disabled={busy || core.state !== "ready" || finished}>{busy ? "PROCESSANDO…" : finished ? "SIMULAÇÃO CONCLUÍDA" : "PROCESSAR E SINCRONIZAR"}</button><button className="secondary-button" type="button" onClick={saveDraft} disabled={!dirty || busy}>SALVAR SEM PROCESSAR</button></div></article></div></>;
  }

  function renderInsumos() {
    if (!company) return <RequireCompany onCreate={() => setDialog("company")} />;
    const suppliers = catalogs?.supplies?.fornecedores || [];
    if (company.usa_insumos && company.insumos?.length) return <><div className="workspace-section-head"><div><p className="eyebrow">INSUMOS</p><h2>Estoque e pedidos</h2></div><span className="section-count">{money.format(Number(company.estoque_valor || 0))}</span></div><div className="table-wrap"><table className="workspace-table"><thead><tr><th>Insumo</th><th>Quantidade</th><th>Unidade</th><th>Custo médio</th><th>Consumo/venda</th><th>Crítico</th></tr></thead><tbody>{company.insumos.map((item) => <tr key={item.id}><td><strong>{item.nome}</strong></td><td>{number.format(Number(item.quantidade || 0))}</td><td>{item.unidade}</td><td>{money.format(Number(item.custo_medio || 0))}</td><td>{number.format(Number(item.consumo_por_venda || 0))}</td><td>{item.critico ? "SIM" : "NÃO"}</td></tr>)}</tbody></table></div><form className="workspace-card order-form" onSubmit={placeOrder}><p className="eyebrow">NOVO PEDIDO</p><div className="orbit-form-four"><label>INSUMO<select value={order.input_id || company.insumos[0]?.id || ""} onChange={(e) => setOrder({ ...order, input_id: e.target.value })}>{company.insumos.map((item) => <option key={item.id} value={item.id}>{item.nome}</option>)}</select></label><label>QUANTIDADE<input type="number" min="0.01" step="0.01" value={order.quantity} onChange={(e) => setOrder({ ...order, quantity: e.target.value })} /></label><label>FORNECEDOR<select value={order.supplier_id} onChange={(e) => setOrder({ ...order, supplier_id: e.target.value })}>{suppliers.map((item) => <option key={item.id} value={item.id}>{item.nome}</option>)}</select></label><label>PAGAMENTO<select value={order.term} onChange={(e) => setOrder({ ...order, term: Number(e.target.value) })}><option value="0">À vista</option><option value="1">1 semana</option><option value="2">2 semanas</option></select></label></div><button className="primary-button" disabled={busy}>REGISTRAR PEDIDO</button></form>{company.pedidos_insumos?.length ? <div className="table-wrap"><table className="workspace-table"><thead><tr><th>Pedido em trânsito</th><th>Quantidade</th><th>Fornecedor</th><th>Entrega</th></tr></thead><tbody>{company.pedidos_insumos.map((item, index) => <tr key={`${item.insumo_id}-${index}`}><td>{item.insumo_nome}</td><td>{number.format(Number(item.quantidade || 0))}</td><td>{item.fornecedor}</td><td>Semana {item.semana_entrega}</td></tr>)}</tbody></table></div> : null}</>;
    if (company.usa_estoque) return <><div className="workspace-section-head"><div><p className="eyebrow">ESTOQUE</p><h2>Mercadoria</h2></div><span className="section-count">{integer.format(Number(company.estoque_unidades || 0))} UN.</span></div><form className="workspace-card order-form" onSubmit={buyStock}><div className="orbit-form-two"><label>QUANTIDADE<input type="number" min="1" step="1" value={stockOrder.quantity} onChange={(e) => setStockOrder({ ...stockOrder, quantity: e.target.value })} /></label><label>PAGAMENTO<select value={stockOrder.term} onChange={(e) => setStockOrder({ ...stockOrder, term: Number(e.target.value) })}><option value="0">À vista</option><option value="1">1 semana</option><option value="2">2 semanas</option></select></label></div><button className="primary-button" disabled={busy}>COMPRAR ESTOQUE</button></form></>;
    return <section className="workspace-card"><p className="eyebrow">INSUMOS</p><h2>Este modelo não utiliza estoque detalhado.</h2><p className="student-muted">Os custos unitários são tratados diretamente pelo motor da simulação.</p></section>;
  }

  function renderFinanceiro() {
    if (!company) return <RequireCompany onCreate={() => setDialog("company")} />;
    const receivable = (company.contas_receber || []).reduce((sum,item) => sum + Number(item.valor || 0),0);
    const payable = (company.contas_pagar || []).reduce((sum,item) => sum + Number(item.valor || 0),0);
    return <><div className="student-metrics-grid"><Metric label="Caixa" value={money.format(Number(company.caixa || 0))} /><Metric label="A receber" value={money.format(receivable)} /><Metric label="A pagar" value={money.format(payable)} /><Metric label="Dívida" value={money.format(Number(company.divida || 0))} /><Metric label="Resultado acumulado" value={indicators ? money.format(Number(indicators.Resultado || 0)) : "…"} /><Metric label="Receita acumulada" value={indicators ? money.format(Number(indicators.Receita || 0)) : "…"} /></div><div className="workspace-two-col"><article><div className="workspace-section-head"><div><p className="eyebrow">CONTAS A RECEBER</p></div></div><div className="table-wrap"><table className="workspace-table"><thead><tr><th>Semana</th><th>Descrição</th><th>Valor</th></tr></thead><tbody>{company.contas_receber?.length ? company.contas_receber.map((item,index) => <tr key={index}><td>{item.semana}</td><td>{item.descricao}</td><td>{money.format(Number(item.valor || 0))}</td></tr>) : <tr><td colSpan="3" className="empty-cell">Sem contas a receber.</td></tr>}</tbody></table></div></article><article><div className="workspace-section-head"><div><p className="eyebrow">CONTAS A PAGAR</p></div></div><div className="table-wrap"><table className="workspace-table"><thead><tr><th>Semana</th><th>Descrição</th><th>Valor</th></tr></thead><tbody>{company.contas_pagar?.length ? company.contas_pagar.map((item,index) => <tr key={index}><td>{item.semana}</td><td>{item.descricao}</td><td>{money.format(Number(item.valor || 0))}</td></tr>) : <tr><td colSpan="3" className="empty-cell">Sem contas a pagar.</td></tr>}</tbody></table></div></article></div></>;
  }

  function renderIndicadores() {
    if (!company) return <RequireCompany onCreate={() => setDialog("company")} />;
    return <><div className="student-metrics-grid"><Metric label="Score total" value={score ? number.format(Number(score.Total || 0)) : "…"} /><Metric label="Financeiro" value={score ? number.format(Number(score.Financeiro || 0)) : "…"} /><Metric label="Mercado" value={score ? number.format(Number(score.Mercado || 0)) : "…"} /><Metric label="Operação" value={score ? number.format(Number(score.Operacao || 0)) : "…"} /><Metric label="Hipóteses" value={score ? number.format(Number(score.Hipoteses || 0)) : "…"} /><Metric label="Gestão" value={score ? number.format(Number(score.Gestao || 0)) : "…"} /><Metric label="Conversão" value={indicators ? `${number.format(Number(indicators.ConversaoAcumuladaPct || 0))}%` : "…"} /><Metric label="Ticket médio" value={indicators ? money.format(Number(indicators.TicketMedio || 0)) : "…"} /></div>{score?.Observacoes?.length ? <div className="workspace-card"><p className="eyebrow">LEITURA DO SCORE</p><ul className="plain-list">{score.Observacoes.map((item,index) => <li key={index}>{item}</li>)}</ul></div> : null}<div className="table-wrap"><table className="workspace-table"><thead><tr><th>Semana</th><th>Vendas</th><th>Receita</th><th>Resultado</th><th>Caixa</th><th>Conversão</th><th>Evento</th></tr></thead><tbody>{company.historico?.length ? [...company.historico].reverse().map((item) => <tr key={item.semana}><td>{item.semana}</td><td>{item.vendas}</td><td>{money.format(Number(item.receita || 0))}</td><td>{money.format(Number(item.resultado || 0))}</td><td>{money.format(Number(item.caixa || 0))}</td><td>{number.format(Number(item.conversao_observada_pct || 0))}%</td><td>{item.evento || "—"}</td></tr>) : <tr><td colSpan="7" className="empty-cell">Processe a primeira semana para formar o histórico.</td></tr>}</tbody></table></div></>;
  }

  function renderJornada() {
    if (!company) return <RequireCompany onCreate={() => setDialog("company")} />;
    const step = Math.max(1, Math.min(4, Number(core.journey || 1)));
    const steps = [
      [1,"Sua ideia de negócio","Defina a empresa, Persona e Lean Canvas."],
      [2,"Seu negócio na internet","Escolha canais digitais coerentes com a Persona."],
      [3,"Venda mais na internet","Teste preço, marketing, promoção e acompanhe conversão."],
      [4,"Ferramentas de apoio","Use ferramentas digitais e indicadores para revisar hipóteses."],
    ];
    return <><section className="journey-head"><p className="eyebrow">JORNADA ORFEU</p><h2>Passo atual: {step} de 4</h2><p className="student-muted">A Jornada organiza o percurso. Persona e Lean Canvas continuam editáveis durante toda a simulação.</p></section><div className="journey-grid">{steps.map(([id,title,description]) => <button type="button" key={id} className={id === step ? "journey-card active" : id < step ? "journey-card done" : "journey-card"} onClick={() => activate(id === 1 ? "persona" : id === 2 ? "digital" : id === 3 ? "decisoes" : "indicadores")}><span>PASSO {id}</span><strong>{title}</strong><small>{description}</small></button>)}</div></>;
  }

  function renderTurmas() {
    return <><div className="workspace-section-head"><div><p className="eyebrow">TURMA</p><h2>Vínculo administrativo</h2></div><span className="section-count">{classes.length ? "ATIVO" : "AGUARDANDO"}</span></div><p className="section-help">A turma é definida pelo Administrador. Não é necessário inserir código nem aceitar convite.</p><div className="class-grid">{classes.length ? classes.map((cl) => <article className="workspace-card" key={cl.id}><p className="eyebrow">T{cl.number || "—"}</p><h3>{cl.name}</h3><dl className="student-data-list"><DataRow label="Matrícula" value={cl.id === user.current_class_id ? user.enrollment_id || "—" : "Histórico"} /><DataRow label="Cenário" value={cl.scenario?.nome || "Mercado estável"} /><DataRow label="Duração" value={`${cl.scenario?.duracao || 12} semanas`} /><DataRow label="Dificuldade" value={cl.scenario?.dificuldade || "intermediario"} /></dl></article>) : <article className="workspace-card"><p className="eyebrow">AGUARDANDO TURMA</p><h3>Seu cadastro ainda não foi vinculado a uma turma.</h3><p className="student-muted">O Administrador fará o vínculo e o Orfeu gerará automaticamente sua matrícula.</p></article>}</div></>;
  }

  const view = activeView === "empresa" ? renderEmpresa() : activeView === "persona" ? renderPersona() : activeView === "canvas" ? renderCanvas() : activeView === "digital" ? renderDigital() : activeView === "decisoes" ? renderDecisoes() : activeView === "insumos" ? renderInsumos() : activeView === "financeiro" ? renderFinanceiro() : activeView === "indicadores" ? renderIndicadores() : activeView === "jornada" ? renderJornada() : renderTurmas();

  return <>
    {error ? <div className="workspace-alert workspace-alert-error">ERRO · {error}</div> : null}
    {notice ? <div className="workspace-alert workspace-alert-success">OK · {notice}</div> : null}
    <section id={activeView} className="module-view">{view}</section>

    <Modal open={dialog === "delete-company"} title="EXCLUIR EMPREENDIMENTO" subtitle={selected?.company?.nome || "Empresa selecionada"} onClose={() => { if (!busy) { setDialog(""); setDeletePhrase(""); } }}>
      <form className="orbit-form" onSubmit={deleteSelectedCompany}>
        <div className="destructive-warning"><strong>EXCLUSÃO DEFINITIVA</strong><span>Você perderá o histórico das semanas simuladas, as decisões, os indicadores e eventual avaliação do Mentor desta empresa. As demais empresas serão preservadas.</span></div>
        <label>DIGITE EXCLUIR PARA CONFIRMAR<input value={deletePhrase} autoComplete="off" onChange={(e) => setDeletePhrase(e.target.value)} /></label>
        {error ? <div className="form-error">{error}</div> : null}
        <div className="modal-actions"><button className="secondary-button" type="button" disabled={busy} onClick={() => { setDialog(""); setDeletePhrase(""); }}>CANCELAR</button><button className="danger-button" disabled={busy || deletePhrase.trim().toUpperCase() !== "EXCLUIR"}>{busy ? "EXCLUINDO…" : "EXCLUIR DEFINITIVAMENTE"}</button></div>
      </form>
    </Modal>

    <Modal open={dialog === "company"} title="NOVO EMPREENDIMENTO" subtitle="Catálogo oficial: Setor → Tipo → Especialidade → dados iniciais." onClose={() => setDialog("")}>
      <form className="orbit-form" onSubmit={createCompany}>
        <div className="wizard-steps"><span>1 SETOR</span><span>2 TIPO</span><span>3 ESPECIALIDADE</span><span>4 EMPRESA</span></div>
        <label>SETOR<select value={newCompany.sector} onChange={(e) => { const sector = e.target.value; const nextTypes = typeOptions(catalogs?.business, sector); const nextModels = nextTypes[0]?.especialidades || []; setNewCompany({ ...newCompany, sector, type: nextTypes[0]?.id || "", model: nextModels[0]?.id || "" }); }}>{sectors.map((item) => <option key={item.id} value={item.id}>{item.nome}</option>)}</select></label>
        <label>TIPO DE NEGÓCIO<select value={newCompany.type} onChange={(e) => { const type = e.target.value; const nextModels = specialtyOptions(catalogs?.business, newCompany.sector, type); setNewCompany({ ...newCompany, type, model: nextModels[0]?.id || "" }); }}>{types.map((item) => <option key={item.id} value={item.id}>{item.nome}</option>)}</select></label>
        <label>ESPECIALIDADE<select value={newCompany.model} onChange={(e) => setNewCompany({ ...newCompany, model: e.target.value })}>{models.map((item) => <option key={item.id} value={item.id}>{item.nome}</option>)}</select></label>
        {currentModel ? <div className="catalog-preview"><div><span>PREÇO REF.</span><strong>{money.format(Number(currentModel.preco_ref || 0))}</strong></div><div><span>CAPACIDADE</span><strong>{integer.format(Number(currentModel.capacidade_base || 0))}/sem.</strong></div><div><span>CUSTO UNIT.</span><strong>{money.format(Number(currentModel.custo_unitario || 0))}</strong></div></div> : null}
        <div className="orbit-form-two"><label>NOME DA EMPRESA<input value={newCompany.name} onChange={(e) => setNewCompany({ ...newCompany, name: e.target.value })} placeholder={currentModel?.nome || "Minha empresa"} required /></label><label>CAPITAL PRÓPRIO<input type="number" min="0" step="100" value={newCompany.capital} onChange={(e) => setNewCompany({ ...newCompany, capital: e.target.value })} required /></label></div>
        <label>DIFICULDADE<select value={newCompany.difficulty} onChange={(e) => setNewCompany({ ...newCompany, difficulty: e.target.value })}><option value="iniciante">Iniciante</option><option value="intermediario">Intermediário</option><option value="avancado">Avançado</option></select></label>
        {classes[0] ? <div className="activation-note"><strong>TURMA / MATRÍCULA</strong><span>T{classes[0].number} · {classes[0].name} · {user.enrollment_id || "matrícula administrativa"}. A empresa usará o cenário definido pela turma.</span></div> : <><div className="activation-note"><strong>AGUARDANDO TURMA</strong><span>O Administrador ainda não vinculou sua conta a uma turma. A empresa poderá ser criada sem vínculo e associada posteriormente.</span></div><label>CENÁRIO<select value={newCompany.scenario_id} onChange={(e) => setNewCompany({ ...newCompany, scenario_id: e.target.value })}>{baseScenarios.map((item) => <option key={item.id} value={item.id}>{item.nome}</option>)}</select></label></>}
        {error ? <div className="form-error">{error}</div> : null}
        <div className="modal-actions"><button type="button" className="secondary-button" onClick={() => setDialog("")}>CANCELAR</button><button className="primary-button" disabled={busy || !newCompany.model}>{busy ? "CRIANDO…" : "CRIAR EMPRESA"}</button></div>
      </form>
    </Modal>
  </>;
}
