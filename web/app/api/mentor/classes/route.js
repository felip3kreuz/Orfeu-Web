import { NextResponse } from "next/server";
import { currentUser, sessionToken } from "@/lib/auth-server";
import { JEDServerError, jedServerRequest } from "@/lib/jed-server";
import { normalizeRole } from "@/lib/roles";

export const runtime = "nodejs";

async function mentorActor() {
  const user = await currentUser();
  if (!user) throw new JEDServerError("Sessão inválida ou expirada.", 401);
  if (normalizeRole(user.role) !== "mentor") throw new JEDServerError("Acesso permitido apenas para Mentores.", 403);
  return user;
}

export async function POST() {
  return NextResponse.json({ error: "As turmas são criadas e nomeadas exclusivamente pelo Administrador." }, { status: 403 });
}

export async function PUT(request) {
  try {
    await mentorActor();
    const input = await request.json();
    const classID = String(input?.class_id || "").trim();
    if (!classID) return NextResponse.json({ error: "Turma inválida." }, { status: 400 });
    if (!input?.scenario?.nome) return NextResponse.json({ error: "Selecione um cenário." }, { status: 400 });
    const token = await sessionToken();
    const updated = await jedServerRequest(`/api/v1/classes/${encodeURIComponent(classID)}/scenario`, { method: "POST", token, body: input.scenario });
    return NextResponse.json({ class: updated }, { headers: { "Cache-Control": "private, no-store" } });
  } catch (error) {
    const status = error instanceof JEDServerError ? error.status : 500;
    return NextResponse.json({ error: error instanceof Error ? error.message : "Falha ao atualizar cenário da turma." }, { status });
  }
}
