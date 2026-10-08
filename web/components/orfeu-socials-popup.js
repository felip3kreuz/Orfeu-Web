"use client";

import { useEffect, useMemo, useState } from "react";
import { createPortal } from "react-dom";

// Apenas URLs públicas verificadas. URLs adicionais são opcionais e configuradas na Vercel.
const socials = [
  { label: "Instagram do JED", value: "@jovensempreendedores.digitais", url: "https://www.instagram.com/jovensempreendedores.digitais/" },
  { label: "YouTube do JED", value: "@jovensempreendedoresdigitais", url: "https://www.youtube.com/@jovensempreendedoresdigitais" },
  { label: "Facebook do JED", value: "Página oficial", url: process.env.NEXT_PUBLIC_JED_FACEBOOK_URL },
  { label: "WhatsApp do JED", value: "Canal oficial", url: process.env.NEXT_PUBLIC_JED_WHATSAPP_URL },
  { label: "Outras redes e canais do JED", value: "Página oficial de links", url: "https://linktr.ee/jovensempreendedoresdigitais" },
].filter((link) => /^https:\/\//i.test(link.url || ""));

export default function OrfeuSocialsPopup({ user, role }) {
  const [ready, setReady] = useState(false);
  const [open, setOpen] = useState(false);
  const key = useMemo(() => `orfeu:socials:seen:${String(user?.id || user?.email || "user")}:${role}`, [user?.id, user?.email, role]);

  useEffect(() => {
    setReady(true);
    function showWhenAvailable() {
      try { if (window.sessionStorage.getItem(key)) return; } catch {}
      // O tutorial da primeira entrada tem prioridade sobre os links sociais.
      const tutorialKey = `orfeu:tutorial:o1.0:${role}:${String(user?.id || user?.email || "sessao")}`;
      try { if (!window.localStorage.getItem(tutorialKey)) return; } catch {}
      setOpen(true);
    }
    showWhenAvailable();
    window.addEventListener("orfeu:tutorial:idle", showWhenAvailable);
    return () => window.removeEventListener("orfeu:tutorial:idle", showWhenAvailable);
  }, [key, role, user?.id, user?.email]);

  function close() {
    setOpen(false);
    try { window.sessionStorage.setItem(key, "1"); } catch {}
  }
  useEffect(() => {
    if (!open) return undefined;
    const onEscape = (event) => { if (event.key === "Escape") close(); };
    document.addEventListener("keydown", onEscape);
    return () => document.removeEventListener("keydown", onEscape);
  }, [open]);
  if (!ready || !open) return null;
  return createPortal(
    <div className="orfeu-social-backdrop" onMouseDown={(e) => { if (e.target === e.currentTarget) close(); }}>
      <section className="orfeu-social-dialog" role="dialog" aria-modal="true" aria-labelledby="orfeu-social-title">
        <span className="eyebrow">JOVENS EMPREENDEDORES DIGITAIS</span>
        <h2 id="orfeu-social-title">Acompanhe o JED nas redes</h2>
        <p>Conheça os canais oficiais do programa Jovens Empreendedores Digitais. Os links são externos ao Orfeu.</p>
        <div className="orfeu-social-links">{socials.map((link) => <a key={link.label} href={link.url} target="_blank" rel="noopener noreferrer"><strong>{link.label}</strong><span>{link.value} ↗</span></a>)}</div>
        <div className="orfeu-tour-actions"><button className="primary-button" type="button" onClick={close}>CONTINUAR NO ORFEU</button></div>
      </section>
    </div>, document.body
  );
}
