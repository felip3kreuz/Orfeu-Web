"use client";

import { useState } from "react";

export default function LogoutButton() {
  const [busy, setBusy] = useState(false);

  async function logout() {
    setBusy(true);
    try {
      await fetch("/api/auth/logout", { method: "POST" });
    } finally {
      try { for (const key of Object.keys(window.sessionStorage)) if (key.startsWith("orfeu:socials:seen:")) window.sessionStorage.removeItem(key); } catch {}
      window.location.assign("/login");
    }
  }

  return (
    <button className="secondary-button" type="button" onClick={logout} disabled={busy}>
      {busy ? "Saindo…" : "Sair"}
    </button>
  );
}
