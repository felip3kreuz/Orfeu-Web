"use client";

import { useEffect, useMemo, useState } from "react";
import { createJEDClient } from "@/lib/jed-core";
import Modal from "@/components/modal";

const money = new Intl.NumberFormat("pt-BR", { style: "currency", currency: "BRL" });
const number = new Intl.NumberFormat("pt-BR", { maximumFractionDigits: 2 });
function fmtDate(value) { if (!value) return "—"; const d = new Date(value); return Number.isNaN(d.getTime()) ? value : new Intl.DateTimeFormat("pt-BR", { dateStyle: "short", timeStyle: "short" }).format(d); }

function DataRow({ label, value }) { return <div><dt>{label}</dt><dd>{value ?? "—"}</dd></div>; }

export default function MentorWorkspace({ user }) {
  const [activeView, setActiveView] = useState("visao-geral");
  const [data, setData] = useState({ classes: [], companies: [], students: [], scenarios: [] });
  const [scores, setScores] = useState({});
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const [dialog, setDialog] = useState("");
  const [scenarioForm, setScenarioForm] = useState({ nome: "", alcance: 1, conversao: 1, oscilacao: 0.08, evento_negativo_extra: 0, duracao: 12, concorrencia_nivel: "media", concorrencia_indice: 1, dificuldade: "intermediario", observacoes: "" });
  const [classScenario, setClassScenario] = useState({ class_id: "", class_name: "", scenario_id: "" });
  const [evaluationForm, setEvaluationForm] = useState({ company_id: "", company_name: "", status: "aprovado", comment: "" });
  const [resultClassFilter, setResultClassFilter] = useState("all");
  const [resultStatusFilter, setResultStatusFilter] = useState("all");
  const [classStatusFilter, setClassStatusFilter] = useState("all");

  useEffect(() => {
    const onNavigate = (event) => {
      if (event?.detail?.role !== "mentor") return;
      const view = String(event.detail.view || "visao-geral");
      setActiveView(view);
      window.dispatchEvent(new CustomEvent("jed:view", { detail: { role: "mentor", view } }));
    };
    window.addEventListener("jed:navigate", onNavigate);
    return () => window.removeEventListener("jed:navigate", onNavigate);
  }, []);

  async function load() {
    setLoading(true); setError("");
    try {
      const [overviewResponse, scenariosResponse] = await Promise.all([
        fetch("/api/mentor/overview", { cache: "no-store" }),
        fetch("/api/mentor/scenarios", { cache: "no-store" }),
      ]);
      const overview = await overviewResponse.json();
      const scenarioPayload = await scenariosResponse.json();
      if (!overviewResponse.ok) throw new Error(overview?.error || "Falha ao carregar painel.");
      if (!scenariosResponse.ok) throw new Error(scenarioPayload?.error || "Falha ao carregar cenários.");
      const next = {
        classes: Array.isArray(overview.classes) ? overview.classes : [],
        companies: Array.isArray(overview.companies) ? overview.companies : [],
        students: Array.isArray(overview.students) ? overview.students : [],
        scenarios: Array.isArray(scenarioPayload.scenarios) ? scenarioPayload.scenarios : [],
      };
      setData(next);
      let client;
      try {
        client = await createJEDClient(1);
        const mapped = {};
        for (const remote of next.companies) mapped[remote.id] = client.score(remote.company || {}).Total || 0;
        setScores(mapped);
      } catch { setScores({}); }
      finally { try { client?.close(); } catch {} }
    } catch (caught) { setError(caught instanceof Error ? caught.message : "Falha ao carregar painel."); }
    finally { setLoading(false); }
  }
  useEffect(() => { void load(); }, []);

  const companiesByClass = useMemo(() => {
    const map = {};
    for (const remote of data.companies) (map[remote.class_id || "sem-turma"] ||= []).push(remote);
    return map;
  }, [data.companies]);

  function evaluationStatus(remote) {
    if (remote?.approval_status === "aprovado") return "aprovado";
    if (remote?.approval_status === "reprovado") return "reprovado";
    return "pendente";
  }

  const filteredCompanies = useMemo(() => data.companies.filter((remote) => {
    const classMatches = resultClassFilter === "all" || remote.class_id === resultClassFilter;
    const statusMatches = resultStatusFilter === "all" || evaluationStatus(remote) === resultStatusFilter;
    return classMatches && statusMatches;
  }), [data.companies, resultClassFilter, resultStatusFilter]);

  const resultCounts = useMemo(() => {
    const scoped = resultClassFilter === "all" ? data.companies : data.companies.filter((remote) => remote.class_id === resultClassFilter);
    return {
      aprovado: scoped.filter((remote) => evaluationStatus(remote) === "aprovado").length,
      pendente: scoped.filter((remote) => evaluationStatus(remote) === "pendente").length,
      reprovado: scoped.filter((remote) => evaluationStatus(remote) === "reprovado").length,
    };
  }, [data.companies, resultClassFilter]);

  const filteredClasses = useMemo(() => {
    if (classStatusFilter === "all") return data.classes;
    return data.classes.filter((cl) => (companiesByClass[cl.id] || []).some((remote) => evaluationStatus(remote) === classStatusFilter));
  }, [data.classes, companiesByClass, classStatusFilter]);

  function classesForStudent(id) { return data.classes.filter((cl) => cl.student_ids?.includes(id)).map((cl) => cl.name).join(", ") || "—"; }
  function activate(view) { setActiveView(view); window.dispatchEvent(new CustomEvent("jed:view", { detail: { role: "mentor", view } })); }

  async function createScenario(event) {
    event.preventDefault(); setBusy(true); setError(""); setNotice("");
    try {
      const response = await fetch("/api/mentor/scenarios", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(scenarioForm) });
      const payload = await response.json();
      if (!response.ok) throw new Error(payload?.error || "Falha ao criar cenário.");
      setScenarioForm({ nome: "", alcance: 1, conversao: 1, oscilacao: 0.08, evento_negativo_extra: 0, duracao: 12, concorrencia_nivel: "media", concorrencia_indice: 1, dificuldade: "intermediario", observacoes: "" });
      setDialog(""); setNotice(`Cenário ${payload.scenario?.scenario?.nome || "criado"} salvo no servidor.`); await load(); activate("cenarios");
    } catch (caught) { setError(caught instanceof Error ? caught.message : "Falha ao criar cenário."); }
    finally { setBusy(false); }
  }

  function openClassScenario(cl) {
    const current = data.scenarios.find((entry) => entry.scenario?.nome === cl.scenario?.nome) || data.scenarios[0];
    setClassScenario({ class_id: cl.id, class_name: cl.name, scenario_id: current?.id || "" });
    setError(""); setNotice(""); setDialog("class-scenario");
  }

  async function saveClassScenario(event) {
    event.preventDefault();
    const entry = data.scenarios.find((item) => item.id === classScenario.scenario_id);
    if (!entry) return;
    setBusy(true); setError(""); setNotice("");
    try {
      const response = await fetch("/api/mentor/classes", { method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ class_id: classScenario.class_id, scenario: entry.scenario }) });
      const payload = await response.json();
      if (!response.ok) throw new Error(payload?.error || "Falha ao atualizar cenário da turma.");
      setDialog(""); setNotice(`${classScenario.class_name}: cenário atualizado para ${entry.scenario?.nome || "cenário selecionado"}.`); await load(); activate("turmas");
    } catch (caught) { setError(caught instanceof Error ? caught.message : "Falha ao atualizar cenário da turma."); }
    finally { setBusy(false); }
  }

  function openEvaluation(remote) {
    setEvaluationForm({
      company_id: remote.id,
      company_name: remote.company?.nome || "Empresa",
      status: remote.approval_status === "reprovado" ? "reprovado" : "aprovado",
      comment: remote.mentor_comment || "",
    });
    setError(""); setNotice(""); setDialog("evaluation");
  }

  async function saveEvaluation(event) {
    event.preventDefault();
    setBusy(true); setError(""); setNotice("");
    try {
      const response = await fetch("/api/mentor/companies/evaluate", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ company_id: evaluationForm.company_id, status: evaluationForm.status, comment: evaluationForm.comment }),
      });
      const payload = await response.json();
      if (!response.ok) throw new Error(payload?.error || "Falha ao salvar avaliação.");
      const updated = payload.company;
      setData((current) => ({ ...current, companies: current.companies.map((item) => item.id === updated.id ? updated : item) }));
      setDialog("");
      setNotice(`${evaluationForm.company_name}: ${evaluationForm.status.toUpperCase()}. Parecer salvo.`);
    } catch (caught) { setError(caught instanceof Error ? caught.message : "Falha ao salvar avaliação."); }
    finally { setBusy(false); }
  }

  if (loading) return <section className="workspace-loading">CARREGANDO TURMAS, CENÁRIOS E EMPRESAS…</section>;

  function renderOverview() {
    return <><section className="orbit-overview-strip"><div><span>SESSÃO</span><strong>MENTOR / {user.name}</strong><small>{user.institution || "Instituição não informada"} · Orfeu Online</small></div><div className="orbit-stat-row"><div><span>TURMAS</span><strong>{data.classes.length}</strong></div><div><span>EMPRESAS</span><strong>{data.companies.length}</strong></div><div><span>ALUNOS</span><strong>{data.students.length}</strong></div><div><span>CENÁRIOS</span><strong>{data.scenarios.length}</strong></div></div></section><div className="orbit-toolbar"><button className="primary-button" onClick={() => setDialog("scenario")}>NOVO CENÁRIO</button><button className="secondary-button" onClick={load}>ATUALIZAR</button></div><div className="dashboard-module-grid"><button className="module-tile" onClick={() => activate("cenarios")}><span>01</span><strong>CENÁRIOS</strong><small>{data.scenarios.length} ambientes disponíveis</small></button><button className="module-tile" onClick={() => activate("turmas")}><span>02</span><strong>TURMAS</strong><small>turmas atribuídas pelo Administrador</small></button><button className="module-tile" onClick={() => activate("alunos")}><span>03</span><strong>ALUNOS</strong><small>contas cadastradas pelo Administrador</small></button><button className="module-tile" onClick={() => activate("resultados")}><span>04</span><strong>RESULTADOS</strong><small>empresas sincronizadas</small></button></div></>;
  }
  function renderScenarios() {
    return <><div className="workspace-section-head"><div><p className="eyebrow">CENÁRIOS</p><h2>Ambientes pedagógicos</h2></div><button className="primary-button" onClick={() => setDialog("scenario")}>NOVO CENÁRIO</button></div><p className="section-help">Cenários personalizados ficam no servidor e podem ser reutilizados em novas turmas.</p><div className="scenario-grid">{data.scenarios.map((entry) => { const sc = entry.scenario || {}; return <article className="workspace-card scenario-card" key={entry.id}><div className="scenario-title"><div><span className="technical-label">{entry.builtin ? "BASE ORFEU" : "PERSONALIZADO"}</span><h3>{sc.nome}</h3></div><span className={entry.builtin ? "state-label used" : "state-label active"}>{entry.builtin ? "PADRÃO" : "SALVO"}</span></div><dl className="student-data-list"><DataRow label="Duração" value={`${sc.duracao || 12} semanas`} /><DataRow label="Dificuldade" value={sc.dificuldade || "intermediario"} /><DataRow label="Alcance" value={number.format(Number(sc.alcance || 1))} /><DataRow label="Conversão" value={number.format(Number(sc.conversao || 1))} /><DataRow label="Oscilação" value={number.format(Number(sc.oscilacao || 0))} /><DataRow label="Concorrência" value={sc.concorrencia_nivel || "media"} /></dl>{sc.observacoes ? <p className="student-muted">{sc.observacoes}</p> : null}</article>; })}</div></>;
  }
  function renderClasses() {
    return <><div className="workspace-section-head"><div><p className="eyebrow">TURMAS</p><h2>Turmas sob sua responsabilidade</h2></div><button className="secondary-button" onClick={load}>ATUALIZAR</button></div><p className="section-help">As turmas são criadas e nomeadas pelo Administrador. O Mentor acompanha cenário, alunos, empresas e resultados.</p><div className="mentor-filter-bar"><label>FILTRAR TURMAS POR AVALIAÇÃO<select value={classStatusFilter} onChange={(e) => setClassStatusFilter(e.target.value)}><option value="all">Todas</option><option value="aprovado">Com APROVADOS</option><option value="pendente">Com PENDENTES</option><option value="reprovado">Com REPROVADOS</option></select></label></div><div className="class-grid">{filteredClasses.length ? filteredClasses.map((cl) => { const list = companiesByClass[cl.id] || []; const approved = list.filter((remote) => evaluationStatus(remote) === "aprovado").length; const pending = list.filter((remote) => evaluationStatus(remote) === "pendente").length; const rejected = list.filter((remote) => evaluationStatus(remote) === "reprovado").length; return <article className="workspace-card orbit-class-card" key={cl.id}><div className="class-title"><div><span className="technical-label">T{cl.number || "—"}</span><h3>{cl.name}</h3></div><strong>{cl.student_ids?.length || 0} ALUNOS</strong></div><dl className="student-data-list"><DataRow label="Cenário" value={cl.scenario?.nome || "Mercado estável"} /><DataRow label="Duração" value={`${cl.scenario?.duracao || 12} semanas`} /><DataRow label="Dificuldade" value={cl.scenario?.dificuldade || "intermediario"} /><DataRow label="Empresas sincronizadas" value={list.length} /></dl><div className="class-evaluation-summary"><span className="state-label approved">{approved} APROVADO(S)</span><span className="state-label pending">{pending} PENDENTE(S)</span><span className="state-label rejected">{rejected} REPROVADO(S)</span></div><button type="button" className="secondary-button compact-button" onClick={() => openClassScenario(cl)}>ALTERAR CENÁRIO</button></article>; }) : <article className="workspace-card"><p className="student-muted">Nenhuma turma corresponde ao filtro selecionado ou foi atribuída a este Mentor.</p></article>}</div></>;
  }
  function renderStudents() {
    return <><div className="workspace-section-head"><div><p className="eyebrow">ALUNOS</p><h2>Alunos vinculados às suas turmas</h2></div><button className="secondary-button" onClick={load}>ATUALIZAR</button></div><p className="section-help">O Administrador cadastra o Aluno e define sua turma. A matrícula é gerada automaticamente e acompanha o vínculo atual.</p><div className="table-wrap"><table className="workspace-table"><thead><tr><th>Aluno</th><th>E-mail</th><th>Matrícula</th><th>Turma</th><th>ID institucional</th><th>Status</th></tr></thead><tbody>{data.students.length ? data.students.map((student) => <tr key={student.id}><td><strong>{student.name}</strong></td><td>{student.email}</td><td><strong>{student.enrollment_id || "—"}</strong></td><td>{classesForStudent(student.id)}</td><td>{student.institutional_id || "—"}</td><td>{student.status === "disabled" ? <span className="state-label revoked">DESATIVADO</span> : <span className="state-label active">ATIVO</span>}</td></tr>) : <tr><td colSpan="6" className="empty-cell">Nenhum Aluno vinculado às suas turmas.</td></tr>}</tbody></table></div></>;
  }
  function renderResults() {
    return <><div className="workspace-section-head"><div><p className="eyebrow">RESULTADOS</p><h2>Empresas sincronizadas</h2></div><button className="secondary-button" onClick={load}>ATUALIZAR</button></div><p className="section-help">Filtre por turma e situação da avaliação. PENDENTE identifica empreendimentos ainda não classificados pelo Mentor.</p><div className="mentor-filter-bar mentor-result-filters"><label>TURMA<select value={resultClassFilter} onChange={(e) => setResultClassFilter(e.target.value)}><option value="all">Todas as turmas</option>{data.classes.map((cl) => <option key={cl.id} value={cl.id}>T{cl.number || "—"} · {cl.name}</option>)}</select></label><label>AVALIAÇÃO<select value={resultStatusFilter} onChange={(e) => setResultStatusFilter(e.target.value)}><option value="all">Todas</option><option value="aprovado">APROVADO</option><option value="pendente">PENDENTE</option><option value="reprovado">REPROVADO</option></select></label><div className="filter-counts"><span className="state-label approved">{resultCounts.aprovado} APROVADO(S)</span><span className="state-label pending">{resultCounts.pendente} PENDENTE(S)</span><span className="state-label rejected">{resultCounts.reprovado} REPROVADO(S)</span></div></div><div className="table-wrap"><table className="workspace-table results-table"><thead><tr><th>Turma</th><th>Responsável</th><th>Empresa</th><th>Semana</th><th>Índice Orfeu</th><th>Avaliação</th><th>Parecer</th><th>Ação</th></tr></thead><tbody>{filteredCompanies.length ? [...filteredCompanies].sort((a,b) => Number(scores[b.id] || 0) - Number(scores[a.id] || 0)).map((remote) => { const company = remote.company || {}; const status = evaluationStatus(remote); return <tr key={remote.id}><td>{data.classes.find((cl) => cl.id === remote.class_id)?.name || "Sem turma"}</td><td>{company.responsavel || remote.owner_id}</td><td><strong>{company.nome || "Empresa"}</strong><small>{company.especialidade || company.modelo_base || ""}</small></td><td>{company.semana || 0}/{company.duracao_semanas || "—"}</td><td><strong>{number.format(Number(scores[remote.id] || 0))}</strong></td><td>{status === "aprovado" ? <span className="state-label approved">APROVADO</span> : status === "reprovado" ? <span className="state-label rejected">REPROVADO</span> : <span className="state-label pending">PENDENTE</span>}</td><td className="mentor-comment-cell">{remote.mentor_comment ? <><span>{remote.mentor_comment}</span><small>{remote.evaluated_by_name || "Mentor"} · {fmtDate(remote.evaluated_at)}</small></> : "—"}</td><td><button type="button" className="secondary-button compact-button" onClick={() => openEvaluation(remote)}>{status === "pendente" ? "AVALIAR" : "REAVALIAR"}</button></td></tr>; }) : <tr><td colSpan="8" className="empty-cell">Nenhum resultado corresponde aos filtros selecionados.</td></tr>}</tbody></table></div></>;
  }

  const view = activeView === "visao-geral" ? renderOverview() : activeView === "cenarios" ? renderScenarios() : activeView === "turmas" ? renderClasses() : activeView === "alunos" ? renderStudents() : renderResults();
  return <>{error ? <div className="workspace-alert workspace-alert-error">ERRO · {error}</div> : null}{notice ? <div className="workspace-alert workspace-alert-success">OK · {notice}</div> : null}<section id={activeView} className="module-view">{view}</section>
    <Modal open={dialog === "class-scenario"} title="CENÁRIO DA TURMA" subtitle={classScenario.class_name || "Turma"} onClose={() => setDialog("")}><form className="orbit-form" onSubmit={saveClassScenario}><label>CENÁRIO<select value={classScenario.scenario_id} onChange={(e) => setClassScenario({ ...classScenario, scenario_id: e.target.value })}>{data.scenarios.map((entry) => <option key={entry.id} value={entry.id}>{entry.scenario?.nome || entry.id}{entry.builtin ? " · base" : " · personalizado"}</option>)}</select></label><div className="activation-note"><strong>RESPONSABILIDADE DO MENTOR</strong><span>O Administrador cria e nomeia a turma; o Mentor pode ajustar o cenário pedagógico da turma sob sua responsabilidade.</span></div><div className="modal-actions"><button type="button" className="secondary-button" onClick={() => setDialog("")}>CANCELAR</button><button className="primary-button" disabled={busy || !classScenario.scenario_id}>SALVAR CENÁRIO</button></div></form></Modal>
    <Modal open={dialog === "scenario"} title="NOVO CENÁRIO" subtitle="Crie um ambiente pedagógico reutilizável nas turmas." onClose={() => setDialog("")}><form className="orbit-form" onSubmit={createScenario}><label>NOME<input value={scenarioForm.nome} onChange={(e) => setScenarioForm({ ...scenarioForm, nome: e.target.value })} placeholder="Piloto - Mercado Estável" required /></label><div className="orbit-form-three"><label>DURAÇÃO<input type="number" min="1" value={scenarioForm.duracao} onChange={(e) => setScenarioForm({ ...scenarioForm, duracao: Number(e.target.value) })} /></label><label>DIFICULDADE<select value={scenarioForm.dificuldade} onChange={(e) => setScenarioForm({ ...scenarioForm, dificuldade: e.target.value })}><option value="iniciante">Iniciante</option><option value="intermediario">Intermediário</option><option value="avancado">Avançado</option></select></label><label>CONCORRÊNCIA<select value={scenarioForm.concorrencia_nivel} onChange={(e) => setScenarioForm({ ...scenarioForm, concorrencia_nivel: e.target.value })}><option value="baixa">Baixa</option><option value="media">Média</option><option value="alta">Alta</option></select></label></div><div className="orbit-form-four"><label>ALCANCE<input type="number" min="0.1" step="0.01" value={scenarioForm.alcance} onChange={(e) => setScenarioForm({ ...scenarioForm, alcance: Number(e.target.value) })} /></label><label>CONVERSÃO<input type="number" min="0.1" step="0.01" value={scenarioForm.conversao} onChange={(e) => setScenarioForm({ ...scenarioForm, conversao: Number(e.target.value) })} /></label><label>OSCILAÇÃO<input type="number" min="0" step="0.01" value={scenarioForm.oscilacao} onChange={(e) => setScenarioForm({ ...scenarioForm, oscilacao: Number(e.target.value) })} /></label><label>EVENTO NEG. EXTRA<input type="number" min="-0.5" max="0.5" step="0.01" value={scenarioForm.evento_negativo_extra} onChange={(e) => setScenarioForm({ ...scenarioForm, evento_negativo_extra: Number(e.target.value) })} /></label></div><label>OBSERVAÇÕES<textarea rows="3" value={scenarioForm.observacoes} onChange={(e) => setScenarioForm({ ...scenarioForm, observacoes: e.target.value })} /></label><div className="modal-actions"><button type="button" className="secondary-button" onClick={() => setDialog("")}>CANCELAR</button><button className="primary-button" disabled={busy}>SALVAR CENÁRIO</button></div></form></Modal>
    <Modal open={dialog === "evaluation"} title="AVALIAR EMPREENDIMENTO" subtitle={evaluationForm.company_name || "Empresa do Aluno"} onClose={() => setDialog("")}><form className="orbit-form" onSubmit={saveEvaluation}><div className="evaluation-choice-grid"><label className={evaluationForm.status === "aprovado" ? "evaluation-choice selected" : "evaluation-choice"}><input type="radio" name="evaluation-status" value="aprovado" checked={evaluationForm.status === "aprovado"} onChange={() => setEvaluationForm({ ...evaluationForm, status: "aprovado" })} /><strong>APROVADO</strong><span>O empreendimento atende aos critérios de avaliação.</span></label><label className={evaluationForm.status === "reprovado" ? "evaluation-choice selected" : "evaluation-choice"}><input type="radio" name="evaluation-status" value="reprovado" checked={evaluationForm.status === "reprovado"} onChange={() => setEvaluationForm({ ...evaluationForm, status: "reprovado" })} /><strong>REPROVADO</strong><span>O empreendimento ainda não atende aos critérios de avaliação.</span></label></div><label>COMENTÁRIOS DO MENTOR<textarea rows="6" maxLength="4000" value={evaluationForm.comment} onChange={(e) => setEvaluationForm({ ...evaluationForm, comment: e.target.value })} placeholder="Ex.: O empreendimento foi aprovado, mas com ressalvas quanto à validação do público-alvo..." /></label><small className="field-hint">{evaluationForm.comment.length}/4000 caracteres</small><div className="modal-actions"><button type="button" className="secondary-button" onClick={() => setDialog("")}>CANCELAR</button><button className="primary-button" disabled={busy}>{busy ? "SALVANDO…" : "SALVAR AVALIAÇÃO"}</button></div></form></Modal>
  </>;
}
