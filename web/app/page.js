import Link from "next/link";
import CoreStatus from "./core-status";
import { JEDChrome, PublicScreenTitle } from "@/components/jed-shell";

export default function Home() {
  return (
    <div className="public-orbit-page">
      <JEDChrome />
      <main className="orbit-frame home-orbit-frame">
        <PublicScreenTitle title="PAINEL DE INICIALIZAÇÃO" subtitle="ambiente de simulação empresarial • versão web" />
        <section className="orbit-home-grid">
          <div className="orbit-dial" aria-label="Orfeu pronto">
            <div className="orbit-dial-ring orbit-dial-ring-a" />
            <div className="orbit-dial-ring orbit-dial-ring-b" />
            <div className="orbit-dial-core"><strong>ORFEU</strong><span>EMPREENDEDORISMO EM JOGO</span><b>READY</b></div>
          </div>
          <div className="orbit-command-stack">
            <Link href="/login" className="orbit-command accent-cyan"><strong>ORFEU ONLINE</strong><span>Entrar com uma conta cadastrada por um Administrador.</span></Link>
            <Link href="/admin" className="orbit-command accent-amber"><strong>CADASTRO CENTRALIZADO</strong><span>Área exclusiva de Administradores para cadastrar usuários individualmente ou por CSV.</span></Link>
            <Link href="/primeiro-acesso" className="orbit-command"><strong>PRIMEIRO ACESSO</strong><span>Entrar com a senha temporária recebida por e-mail e substituí-la por uma senha pessoal.</span></Link>
            <div className="orbit-core-box"><CoreStatus /></div>
          </div>
        </section>
        <footer className="orbit-footer"><span>ORFEU WEB · O1.0</span><span>SIMULAÇÃO EMPREENDEDORA · AVALIAÇÃO PEDAGÓGICA</span></footer>
      </main>
    </div>
  );
}
