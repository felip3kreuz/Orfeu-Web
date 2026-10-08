import { NextResponse } from "next/server";
import { currentUser, sessionToken } from "@/lib/auth-server";
import { JEDServerError, jedServerRequest } from "@/lib/jed-server";
import { normalizeRole } from "@/lib/roles";

export const runtime = "nodejs";
export const dynamic = "force-dynamic";

export async function GET() {
  try {
    const user = await currentUser();
    if (!user) throw new JEDServerError("Sessão inválida ou expirada.", 401);
    if (normalizeRole(user.role) !== "mentor") throw new JEDServerError("Acesso permitido apenas para Mentores.", 403);
    const token = await sessionToken();
    const [classes, companies, students] = await Promise.all([
      jedServerRequest("/api/v1/classes", { token }),
      jedServerRequest("/api/v1/companies", { token }),
      jedServerRequest("/api/v1/mentor/students", { token }),
    ]);
    return NextResponse.json({ classes, companies, students }, { headers: { "Cache-Control": "private, no-store" } });
  } catch (error) {
    const status = error instanceof JEDServerError ? error.status : 500;
    return NextResponse.json({ error: error instanceof Error ? error.message : "Falha ao carregar painel." }, { status, headers: { "Cache-Control": "private, no-store" } });
  }
}
