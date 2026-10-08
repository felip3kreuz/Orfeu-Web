import LoginForm from "../login/login-form";
import { JEDChrome, PublicScreenTitle } from "@/components/jed-shell";

export const metadata = { title: "Primeiro acesso · Orfeu" };

export default function FirstAccessPage() {
  return (
    <div className="public-orbit-page">
      <JEDChrome />
      <main className="orbit-frame auth-orbit-frame">
        <PublicScreenTitle title="PRIMEIRO ACESSO" subtitle="senha temporária • substituição obrigatória" />
        <div className="orbit-auth-grid">
          <div className="orbit-auth-copy">
            <span className="orbit-index">01 / PRIMEIRO ACESSO</span>
            <h2>CONTA JÁ CADASTRADA</h2>
            <p>Use o e-mail cadastrado pelo Administrador e a senha temporária enviada para esse endereço. Depois da autenticação, o Orfeu exigirá uma nova senha pessoal antes de liberar o restante da plataforma.</p>
          </div>
          <LoginForm firstAccess />
        </div>
      </main>
    </div>
  );
}
