import JEDShell from "@/components/jed-shell";
import AdminWorkspace from "@/components/admin-workspace";
import { requireRole } from "@/lib/auth-server";

export const dynamic = "force-dynamic";
export const metadata = { title: "Administrador · Orfeu" };

export default async function AdminPage() {
  const user = await requireRole("admin");
  return <JEDShell user={user} role="admin" title="ADMINISTRAÇÃO" subtitle="usuários • importação CSV • primeiro acesso • e-mail"><AdminWorkspace user={user} /></JEDShell>;
}
