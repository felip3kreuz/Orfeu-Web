"use client";

import { useState } from "react";

export default function FirstAccessPasswordForm({ user }) {
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  async function submit(event) {
    event.preventDefault(); setBusy(true); setError("");
    const form = new FormData(event.currentTarget);
    const current = String(form.get("current") || "");
    const next = String(form.get("next") || "");
    const confirm = String(form.get("confirm") || "");
    if (next !== confirm) { setError("As novas senhas não coincidem."); setBusy(false); return; }
    try {
      const response = await fetch("/api/auth/password", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ current, next }) });
      const data = await response.json();
      if (!response.ok) throw new Error(data?.error || "Falha ao alterar senha.");
      window.location.assign(data.redirectTo || "/");
    } catch (caught) { setError(caught instanceof Error ? caught.message : "Falha ao alterar senha."); setBusy(false); }
  }

  return <section className="orbit-auth-panel"><div className="orbit-auth-heading"><span>CONTA CADASTRADA</span><h2>CRIE SUA SENHA PESSOAL</h2><p>{user.name}, sua conta foi criada por um Administrador. A senha temporária só pode ser usada para este primeiro acesso.</p></div><form className="orbit-form" onSubmit={submit}><label>SENHA TEMPORÁRIA<input name="current" type="password" autoComplete="current-password" required /></label><label>NOVA SENHA<input name="next" type="password" minLength="8" autoComplete="new-password" required /></label><label>CONFIRMAR NOVA SENHA<input name="confirm" type="password" minLength="8" autoComplete="new-password" required /></label>{error ? <div className="form-error">{error}</div> : null}<button className="primary-button" disabled={busy}>{busy ? "SALVANDO…" : "SUBSTITUIR SENHA E CONTINUAR"}</button></form></section>;
}
