"use client";

import { useState } from "react";

export default function LogoutButton() {
  const [busy, setBusy] = useState(false);

  async function logout() {
    setBusy(true);
    try {
      await fetch("/api/auth/logout", { method: "POST" });
    } finally {
      window.location.assign("/login");
    }
  }

  return (
    <button className="secondary-button" type="button" onClick={logout} disabled={busy}>
      {busy ? "Saindo…" : "Sair"}
    </button>
  );
}
