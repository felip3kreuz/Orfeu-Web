"use client";

import { useEffect, useMemo, useState } from "react";
import Modal from "@/components/modal";

const TEMP_PASSWORD = "abcd1234";

function roleName(role) {
  if (role === "admin") return "ADMINISTRADOR";
  if (role === "mentor" || role === "tutor") return "MENTOR";
  if (role === "aluno") return "ALUNO";
  return String(role || "USUÁRIO").toUpperCase();
}

function emailState(account) {
  if (account.email_status === "sent") return ["ENVIADO", "active"];
  if (account.email_status === "failed") return ["FALHOU", "revoked"];
  if (account.email_status === "not_configured") return ["SMTP NÃO CONFIG.", "used"];
  return ["—", "used"];
}

function parseCSVLine(line, delimiter) {
  const values = [];
  let current = "";
  let quoted = false;
  for (let i = 0; i < line.length; i += 1) {
    const ch = line[i];
    if (ch === '"') {
      if (quoted && line[i + 1] === '"') { current += '"'; i += 1; }
      else quoted = !quoted;
    } else if (ch === delimiter && !quoted) {
      values.push(current.trim()); current = "";
    } else current += ch;
  }
  values.push(current.trim());
  return values;
}

function normalizeHeader(value) {
  return String(value || "").trim().toLowerCase().normalize("NFD").replace(/[\u0300-\u036f]/g, "").replace(/[\s-]+/g, "_");
}

function parseUsersCSV(text) {
  const lines = String(text || "").split(/\r?\n/).filter((line) => line.trim());
  if (lines.length < 2) throw new Error("O CSV deve conter cabeçalho e pelo menos um usuário.");
  const delimiter = lines[0].includes(";") ? ";" : ",";
  const headers = parseCSVLine(lines[0], delimiter).map(normalizeHeader);
  const index = (names) => headers.findIndex((h) => names.includes(h));
  const nameIndex = index(["nome", "name"]);
  const emailIndex = index(["email", "e_mail"]);
  const roleIndex = index(["papel", "perfil", "role"]);
  const institutionIndex = index(["instituicao", "institution"]);
  const idIndex = index(["id_institucional", "identificador_institucional", "institutional_id"]);
  const classIndex = index(["turma", "classe", "class", "class_ref", "turma_id"]);
  if (nameIndex < 0 || emailIndex < 0 || roleIndex < 0) throw new Error("Cabeçalho obrigatório: nome,email,papel. Instituição, ID institucional e turma são opcionais.");

  return lines.slice(1).map((line, row) => {
    const cells = parseCSVLine(line, delimiter);
    const rawRole = String(cells[roleIndex] || "").trim().toLowerCase();
    const role = rawRole === "administrador" ? "admin" : rawRole === "tutor" ? "mentor" : rawRole;
    if (!["aluno", "mentor", "admin"].includes(role)) throw new Error(`Linha ${row + 2}: papel deve ser aluno, mentor ou admin.`);
    const name = String(cells[nameIndex] || "").trim();
    const email = String(cells[emailIndex] || "").trim().toLowerCase();
    if (!name || !email.includes("@")) throw new Error(`Linha ${row + 2}: nome ou e-mail inválido.`);
    return {
      name,
      email,
      role,
      institution: institutionIndex >= 0 ? String(cells[institutionIndex] || "").trim() : "",
      institutional_id: idIndex >= 0 ? String(cells[idIndex] || "").trim() : "",
      class_ref: classIndex >= 0 ? String(cells[classIndex] || "").trim() : "",
    };
  });
}

export default function AdminWorkspace({ user }) {
  const [data, setData] = useState({ users: [], classes: [], email: { configured: false } });
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState("");
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const [dialog, setDialog] = useState("");
  const [form, setForm] = useState({ role: "aluno", name: "", email: "", institution: "", institutional_id: "", class_id: "" });
  const [classForm, setClassForm] = useState({ name: "", mentor_id: "" });
  const [classMentor, setClassMentor] = useState({ class_id: "", name: "", mentor_id: "" });
  const [assignment, setAssignment] = useState({ user_id: "", student_name: "", class_id: "" });
  const [deleteTarget, setDeleteTarget] = useState(null);
  const [deleteConfirm, setDeleteConfirm] = useState("");

  async function load() {
    setLoading(true); setError("");
    try {
      const response = await fetch("/api/admin/overview", { cache: "no-store" });
      const payload = await response.json();
      if (!response.ok) throw new Error(payload?.error || "Falha ao carregar administração.");
      const next = {
        users: Array.isArray(payload.users) ? payload.users : [],
        classes: Array.isArray(payload.classes) ? payload.classes : [],
        email: payload.email || { configured: false },
      };
      setData(next);
      // Do not force the first Mentor: classes may be opened before one is appointed.
    } catch (caught) { setError(caught instanceof Error ? caught.message : "Falha ao carregar administração."); }
    finally { setLoading(false); }
  }
  useEffect(() => { void load(); }, []);

  const mentors = useMemo(() => data.users.filter((entry) => (entry.role === "mentor" || entry.role === "tutor") && entry.status !== "disabled"), [data.users]);
  const stats = useMemo(() => ({
    admins: data.users.filter((u) => u.role === "admin").length,
    mentors: data.users.filter((u) => u.role === "mentor" || u.role === "tutor").length,
    students: data.users.filter((u) => u.role === "aluno").length,
    classes: data.classes.length,
  }), [data.users, data.classes]);

  function classByID(id) { return data.classes.find((entry) => entry.id === id); }
  function classLabel(cl) { return cl ? `T${cl.number} · ${cl.name}` : "AGUARDANDO TURMA"; }
  function mentorName(id) { return !id ? "SEM MENTOR" : data.users.find((entry) => entry.id === id)?.name || "Mentor não encontrado"; }
  const assignedClasses = data.classes.filter((cl) => Boolean(cl.tutor_id));

  function resolveCSVClass(ref) {
    const value = String(ref || "").trim();
    if (!value) return "";
    const normalized = value.toUpperCase().replace(/^T/, "");
    return data.classes.find((cl) => cl.id === value || String(cl.number) === normalized || cl.name.toLowerCase() === value.toLowerCase())?.id || null;
  }

  async function action(input, label = "Operação concluída.") {
    setBusy(input.action); setError(""); setNotice("");
    try {
      const response = await fetch("/api/admin/actions", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(input) });
      const payload = await response.json();
      if (!response.ok) throw new Error(payload?.error || "Falha na operação.");
      setNotice(label);
      await load();
      return payload.result || true;
    } catch (caught) { setError(caught instanceof Error ? caught.message : "Falha na operação."); return false; }
    finally { setBusy(""); }
  }

  async function createUser(event) {
    event.preventDefault();
    const result = await action({ action: "create_user", ...form, class_id: form.role === "aluno" ? form.class_id : "" }, `${roleName(form.role)} cadastrado.`);
    if (result) {
      const enrollmentMessage = result.enrollment_id ? ` Matrícula: ${result.enrollment_id}.` : result.role === "aluno" ? " Aluno aguardando turma." : "";
      const emailMessage = result.email_status === "sent" ? " E-mail enviado." : result.email_status === "not_configured" ? " SMTP não configurado: o e-mail ainda não foi enviado." : result.email_status === "failed" ? " O envio do e-mail falhou; confira a configuração SMTP." : "";
      setNotice(`${roleName(form.role)} cadastrado. Senha temporária: ${TEMP_PASSWORD}.${enrollmentMessage}${emailMessage}`);
      setForm({ role: "aluno", name: "", email: "", institution: "", institutional_id: "", class_id: "" });
      setDialog("");
    }
  }

  async function createClass(event) {
    event.preventDefault();
    const result = await action({ action: "create_class", ...classForm }, "Turma criada e numerada automaticamente.");
    if (result) {
      setNotice(`Turma T${result.number} · ${result.name} criada. ${result.tutor_id ? `Mentor: ${mentorName(result.tutor_id)}.` : "Aguardando designação de Mentor."}`);
      setClassForm({ name: "", mentor_id: "" });
      setDialog("");
    }
  }

  function openClassMentor(cl) {
    setClassMentor({ class_id: cl.id, name: `T${cl.number} · ${cl.name}`, mentor_id: "" });
    setDialog("classMentor");
  }

  async function saveClassMentor(event) {
    event.preventDefault();
    const result = await action({ action: "assign_class_mentor", class_id: classMentor.class_id, mentor_id: classMentor.mentor_id });
    if (result) {
      setNotice(`${classMentor.name}: Mentor ${mentorName(result.tutor_id)} designado.`);
      setDialog("");
    }
  }

  function openAssignment(account) {
    setAssignment({ user_id: account.id, student_name: account.name, class_id: account.current_class_id || "" });
    setDialog("assignment");
  }

  async function saveAssignment(event) {
    event.preventDefault();
    const previous = data.users.find((entry) => entry.id === assignment.user_id);
    const result = await action({ action: "assign_student_class", user_id: assignment.user_id, class_id: assignment.class_id }, assignment.class_id ? "Matrícula atualizada." : "Aluno removido da turma atual e colocado em espera.");
    if (result) {
      const changed = previous?.current_class_id !== result.current_class_id;
      setNotice(result.enrollment_id ? `${assignment.student_name}: ${changed ? "nova matrícula" : "matrícula"} ${result.enrollment_id}.` : `${assignment.student_name}: aguardando turma.`);
      setDialog("");
    }
  }

  async function importCSV(event) {
    const file = event.target.files?.[0];
    if (!file) return;
    setBusy("import_users"); setError(""); setNotice("");
    try {
      const rows = parseUsersCSV(await file.text());
      const created = [];
      const failures = [];
      for (let index = 0; index < rows.length; index += 1) {
        const row = rows[index];
        try {
          let classID = "";
          if (row.class_ref) {
            if (row.role !== "aluno") throw new Error("a coluna turma só pode ser usada para Alunos");
            classID = resolveCSVClass(row.class_ref);
            if (classID === null) throw new Error(`turma não encontrada: ${row.class_ref}`);
          }
          const response = await fetch("/api/admin/actions", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ action: "create_user", ...row, class_id: classID }) });
          const payload = await response.json();
          if (!response.ok) throw new Error(payload?.error || "Falha ao cadastrar usuário.");
          created.push(payload.result);
        } catch (caught) {
          failures.push({ row: index + 2, email: row.email, error: caught instanceof Error ? caught.message : "Falha no cadastro" });
        }
      }
      const sent = created.filter((u) => u?.email_status === "sent").length;
      const enrolled = created.filter((u) => u?.role === "aluno" && u?.enrollment_id).length;
      setNotice(`${created.length} usuário(s) cadastrado(s); ${enrolled} matrícula(s) gerada(s); ${sent} e-mail(s) enviado(s); ${failures.length} linha(s) rejeitada(s).`);
      if (failures.length) setError(failures.slice(0, 5).map((item) => `Linha ${item.row}: ${item.email || "—"} — ${item.error}`).join(" | "));
      await load();
    } catch (caught) { setError(caught instanceof Error ? caught.message : "Falha ao importar CSV."); }
    finally { setBusy(""); event.target.value = ""; }
  }

  function downloadTemplate() {
    const sampleClass = data.classes[0] ? `T${data.classes[0].number}` : "";
    const csv = `nome,email,papel,instituicao,id_institucional,turma\nAna Lima,ana@example.com,aluno,Escola Modelo,A-001,${sampleClass}\nCarlos Souza,carlos@example.com,mentor,Escola Modelo,M-001,\nMaria Silva,maria@example.com,admin,Escola Modelo,ADM-002,\n`;
    const url = URL.createObjectURL(new Blob([csv], { type: "text/csv;charset=utf-8" }));
    const a = document.createElement("a"); a.href = url; a.download = "modelo_usuarios_jed.csv"; a.click(); URL.revokeObjectURL(url);
  }

  function mentorClassCount(account) {
    return data.classes.filter((cl) => cl.tutor_id === account.id).length;
  }

  function requestDelete(target) {
    setError(""); setNotice(""); setDeleteConfirm(""); setDeleteTarget(target); setDialog("delete");
  }

  async function confirmDelete(event) {
    event.preventDefault();
    if (!deleteTarget || deleteConfirm.trim().toUpperCase() !== "EXCLUIR") return;
    const input = deleteTarget.type === "class"
      ? { action: "delete_class", class_id: deleteTarget.id }
      : { action: "delete_user", user_id: deleteTarget.id };
    const label = deleteTarget.type === "class" ? `Turma ${deleteTarget.name} excluída.` : `${deleteTarget.name} excluído(a).`;
    const result = await action(input, label);
    if (result) { setDialog(""); setDeleteTarget(null); setDeleteConfirm(""); }
  }

  if (loading) return <section className="workspace-loading">CARREGANDO USUÁRIOS, TURMAS E CONFIGURAÇÃO…</section>;

  return <>
    <section id="visao-geral" className="orbit-overview-strip">
      <div><span>SESSÃO</span><strong>{user.is_primary_admin ? "ADMINISTRADOR PRINCIPAL" : "ADMINISTRADOR"}</strong><small>Contas, turmas e matrículas centralizadas</small></div>
      <div className="orbit-stat-row">
        <div><span>ADMINS</span><strong>{stats.admins}</strong></div><div><span>MENTORES</span><strong>{stats.mentors}</strong></div><div><span>ALUNOS</span><strong>{stats.students}</strong></div><div><span>TURMAS</span><strong>{stats.classes}</strong></div>
      </div>
    </section>

    {error ? <div className="workspace-alert workspace-alert-error">ATENÇÃO · {error}</div> : null}
    {notice ? <div className="workspace-alert workspace-alert-success">OK · {notice}</div> : null}
    {!data.email?.configured ? <div className="workspace-alert workspace-alert-error">E-MAIL NÃO CONFIGURADO · As contas serão criadas, mas as mensagens de cadastro não poderão ser entregues até configurar o SMTP do servidor de simulação.</div> : <div className="workspace-alert workspace-alert-success">E-MAIL ATIVO · Remetente: {data.email.from_email || "configurado"}</div>}

    <section className="orbit-toolbar" aria-label="Ações administrativas">
      <button className="primary-button" type="button" onClick={() => setDialog("user")}>CADASTRAR USUÁRIO</button>
      <button className="secondary-button" type="button" onClick={() => setDialog("class")}>NOVA TURMA</button>
      <label className="secondary-button file-button">IMPORTAR CSV<input type="file" accept=".csv,text/csv" onChange={importCSV} disabled={Boolean(busy)} /></label>
      <button className="secondary-button" type="button" onClick={downloadTemplate}>BAIXAR MODELO CSV</button>
      <button className="secondary-button" type="button" onClick={load}>ATUALIZAR</button>
    </section>

    <section id="admin-turmas" className="workspace-section orbit-section">
      <div className="workspace-section-head"><div><p className="eyebrow">TURMAS</p><h2>GESTÃO CENTRALIZADA</h2></div><span className="section-count">{data.classes.length} TURMA(S)</span></div>
      <p className="section-help">Crie uma turma com nome e Mentor opcional: o número T é sequencial e permanente. Turmas sem Mentor aguardam designação antes das matrículas. Cada Aluno matriculado recebe uma ID própria, como <strong>T25A4</strong>. Apenas o Administrador Principal pode excluir turmas.</p>
      <div className="table-wrap"><table className="workspace-table"><thead><tr><th>ID</th><th>Nome</th><th>Mentor</th><th>Alunos</th><th>Cenário</th>{user.is_primary_admin ? <th>Ação</th> : null}</tr></thead><tbody>{data.classes.length ? data.classes.map((cl) => <tr key={cl.id}><td><strong>T{cl.number}</strong></td><td>{cl.name}</td><td>{cl.tutor_id ? mentorName(cl.tutor_id) : <><strong>AGUARDANDO MENTOR</strong><button className="table-action" type="button" disabled={!mentors.length || Boolean(busy)} onClick={() => openClassMentor(cl)} title={!mentors.length ? "Cadastre um Mentor para poder designá-lo" : "Designar Mentor à turma"}>DESIGNAR MENTOR</button></>}</td><td>{cl.student_ids?.length || 0}</td><td>{cl.scenario?.nome || "Mercado estável"}</td>{user.is_primary_admin ? <td><button className="table-action danger-action" disabled={Boolean(busy)} onClick={() => requestDelete({ type: "class", id: cl.id, name: `T${cl.number} · ${cl.name}`, detail: `${cl.student_ids?.length || 0} aluno(s) · Mentor: ${mentorName(cl.tutor_id)}` })}>EXCLUIR</button></td> : null}</tr>) : <tr><td colSpan={user.is_primary_admin ? 6 : 5} className="empty-cell">Nenhuma turma cadastrada. Clique em NOVA TURMA para criar a primeira, mesmo sem Mentor.</td></tr>}</tbody></table></div>
    </section>

    <section id="usuarios" className="workspace-section orbit-section">
      <div className="workspace-section-head"><div><p className="eyebrow">CONTAS</p><h2>USUÁRIOS DO SERVIDOR</h2></div><span className="section-count">{data.users.length} REGISTROS</span></div>
      <p className="section-help">Ao cadastrar um Aluno, a turma pode ser definida imediatamente. Alunos sem turma permanecem como <strong>AGUARDANDO TURMA</strong> até o Administrador vinculá-los.</p>
      <div className="table-wrap"><table className="workspace-table"><thead><tr><th>Nome</th><th>Papel</th><th>E-mail</th><th>Turma / matrícula</th><th>Primeiro acesso</th><th>E-mail</th><th>Status</th>{user.is_primary_admin ? <th>Ação</th> : null}</tr></thead><tbody>{data.users.map((account) => {
        const active = account.status !== "disabled";
        const [mailLabel, mailClass] = emailState(account);
        const cl = account.role === "aluno" ? classByID(account.current_class_id) : null;
        return <tr key={account.id}>
          <td><strong>{account.name}</strong>{account.is_primary_admin ? <small>ADMIN PRINCIPAL</small> : null}{account.institutional_id ? <small>{account.institutional_id}</small> : null}</td>
          <td><span className={`role-chip role-${account.role}`}>{roleName(account.role)}</span></td>
          <td>{account.email}</td>
          <td>{account.role === "aluno" ? <><strong>{classLabel(cl)}</strong>{account.enrollment_id ? <small>{account.enrollment_id}</small> : null}<button className="table-action" disabled={Boolean(busy)} onClick={() => openAssignment(account)}>{account.current_class_id ? "TRANSFERIR" : "VINCULAR"}</button></> : "—"}</td>
          <td>{account.must_change_password ? <span className="state-label active">TROCA PENDENTE</span> : <span className="state-label used">CONCLUÍDO</span>}</td>
          <td><span className={`state-label ${mailClass}`} title={account.email_error || ""}>{mailLabel}</span>{account.must_change_password && account.email_status !== "sent" ? <button className="table-action" disabled={Boolean(busy)} onClick={() => action({ action: "resend_email", user_id: account.id }, "E-mail de cadastro reenviado.")}>REENVIAR</button> : null}</td>
          <td><button className={`table-action ${active ? "state-active" : ""}`} disabled={Boolean(busy) || account.id === user.id} onClick={() => action({ action: "user_status", user_id: account.id, status: active ? "disabled" : "active" }, active ? "Conta desativada." : "Conta ativada.")}>{active ? "ATIVA" : "DESATIVADA"}</button></td>
          {user.is_primary_admin ? <td>{account.id === user.id || account.is_primary_admin ? <span className="student-muted">PROTEGIDO</span> : (account.role === "mentor" || account.role === "tutor") && mentorClassCount(account) > 0 ? <button className="table-action danger-action" disabled title="Exclua primeiro as turmas deste Mentor">{mentorClassCount(account)} TURMA(S)</button> : <button className="table-action danger-action" disabled={Boolean(busy)} onClick={() => requestDelete({ type: "user", id: account.id, name: account.name, detail: `${roleName(account.role)} · ${account.email}` })}>EXCLUIR</button>}</td> : null}
        </tr>;
      })}</tbody></table></div>
    </section>

    <section id="importacao" className="workspace-section orbit-section">
      <div className="workspace-section-head"><div><p className="eyebrow">IMPORTAÇÃO</p><h2>LISTA CSV</h2></div></div>
      <div className="workspace-card"><p className="section-help">Cabeçalhos: <code>nome,email,papel,instituicao,id_institucional,turma</code>. Para Alunos, <code>turma</code> aceita <code>T25</code>, <code>25</code> ou o nome exato da turma. Em branco significa AGUARDANDO TURMA.</p></div>
    </section>

    <Modal open={dialog === "delete"} title="CONFIRMAR EXCLUSÃO" subtitle={deleteTarget?.name || "Registro selecionado"} onClose={() => { setDialog(""); setDeleteTarget(null); setDeleteConfirm(""); }}>
      <form className="orbit-form" onSubmit={confirmDelete}>
        <div className="destructive-warning"><strong>EXCLUSÃO DEFINITIVA</strong><span>{deleteTarget?.detail || ""}</span><small>{deleteTarget?.type === "class" ? "A turma será removida. Os Alunos permanecerão cadastrados como AGUARDANDO TURMA, o histórico de matrícula será encerrado e os empreendimentos serão preservados sem vínculo com a turma excluída." : "A conta será removida do servidor e suas sessões serão encerradas. Se for Aluno, seus empreendimentos persistidos também serão removidos."}</small></div>
        <label>DIGITE EXCLUIR PARA CONFIRMAR<input value={deleteConfirm} onChange={(e) => setDeleteConfirm(e.target.value)} autoComplete="off" /></label>
        <div className="modal-actions"><button type="button" className="secondary-button" onClick={() => { setDialog(""); setDeleteTarget(null); setDeleteConfirm(""); }}>CANCELAR</button><button className="danger-button" disabled={Boolean(busy) || deleteConfirm.trim().toUpperCase() !== "EXCLUIR"}>EXCLUIR DEFINITIVAMENTE</button></div>
      </form>
    </Modal>

    <Modal open={dialog === "user"} title="CADASTRAR USUÁRIO" subtitle={`A senha inicial será ${TEMP_PASSWORD} e deverá ser alterada no primeiro acesso.`} onClose={() => setDialog("")}>
      <form className="orbit-form" onSubmit={createUser}>
        <label>PERFIL<select value={form.role} onChange={(e) => setForm({ ...form, role: e.target.value, class_id: e.target.value === "aluno" ? form.class_id : "" })}><option value="aluno">Aluno</option><option value="mentor">Mentor</option><option value="admin">Administrador</option></select></label>
        <label>NOME<input value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} required /></label>
        <label>E-MAIL<input type="email" value={form.email} onChange={(e) => setForm({ ...form, email: e.target.value })} required /></label>
        <div className="orbit-form-two"><label>INSTITUIÇÃO<input value={form.institution} onChange={(e) => setForm({ ...form, institution: e.target.value })} /></label><label>ID INSTITUCIONAL<input value={form.institutional_id} onChange={(e) => setForm({ ...form, institutional_id: e.target.value })} /></label></div>
        {form.role === "aluno" ? <label>TURMA<select value={form.class_id} onChange={(e) => setForm({ ...form, class_id: e.target.value })}><option value="">Aguardando turma</option>{assignedClasses.map((cl) => <option key={cl.id} value={cl.id}>T{cl.number} · {cl.name} · {mentorName(cl.tutor_id)}</option>)}</select></label> : null}
        {form.role === "aluno" && form.class_id ? <div className="activation-note"><strong>MATRÍCULA AUTOMÁTICA</strong><span>O sistema gerará o próximo identificador disponível da turma, por exemplo T25A4.</span></div> : null}
        <div className="activation-note"><strong>SENHA TEMPORÁRIA</strong><span>{TEMP_PASSWORD} · troca obrigatória no primeiro acesso.</span></div>
        <div className="modal-actions"><button type="button" className="secondary-button" onClick={() => setDialog("")}>CANCELAR</button><button className="primary-button" disabled={Boolean(busy)}>CADASTRAR E AVISAR POR E-MAIL</button></div>
      </form>
    </Modal>

    <Modal open={dialog === "class"} title="CRIAR TURMA" subtitle="O Administrador define o nome; o número da turma é gerado automaticamente." onClose={() => setDialog("")}>
      <form className="orbit-form" onSubmit={createClass}>
        <label>NOME DA TURMA<input value={classForm.name} onChange={(e) => setClassForm({ ...classForm, name: e.target.value })} placeholder="Empreendedorismo 2026 — Turma B" required /></label>
        <label>MENTOR RESPONSÁVEL (OPCIONAL)<select value={classForm.mentor_id} onChange={(e) => setClassForm({ ...classForm, mentor_id: e.target.value })}><option value="">Designar posteriormente</option>{mentors.map((mentor) => <option key={mentor.id} value={mentor.id}>{mentor.name} · {mentor.email}</option>)}</select></label>
        <div className="activation-note"><strong>NUMERAÇÃO AUTOMÁTICA</strong><span>A turma receberá um identificador permanente T (por exemplo, T26). {classForm.mentor_id ? "Alunos poderão ser matriculados normalmente." : "Sem Mentor, a turma ficará aguardando designação e não receberá matrículas ainda."}</span></div>
        <div className="modal-actions"><button type="button" className="secondary-button" onClick={() => setDialog("")}>CANCELAR</button><button className="primary-button" disabled={Boolean(busy) || !classForm.name.trim()}>CRIAR TURMA</button></div>
      </form>
    </Modal>

    <Modal open={dialog === "classMentor"} title="DESIGNAR MENTOR" subtitle={classMentor.name || "Turma"} onClose={() => setDialog("")}>
      <form className="orbit-form" onSubmit={saveClassMentor}>
        <label>MENTOR RESPONSÁVEL<select value={classMentor.mentor_id} onChange={(e) => setClassMentor({ ...classMentor, mentor_id: e.target.value })} required><option value="">Selecione um Mentor</option>{mentors.map((mentor) => <option key={mentor.id} value={mentor.id}>{mentor.name} · {mentor.email}</option>)}</select></label>
        <div className="activation-note"><strong>ATIVAR MATRÍCULAS</strong><span>Após a designação, o Administrador poderá matricular alunos e o Mentor verá esta turma no próprio painel. O número T permanece inalterado.</span></div>
        <div className="modal-actions"><button type="button" className="secondary-button" onClick={() => setDialog("")}>CANCELAR</button><button className="primary-button" disabled={Boolean(busy) || !classMentor.mentor_id}>CONFIRMAR MENTOR</button></div>
      </form>
    </Modal>

    <Modal open={dialog === "assignment"} title="TURMA DO ALUNO" subtitle={assignment.student_name || "Aluno"} onClose={() => setDialog("")}>
      <form className="orbit-form" onSubmit={saveAssignment}>
        <label>TURMA<select value={assignment.class_id} onChange={(e) => setAssignment({ ...assignment, class_id: e.target.value })}><option value="">Aguardando turma</option>{assignedClasses.map((cl) => <option key={cl.id} value={cl.id}>T{cl.number} · {cl.name} · {mentorName(cl.tutor_id)}</option>)}</select></label>
        <div className="activation-note"><strong>HISTÓRICO PRESERVADO</strong><span>Ao transferir, a matrícula anterior é encerrada e uma nova ID é criada na turma de destino. IDs antigas nunca são reutilizadas.</span></div>
        <div className="modal-actions"><button type="button" className="secondary-button" onClick={() => setDialog("")}>CANCELAR</button><button className="primary-button" disabled={Boolean(busy)}>SALVAR VÍNCULO</button></div>
      </form>
    </Modal>
  </>;
}
