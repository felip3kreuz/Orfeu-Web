import JEDShell from "@/components/jed-shell";
import StudentWorkspace from "@/components/student-workspace";
import { requireRole } from "@/lib/auth-server";

export const dynamic = "force-dynamic";
export const metadata = { title: "Aluno · Orfeu" };

export default async function AlunoPage() {
  const user = await requireRole("aluno");
  return <JEDShell user={user} role="aluno" title="ORFEU ONLINE / ALUNO" subtitle="empresa • persona • canvas • decisões • insumos • indicadores • turmas"><StudentWorkspace user={user} /></JEDShell>;
}
