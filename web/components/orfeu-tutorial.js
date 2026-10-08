"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { createPortal } from "react-dom";

// Fluxos pedagógicos independentes da API e do motor de simulação.
// Os alvos são elementos reais das páginas W7.3.1; as etapas não executam ações destrutivas.
export const tutorials = {
  aluno: [
    { title: "Seu espaço no Orfeu", target: ".orbit-user-block", text: "Este painel mostra seu perfil. O menu à esquerda dá acesso aos módulos do seu empreendimento.", example: "Exemplo: o Administrador matricula você em uma turma; no Orfeu, você acompanha sua própria empresa." },
    { title: "Criando um empreendimento", view: "empresa", target: "#empresa", text: "Em MINHA EMPRESA, crie ou selecione seu negócio, escolha setor, tipo e especialidade e acompanhe a evolução.", example: "Exemplo: a empresa fictícia 'Doce do Guamá' atua no setor de alimentação e vende doces artesanais." },
    { title: "Definindo a Persona", view: "persona", target: "#persona .edit-card", fallback: "#persona", text: "Registre um cliente plausível, suas necessidades, objetivos e objeções. Você poderá revisar a Persona durante a simulação.", example: "Exemplo: Mariana, 24 anos, compra pelo celular e prioriza rapidez nas entregas." },
    { title: "Preenchendo o Lean Canvas", view: "canvas", target: "#canvas .edit-card", fallback: "#canvas", text: "Descreva os problemas do cliente, sua solução, proposta de valor, canais, métricas, custos e fontes de receita.", example: "Exemplo: problema: pouco tempo para cozinhar; solução: doces prontos com encomenda online." },
    { title: "Canais e ferramentas digitais", view: "digital", target: "#digital .choice-card", fallback: "#digital", text: "Escolha os canais e as ferramentas adequados à Persona. Justifique cada escolha antes de comparar os resultados.", example: "Exemplo: Instagram para divulgação e WhatsApp Business para atendimento." },
    { title: "Tomando decisões semanais", view: "decisoes", target: "#decisoes .decision-grid", fallback: "#decisoes", text: "Defina preço, marketing e promoções. ENCERRAR SEMANA avança um período simulado, não uma semana do calendário real.", example: "Exemplo: investir R$ 50 em marketing e observar os resultados após encerrar a semana." },
    { title: "Finanças e indicadores", view: "financeiro", target: "#financeiro .workspace-card", fallback: "#financeiro", text: "Compare o caixa com as contas a receber, contas a pagar e o resultado do empreendimento.", example: "Exemplo: vendas podem crescer enquanto o caixa diminui por compras ou pagamentos." },
    { title: "Analisando o desempenho", view: "indicadores", target: "#indicadores .workspace-card", fallback: "#indicadores", text: "Use os indicadores para interpretar suas decisões e ajustar hipóteses, em vez de tratar um número isolado como nota final.", example: "Exemplo: mais visitas com poucas vendas pode indicar problema na oferta ou na conversão." },
    { title: "Turma e parecer do mentor", view: "empresa", target: "#empresa .mentor-evaluation-card", fallback: "#empresa", text: "A avaliação pedagógica aparece em MINHA EMPRESA. O Mentor pode aprovar ou reprovar o empreendimento e registrar comentários.", example: "Exemplo: 'Aprovado com ressalvas: revise a justificativa de preço e os canais digitais.'" },
  ],
  mentor: [
    { title: "Seu espaço no Orfeu", target: ".orbit-user-block", text: "O Mentor acompanha alunos e empreendimentos vinculados às suas turmas. A criação das turmas é administrativa.", example: "Exemplo: a turma T25 foi criada pelo Administrador e atribuída ao seu acompanhamento." },
    { title: "Visão geral", view: "visao-geral", target: "#visao-geral", text: "Consulte o resumo de turmas, alunos, empresas e cenários disponíveis para suas atividades.", example: "Exemplo: verificar quantas empresas da turma já foram sincronizadas." },
    { title: "Cenários pedagógicos", view: "cenarios", target: "#cenarios .workspace-section-head", fallback: "#cenarios", text: "Configure os cenários permitidos e selecione ambientes adequados aos objetivos da turma.", example: "Exemplo: usar 'Mercado estável' para a primeira atividade comparativa." },
    { title: "Turmas atribuídas", view: "turmas", target: "#turmas .workspace-section-head", fallback: "#turmas", text: "Confira as turmas criadas pelo Administrador e, quando disponível, ajuste o cenário utilizado.", example: "Exemplo: T25 — Empreendedorismo Digital — 2026." },
    { title: "Acompanhamento dos alunos", view: "alunos", target: "#alunos .workspace-section-head", fallback: "#alunos", text: "Veja os vínculos de matrícula e identifique quem precisa de ajuda para avançar na simulação.", example: "Exemplo: um aluno permanece na semana 0 enquanto seus colegas estão na semana 3." },
    { title: "Resultados sincronizados", view: "resultados", target: "#resultados .workspace-section-head", fallback: "#resultados", text: "Analise as empresas enviadas ao servidor, as semanas concluídas e os indicadores disponíveis.", example: "Exemplo: comparar desempenho após uma mudança de preço ou marketing." },
    { title: "Aprovação e parecer", view: "resultados", target: "#resultados .workspace-section-head", fallback: "#resultados", text: "Na área de resultados, avalie o empreendimento como APROVADO ou REPROVADO e registre um comentário pedagógico, sem modificar a empresa do aluno.", example: "Exemplo: 'Aprovado, com ressalvas quanto à validação da Persona.'" },
  ],
  admin: [
    { title: "Seu espaço no Orfeu", target: ".orbit-user-block", text: "O Administrador gerencia contas, turmas, matrículas e permissões. O tutorial não cria nem altera registros.", example: "Exemplo: cadastrar o Mentor antes de vinculá-lo à turma." },
    { title: "Resumo administrativo", target: "#visao-geral", text: "Veja quantos Administradores, Mentores, Alunos e turmas estão cadastrados.", example: "Exemplo: revisar o total de contas antes de importar uma nova turma." },
    { title: "Cadastro e criação de turmas", target: "[aria-label='Ações administrativas'] .primary-button", fallback: "[aria-label='Ações administrativas']", text: "Utilize os comandos de cadastro de usuários, criação de turmas e importação de arquivos CSV.", example: "Exemplo: cadastrar uma Mentora e criar a turma 'Marketing Digital A'." },
    { title: "Turmas e matrículas", target: "#admin-turmas", text: "As turmas recebem números sequenciais, e a matrícula de cada aluno segue o formato T{turma}A{sequência}.", example: "Exemplo: o quarto aluno da turma T25 recebe a matrícula T25A4." },
    { title: "Contas e vínculos", target: "#usuarios", text: "Cadastre Alunos, Mentores e Administradores; vincule alunos às turmas e acompanhe o primeiro acesso.", example: "Exemplo: um aluno sem turma aparece como AGUARDANDO TURMA até ser vinculado." },
    { title: "Importação de usuários", target: "#importacao", text: "O CSV utiliza as colunas nome, email, papel e, opcionalmente, instituição, ID institucional e turma.", example: "Exemplo: Ana Souza,ana@exemplo.edu.br,aluno,Instituto X,123,T25" },
    { title: "Segurança e e-mail", target: "#usuarios", text: "Verifique o estado das contas, a troca obrigatória de senha e o envio das mensagens de cadastro. Dados reais são administrados fora deste tutorial.", example: "Exemplo: se o SMTP falhar, confira a configuração do servidor antes de reenviar o aviso." },
  ],
};

const roleNames = { aluno: "Aluno", mentor: "Mentor", admin: "Administrador" };
function storageKey(role, user) { return `orfeu:tutorial:o1.0:${role}:${String(user?.id || user?.email || "sessao")}`; }
function readProgress(key) { try { return JSON.parse(window.localStorage.getItem(key) || "null"); } catch { return null; } }
function writeProgress(key, entry) { try { window.localStorage.setItem(key, JSON.stringify(entry)); } catch { /* armazenamento opcional */ } }

function screenRect(element) {
  if (!element) return null;
  const b = element.getBoundingClientRect();
  const x = Math.max(6, Math.min(b.left - 5, innerWidth - 6));
  const y = Math.max(6, Math.min(b.top - 5, innerHeight - 6));
  const right = Math.max(x + 1, Math.min(b.right + 5, innerWidth - 6));
  const bottom = Math.max(y + 1, Math.min(b.bottom + 5, innerHeight - 6));
  return { x, y, width: right - x, height: bottom - y };
}

function overlayStyle(rect) {
  const W = window.innerWidth, H = window.innerHeight;
  if (!rect) return [{ left: 0, top: 0, width: W, height: H }];
  const { x, y, width, height } = rect;
  return [
    { left: 0, top: 0, width: W, height: y },
    { left: 0, top: y, width: x, height },
    { left: x + width, top: y, width: Math.max(0, W - x - width), height },
    { left: 0, top: y + height, width: W, height: Math.max(0, H - y - height) },
  ];
}

function popoverPosition(rect) {
  const W = window.innerWidth, H = window.innerHeight;
  const boxW = Math.min(380, W - 24), boxH = Math.min(450, H - 24);
  if (!rect) return { left: Math.max(12, (W - boxW) / 2), top: Math.max(12, (H - boxH) / 2) };
  let left = rect.x + rect.width + 18;
  if (left + boxW > W - 12) left = rect.x - boxW - 18;
  if (left < 12) left = Math.min(W - boxW - 12, Math.max(12, rect.x));
  const top = Math.max(12, Math.min(H - boxH - 12, rect.y));
  return { left, top };
}

export default function OrfeuTutorial({ role, user }) {
  const steps = tutorials[role] || [];
  const key = useMemo(() => storageKey(role, user), [role, user?.id, user?.email]);
  const [ready, setReady] = useState(false);
  const [welcome, setWelcome] = useState(false);
  const [running, setRunning] = useState(false);
  const [stepIndex, setStepIndex] = useState(0);
  const [rect, setRect] = useState(null);
  const dialogRef = useRef(null);
  const launcherRef = useRef(null);

  useEffect(() => {
    const saved = readProgress(key);
    setStepIndex(Math.min(Math.max(Number(saved?.step) || 0, 0), Math.max(0, steps.length - 1)));
    setWelcome(!saved);
    setRunning(false);
    setReady(true);
  }, [key, steps.length]);

  const persist = useCallback((index, completed = false) => {
    writeProgress(key, { step: index, completed, dismissed: true, version: "O1.0" });
  }, [key]);

  function begin(restart = false) {
    const index = restart ? 0 : stepIndex;
    setWelcome(false);
    setStepIndex(index);
    setRunning(true);
    persist(index);
  }
  function close() { persist(stepIndex); setRunning(false); setWelcome(false); launcherRef.current?.focus(); }
  function move(delta) {
    const next = stepIndex + delta;
    if (next >= steps.length) { persist(steps.length, true); setRunning(false); launcherRef.current?.focus(); return; }
    const index = Math.max(0, next);
    setStepIndex(index);
    persist(index);
  }
  function dismissWelcome() { setWelcome(false); persist(stepIndex); launcherRef.current?.focus(); }

  useEffect(() => {
    if (!running || !steps[stepIndex]) return undefined;
    const current = steps[stepIndex];
    if (current.view && role !== "admin") {
      window.dispatchEvent(new CustomEvent("jed:navigate", { detail: { role, view: current.view } }));
    }
    const targetQuery = () => document.querySelector(current.target) || (current.fallback && document.querySelector(current.fallback)) || document.querySelector(`.orbit-command-rail [data-orfeu-view="${current.view}"]`) || document.querySelector(".orbit-work-area");
    const measure = () => {
      const next = screenRect(targetQuery());
      setRect((previous) => JSON.stringify(previous) === JSON.stringify(next) ? previous : next);
    };
    const timers = [window.setTimeout(() => { targetQuery()?.scrollIntoView({ behavior: "auto", block: "nearest" }); measure(); }, 120), window.setTimeout(measure, 320)];
    const observed = document.querySelector(".orbit-work-area");
    const observer = observed ? new MutationObserver(measure) : null;
    if (observed && observer) observer.observe(observed, { childList: true, subtree: true });
    window.addEventListener("resize", measure);
    window.addEventListener("scroll", measure, true);
    measure();
    return () => { timers.forEach(clearTimeout); observer?.disconnect(); window.removeEventListener("resize", measure); window.removeEventListener("scroll", measure, true); };
  }, [running, stepIndex, role, steps]);

  useEffect(() => {
    if (!running && !welcome) return undefined;
    function onKeyDown(event) {
      if (event.key === "Escape") { event.preventDefault(); running ? close() : dismissWelcome(); }
      if (running && event.key === "ArrowRight") { event.preventDefault(); move(1); }
      if (running && event.key === "ArrowLeft") { event.preventDefault(); move(-1); }
      if (event.key === "Tab" && dialogRef.current) {
        const buttons = [...dialogRef.current.querySelectorAll("button:not(:disabled)")];
        if (!buttons.length) return;
        const index = buttons.indexOf(document.activeElement);
        if (event.shiftKey && index <= 0) { event.preventDefault(); buttons[buttons.length - 1].focus(); }
        else if (!event.shiftKey && index === buttons.length - 1) { event.preventDefault(); buttons[0].focus(); }
      }
    }
    document.addEventListener("keydown", onKeyDown);
    return () => document.removeEventListener("keydown", onKeyDown);
  }, [running, welcome, stepIndex]);

  useEffect(() => {
    if (running || welcome) dialogRef.current?.querySelector("button")?.focus();
  }, [running, welcome, stepIndex]);

  if (!steps.length) return null;
  const current = steps[stepIndex];
  const welcomeDialog = welcome ? (
    <div className="orfeu-tour-welcome">
      <div className="orfeu-tour-popover" ref={dialogRef} role="dialog" aria-modal="true" aria-labelledby="orfeu-welcome-title">
        <span className="orfeu-tour-count">ORFEU WEB O1.0 · TUTORIAL DO {roleNames[role].toUpperCase()}</span>
        <h2 id="orfeu-welcome-title">Conheça seu painel</h2>
        <p>Deseja realizar uma visita guiada às funções disponíveis para o perfil {roleNames[role]}? O tutorial é ilustrativo e não modifica seus dados.</p>
        <p>Você poderá interromper, retomar ou reiniciar a qualquer momento pelo botão TUTORIAL.</p>
        <div className="orfeu-tour-actions">
          <button type="button" className="primary-button" onClick={() => begin(true)}>INICIAR TUTORIAL</button>
          <button type="button" className="secondary-button" onClick={dismissWelcome}>AGORA NÃO</button>
        </div>
      </div>
    </div>
  ) : null;
  const activeDialog = running && current ? (
    <>
      {overlayStyle(rect).map((style, i) => <div key={i} className="orfeu-tour-dim" style={style} aria-hidden="true" />)}
      {rect ? <div className="orfeu-tour-target" style={{ left: rect.x, top: rect.y, width: rect.width, height: rect.height }} aria-hidden="true" /> : null}
      <div className="orfeu-tour-popover" ref={dialogRef} role="dialog" aria-modal="true" aria-label={`Tutorial do ${roleNames[role]}, etapa ${stepIndex + 1} de ${steps.length}`} style={popoverPosition(rect)}>
        <span className="orfeu-tour-count">TUTORIAL DO {roleNames[role].toUpperCase()} · ETAPA {stepIndex + 1} DE {steps.length}</span>
        <h2>{current.title}</h2>
        <p>{current.text}</p>
        <div className="orfeu-tour-example"><strong>EXEMPLO · </strong>{current.example}</div>
        <div className="orfeu-tour-progress" aria-hidden="true">{steps.map((_, index) => <span key={index} className={index <= stepIndex ? "done" : ""} />)}</div>
        <div className="orfeu-tour-actions">
          <button type="button" className="secondary-button" disabled={stepIndex === 0} onClick={() => move(-1)}>ANTERIOR</button>
          <button type="button" className="primary-button" onClick={() => move(1)}>{stepIndex === steps.length - 1 ? "CONCLUIR" : "PRÓXIMO"}</button>
          <button type="button" className="secondary-button" onClick={() => { setStepIndex(0); persist(0); }}>REINICIAR</button>
          <button type="button" className="secondary-button orfeu-tour-close" onClick={close}>SAIR</button>
        </div>
      </div>
    </>
  ) : null;
  return <>
    <button ref={launcherRef} type="button" className="secondary-button orfeu-tutorial-launcher" onClick={() => begin(Boolean(readProgress(key)?.completed))}>TUTORIAL · {roleNames[role].toUpperCase()}</button>
    {ready ? createPortal(welcomeDialog || activeDialog, document.body) : null}
  </>;
}
