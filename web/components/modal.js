"use client";

import { useEffect } from "react";

export default function Modal({ open, title, subtitle, onClose, children }) {
  useEffect(() => {
    if (!open) return;
    function key(event) { if (event.key === "Escape") onClose?.(); }
    document.addEventListener("keydown", key);
    return () => document.removeEventListener("keydown", key);
  }, [open, onClose]);

  if (!open) return null;
  return (
    <div className="orbit-modal-backdrop" role="presentation" onMouseDown={(event) => { if (event.target === event.currentTarget) onClose?.(); }}>
      <section className="orbit-modal" role="dialog" aria-modal="true" aria-label={title}>
        <header>
          <div><span>ORFEU ONLINE</span><h2>{title}</h2>{subtitle ? <p>{subtitle}</p> : null}</div>
          <button type="button" className="orbit-close" onClick={onClose} aria-label="Fechar">×</button>
        </header>
        <div className="orbit-modal-body">{children}</div>
      </section>
    </div>
  );
}
