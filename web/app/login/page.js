import LoginForm from "./login-form";
import { JEDChrome, PublicScreenTitle } from "@/components/jed-shell";

export const metadata = { title: "Entrar · Orfeu" };

export default function LoginPage() {
  return (
    <div className="public-orbit-page">
      <JEDChrome />
      <main className="orbit-frame auth-orbit-frame">
        <PublicScreenTitle title="ORFEU ONLINE" subtitle="contas • turmas • administração • sincronização" />
        <div className="orbit-auth-grid">
          <div className="orbit-auth-copy">
            <span className="orbit-index">01 / ACESSO</span>
            <h2>NENHUMA SESSÃO ATIVA</h2>
            <p>Contas de Aluno, Mentor e Administrador são cadastradas exclusivamente por Administradores. No primeiro acesso, use a senha temporária recebida por e-mail e substitua-a por uma senha pessoal.</p>
          </div>
          <LoginForm />
        </div>
      </main>
    </div>
  );
}
