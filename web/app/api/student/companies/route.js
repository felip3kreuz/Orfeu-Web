import { NextResponse } from "next/server";
import { currentUser, sessionToken } from "@/lib/auth-server";
import { JEDServerError, jedServerRequest } from "@/lib/jed-server";
import { normalizeRole } from "@/lib/roles";

export const runtime = "nodejs";
export const dynamic = "force-dynamic";

function fail(error, fallback) {
  const status = error instanceof JEDServerError ? error.status : 500;
  const response = NextResponse.json({ error: error instanceof Error ? error.message : fallback }, { status });
  response.headers.set("Cache-Control", "private, no-store");
  return response;
}

async function requireStudent() {
  const user = await currentUser();
  if (!user) throw new JEDServerError("Sessão inválida ou expirada.", 401);
  if (normalizeRole(user.role) !== "aluno") throw new JEDServerError("Acesso permitido apenas para Alunos.", 403);
  return user;
}

export async function GET() {
  try {
    await requireStudent();
    const token = await sessionToken();
    const companies = await jedServerRequest("/api/v1/companies", { token });
    const response = NextResponse.json({ companies: Array.isArray(companies) ? companies : [] });
    response.headers.set("Cache-Control", "private, no-store");
    return response;
  } catch (error) {
    return fail(error, "Falha ao carregar empresas.");
  }
}

export async function PUT(request) {
  try {
    await requireStudent();
    const input = await request.json();
    const localID = String(input?.local_id || input?.company?.local_id || "").trim();
    if (!localID || localID.includes("/")) {
      return NextResponse.json({ error: "ID local da empresa inválido." }, { status: 400 });
    }
    const token = await sessionToken();
    const remote = await jedServerRequest(`/api/v1/companies/${encodeURIComponent(localID)}`, {
      method: "PUT",
      token,
      body: { class_id: String(input?.class_id || ""), company: input?.company || {} },
    });
    const response = NextResponse.json({ company: remote });
    response.headers.set("Cache-Control", "private, no-store");
    return response;
  } catch (error) {
    return fail(error, "Falha ao sincronizar empresa.");
  }
}

// OX-78-1: remoção autenticada de empreendimento do próprio aluno.
export async function DELETE(request) {
  try {
    await requireStudent();
    const input = await request.json();
    const localID = String(input?.local_id || "").trim();
    if (!localID || localID.includes("/")) {
      return NextResponse.json({ error: "Identificador de empreendimento inválido." }, { status: 400 });
    }
    const token = await sessionToken();
    const result = await jedServerRequest(`/api/v1/companies/${encodeURIComponent(localID)}`, { method: "DELETE", token });
    const response = NextResponse.json(result, { headers: { "Cache-Control": "private, no-store" } });
    return response;
  } catch (error) {
    return fail(error, "Falha ao excluir empreendimento.");
  }
}
