"use client";

import { useEffect, useState } from "react";

export default function OrfeuNotifications() {
  const [items, setItems] = useState([]);
  const [open, setOpen] = useState(false);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState("");
  useEffect(() => {
    let mounted = true;
    async function refresh() {
      try {
        const response = await fetch("/api/notifications", { cache: "no-store" });
        const payload = await response.json();
        if (!response.ok) throw new Error(payload.error || "Falha ao carregar notificações.");
        if (mounted) setItems(Array.isArray(payload.notifications) ? payload.notifications : []);
      } catch (err) { if (mounted) setError(err.message); }
    }
    void refresh();
    const onFocus = () => void refresh();
    window.addEventListener("focus", onFocus);
    return () => { mounted = false; window.removeEventListener("focus", onFocus); };
  }, []);
  const unread = items.filter((item) => !item.read_at).length;
  async function markRead(id) {
    setBusy(id);
    try {
      const response = await fetch("/api/notifications", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ notification_id: id }) });
      const payload = await response.json();
      if (!response.ok) throw new Error(payload.error || "Falha ao marcar notificação.");
      setItems((current) => current.map((item) => item.notification_id === id ? payload.notification : item));
    } catch (err) { setError(err.message); }
    finally { setBusy(""); }
  }
  return <div className="ox-notifications">
    <button type="button" className="ox-notifications-summary" aria-expanded={open} onClick={() => setOpen((value) => !value)}>NOTIFICAÇÕES · {unread} NÃO LIDA(S) {open ? "▴" : "▾"}</button>
    {open ? <div>{error ? <p className="ox-notifications-error">{error}</p> : null}{items.length ? items.slice(0, 20).map((item) => <div className="ox-notifications-item" key={item.notification_id}><p>{!item.read_at ? <strong>● </strong> : null}{item.message}</p><small>{new Date(item.created_at).toLocaleString("pt-BR")}</small>{!item.read_at ? <button type="button" className="table-action" disabled={!!busy} onClick={() => markRead(item.notification_id)}>MARCAR COMO LIDA</button> : null}</div>) : <p className="student-muted">Nenhuma notificação.</p>}</div> : null}
  </div>;
}
