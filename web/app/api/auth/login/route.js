import { NextResponse } from "next/server";
import { SESSION_COOKIE, sessionCookieOptions } from "@/lib/session";
import { JEDServerError, jedServerRequest } from "@/lib/jed-server";
import { normalizeRole, roleHome } from "@/lib/roles";

export const runtime = "nodejs";

export async function POST(request) {
  let input;
  try {
    input = await request.json();
  } catch {
    return NextResponse.json({ error: "JSON inválido." }, { status: 400 });
  }

  const email = String(input?.email || "").trim().toLowerCase();
  const password = String(input?.password || "");
  if (!email || !password) {
    return NextResponse.json({ error: "Informe e-mail e senha." }, { status: 400 });
  }

  try {
    const login = await jedServerRequest("/api/v1/login", {
      method: "POST",
      body: { email, password },
    });

    const user = { ...login.user, role: normalizeRole(login.user?.role) };
    const response = NextResponse.json({ user, redirectTo: user.must_change_password ? "/alterar-senha" : roleHome(user.role) });
    response.headers.set("Cache-Control", "private, no-store");
    response.cookies.set(SESSION_COOKIE, login.token, sessionCookieOptions());
    return response;
  } catch (error) {
    const status = error instanceof JEDServerError ? error.status : 500;
    const message = error instanceof Error ? error.message : "Falha ao entrar.";
    return NextResponse.json({ error: message }, { status });
  }
}
