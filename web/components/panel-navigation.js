"use client";

import { useEffect, useMemo, useState } from "react";

const navigation = {
  admin: [],
  mentor: [
    ["visao-geral", "VISÃO GERAL", "turmas e resultados"],
    ["cenarios", "CENÁRIOS", "ambientes pedagógicos"],
    ["turmas", "TURMAS", "atribuição administrativa"],
    ["alunos", "ALUNOS", "matrículas das turmas"],
    ["resultados", "RESULTADOS", "empresas sincronizadas"],
  ],
  aluno: [
    ["empresa", "MINHA EMPRESA", "criar e selecionar"],
    ["persona", "PERSONA", "cliente principal"],
    ["canvas", "LEAN CANVAS", "modelo de negócio"],
    ["digital", "CANAIS E FERRAMENTAS", "presença digital"],
    ["decisoes", "DECISÕES", "próxima semana"],
    ["insumos", "INSUMOS", "estoque e compras"],
    ["financeiro", "FINANCEIRO", "caixa e compromissos"],
    ["indicadores", "INDICADORES", "score e histórico"],
    ["jornada", "JORNADA ORFEU", "percurso pedagógico"],
    ["turmas", "TURMA", "vínculo administrativo"],
  ],
};

export default function PanelNavigation({ role }) {
  const items = useMemo(() => navigation[role] || [], [role]);
  const [active, setActive] = useState(items[0]?.[0] || "");

  useEffect(() => {
    setActive(items[0]?.[0] || "");
    const onView = (event) => {
      if (event?.detail?.role && event.detail.role !== role) return;
      const view = String(event?.detail?.view || "");
      if (items.some(([candidate]) => candidate === view)) setActive(view);
    };
    window.addEventListener("jed:view", onView);
    return () => window.removeEventListener("jed:view", onView);
  }, [items, role]);

  function navigate(view) {
    setActive(view);
    window.dispatchEvent(new CustomEvent("jed:navigate", { detail: { role, view } }));

    // Admin ainda usa seções contínuas; este fallback mantém esse painel funcional.
    window.setTimeout(() => {
      const target = document.getElementById(view);
      if (target) target.scrollIntoView({ behavior: "smooth", block: "start" });
    }, 0);
  }

  if (!items.length) return null;

  return (
    <nav>
      {items.map(([view, label, detail]) => (
        <button
          key={view}
          data-orfeu-view={view}
          type="button"
          className={active === view ? "active orbit-nav-button" : "orbit-nav-button"}
          aria-current={active === view ? "page" : undefined}
          onClick={() => navigate(view)}
        >
          <strong>{label}</strong>
          <span>{detail}</span>
        </button>
      ))}
    </nav>
  );
}
