import Link from "next/link";
import LogoutButton from "@/components/logout-button";
import { roleLabel } from "@/lib/roles";

const content = {
  admin: {
    eyebrow: "Administração",
    title: "Painel do Administrador",
    description: "A sessão administrativa está autenticada no servidor de simulação.",
    cards: [
      ["Usuários", "Gestão de contas e estados entra no W6."],
      ["Mentores", "Credenciamento e permissões entram no W6."],
      ["Servidor", "A conexão autenticada já está ativa nesta build."],
    ],
  },
  mentor: {
    eyebrow: "Mentoria",
    title: "Painel do Mentor",
    description: "A sessão de Mentor está autenticada no servidor de simulação.",
    cards: [
      ["Turmas", "Criação e administração de turmas entra no W6."],
      ["Cenários", "Configuração e acompanhamento entram no W6."],
      ["Resultados", "Visão consolidada dos alunos entra no W6."],
    ],
  },
  aluno: {
    eyebrow: "Simulação",
    title: "Painel do Aluno",
    description: "A sessão do Aluno está autenticada no servidor de simulação.",
    cards: [
      ["Minha empresa", "A operação completa da empresa entra no W5."],
      ["Decisões", "As decisões serão processadas pelo motor de simulação em WebAssembly."],
      ["Sincronização", "O canal autenticado com o servidor já está estabelecido."],
    ],
  },
};

export default function RoleDashboard({ user, role }) {
  const view = content[role];
  return (
    <main>
      <header className="topbar dashboard-topbar">
        <Link className="brand-link" href="/">
          <span className="brand-mark">ORFEU</span>
          <span>
            <span className="brand-name">Orfeu</span>
            <span className="brand-edition">Web · O1.0</span>
          </span>
        </Link>
        <div className="user-menu">
          <div>
            <strong>{user.name}</strong>
            <span>{roleLabel(user.role)}</span>
          </div>
          <LogoutButton />
        </div>
      </header>

      <section className="dashboard-hero">
        <div>
          <p className="eyebrow">{view.eyebrow}</p>
          <h1>{view.title}</h1>
          <p className="lede">{view.description}</p>
        </div>
        <div className="session-card">
          <span className="status-dot" />
          <div>
            <strong>Sessão ativa</strong>
            <span>{user.email}</span>
          </div>
        </div>
      </section>

      <section className="dashboard-grid">
        {view.cards.map(([title, description]) => (
          <article className="dashboard-card" key={title}>
            <h2>{title}</h2>
            <p>{description}</p>
            <span className="coming-soon">Próxima etapa</span>
          </article>
        ))}
      </section>

      <section className="w4-proof">
        <div>
          <p className="eyebrow">W4 concluído neste painel</p>
          <h2>Web → sessão HttpOnly → servidor de simulação</h2>
        </div>
        <p>
          O navegador não recebe o token do servidor. As chamadas autenticadas passam pelo backend
          Next.js, que mantém a compatibilidade com a API RC1.8.
        </p>
      </section>
    </main>
  );
}
