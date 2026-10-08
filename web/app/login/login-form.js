"use client";

import { useEffect, useState } from "react";

export default function LoginForm({ firstAccess = false }) {
  const [status, setStatus] = useState({ state: "checking", message: "VERIFICANDO SERVIDOR DE SIMULAÇÃO…" });
  const [error, setError] = useState("");
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    let cancelled = false;
    fetch("/api/server/health", { cache: "no-store" })
      .then(async (response) => {
        const data = await response.json().catch(() => ({}));
        if (cancelled) return;
        if (response.ok && data.ok) setStatus({ state: "ready", message: "SERVIDOR DE SIMULAÇÃO ACESSÍVEL" });
        else setStatus({ state: "error", message: data.error || "SERVIDOR DE SIMULAÇÃO INDISPONÍVEL" });
      })
      .catch(() => { if (!cancelled) setStatus({ state: "error", message: "SERVIDOR DE SIMULAÇÃO INDISPONÍVEL" }); });
    return () => { cancelled = true; };
  }, []);

  async function submit(event) {
    event.preventDefault();
    setSubmitting(true); setError("");
    const form = new FormData(event.currentTarget);
    try {
      const response = await fetch("/api/auth/login", {
        method: "POST", headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ email: String(form.get("email") || ""), password: String(form.get("password") || "") }),
      });
      const data = await response.json().catch(() => ({}));
      if (!response.ok) throw new Error(data.error || "Não foi possível entrar.");
      window.location.assign(data.redirectTo || "/");
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : "Não foi possível conectar à aplicação.");
    } finally { setSubmitting(false); }
  }

  return (
    <div className="orbit-auth-panel">
      <div className={`server-state server-state-${status.state}`}><span className="status-dot" />{status.message}</div>
      <div className="orbit-auth-heading"><span>{firstAccess ? "PRIMEIRO ACESSO" : "ORFEU ONLINE"}</span><h2>{firstAccess ? "ENTRAR COM SENHA TEMPORÁRIA" : "ENTRAR"}</h2><p>{firstAccess ? "Use o e-mail cadastrado e a senha temporária recebida por e-mail. A troca da senha será exigida em seguida." : "Use o e-mail e a senha da sua conta existente."}</p></div>
      <form onSubmit={submit} className="orbit-form">
        <label>E-MAIL<input name="email" type="email" autoComplete="email" required /></label>
        <label>SENHA<input name="password" type="password" autoComplete="current-password" required /></label>
        {error ? <div className="form-error" role="alert">{error}</div> : null}
        <button className="primary-button" type="submit" disabled={submitting || status.state === "error"}>{submitting ? "ENTRANDO…" : firstAccess ? "CONTINUAR PRIMEIRO ACESSO" : "ENTRAR"}</button>
      </form>
      <div className="activation-note"><strong>{firstAccess ? "TROCA OBRIGATÓRIA" : "CADASTRO"}</strong><span>{firstAccess ? "Após autenticar, a plataforma abrirá a tela para substituir a senha temporária." : "Contas são criadas exclusivamente por Administradores. Se este for seu primeiro acesso, use a opção PRIMEIRO ACESSO."}</span></div>
      <div className="auth-link-row">
        {firstAccess ? <a className="secondary-button" href="/login">LOGIN NORMAL</a> : <a className="secondary-button" href="/primeiro-acesso">PRIMEIRO ACESSO</a>}
        <a className="secondary-button" href="/admin">CADASTRO CENTRALIZADO</a>
      </div>
    </div>
  );
}
