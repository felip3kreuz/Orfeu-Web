import { NextResponse } from "next/server";
import { currentUser, sessionToken } from "@/lib/auth-server";
import { JEDServerError, jedServerRequest } from "@/lib/jed-server";
import { normalizeRole } from "@/lib/roles";

export const runtime = "nodejs";

export async function POST(request) {
  try {
    const user = await currentUser();
    if (!user) throw new JEDServerError("Sessão inválida ou expirada.", 401);
    if (normalizeRole(user.role) !== "mentor") throw new JEDServerError("Acesso permitido apenas para Mentores.", 403);
    const input = await request.json();
    const token = await sessionToken();
    const invitation = await jedServerRequest("/api/v1/invitations", {
      method: "POST",
      token,
      body: {
        class_id: String(input?.class_id || ""),
        name: String(input?.name || "").trim(),
        email: String(input?.email || "").trim().toLowerCase(),
        institutional_id: String(input?.institutional_id || "").trim(),
      },
    });
    return NextResponse.json({ invitation }, { status: 201, headers: { "Cache-Control": "private, no-store" } });
  } catch (error) {
    const status = error instanceof JEDServerError ? error.status : 500;
    return NextResponse.json({ error: error instanceof Error ? error.message : "Falha ao criar convite." }, { status });
  }
}
