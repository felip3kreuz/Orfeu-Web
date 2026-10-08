import { NextResponse } from "next/server";
import { sessionToken } from "@/lib/auth-server";
import { JEDServerError, jedServerRequest } from "@/lib/jed-server";
import { roleHome } from "@/lib/roles";

export const runtime = "nodejs";

export async function POST(request) {
  try {
    const input = await request.json();
    const current = String(input?.current || "");
    const next = String(input?.next || "");
    if (!current || next.length < 8) return NextResponse.json({ error: "Informe a senha temporária e uma nova senha com pelo menos 8 caracteres." }, { status: 400 });
    if (next === "acbd1234") return NextResponse.json({ error: "Escolha uma senha diferente da senha temporária." }, { status: 400 });
    const token = await sessionToken();
    if (!token) return NextResponse.json({ error: "Sessão inválida ou expirada." }, { status: 401 });
    await jedServerRequest("/api/v1/password", { method: "POST", token, body: { current, new: next } });
    const user = await jedServerRequest("/api/v1/me", { token });
    return NextResponse.json({ ok: true, redirectTo: roleHome(user.role) }, { headers: { "Cache-Control": "private, no-store" } });
  } catch (error) {
    const status = error instanceof JEDServerError ? error.status : 500;
    return NextResponse.json({ error: error instanceof Error ? error.message : "Falha ao alterar senha." }, { status });
  }
}
