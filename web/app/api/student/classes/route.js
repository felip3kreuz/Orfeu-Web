import { NextResponse } from "next/server";
import { currentUser, sessionToken } from "@/lib/auth-server";
import { JEDServerError, jedServerRequest } from "@/lib/jed-server";
import { normalizeRole } from "@/lib/roles";

export const runtime = "nodejs";
export const dynamic = "force-dynamic";

async function actor() {
  const user = await currentUser();
  if (!user) throw new JEDServerError("Sessão inválida ou expirada.", 401);
  if (normalizeRole(user.role) !== "aluno") throw new JEDServerError("Acesso permitido apenas para Alunos.", 403);
  return user;
}

function errorResponse(error, fallback) {
  const status = error instanceof JEDServerError ? error.status : 500;
  const response = NextResponse.json({ error: error instanceof Error ? error.message : fallback }, { status });
  response.headers.set("Cache-Control", "private, no-store");
  return response;
}

export async function GET() {
  try {
    await actor();
    const token = await sessionToken();
    const classes = await jedServerRequest("/api/v1/classes", { token });
    return NextResponse.json({ classes: Array.isArray(classes) ? classes : [] }, { headers: { "Cache-Control": "private, no-store" } });
  } catch (error) {
    return errorResponse(error, "Falha ao carregar turmas.");
  }
}

export async function POST() {
  return NextResponse.json({ error: "O vínculo com turmas é realizado exclusivamente pelo Administrador." }, { status: 403, headers: { "Cache-Control": "private, no-store" } });
}
