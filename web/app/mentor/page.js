import JEDShell from "@/components/jed-shell";
import MentorWorkspace from "@/components/mentor-workspace";
import { requireRole } from "@/lib/auth-server";

export const dynamic = "force-dynamic";
export const metadata = { title: "Mentor · Orfeu" };

export default async function MentorPage() {
  const user = await requireRole("mentor");
  return <JEDShell user={user} role="mentor" title="ORFEU ONLINE / MENTOR" subtitle="cenários • turmas • alunos • resultados • acompanhamento"><MentorWorkspace user={user} /></JEDShell>;
}
