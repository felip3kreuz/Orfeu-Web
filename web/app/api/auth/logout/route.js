import { NextResponse } from "next/server";
import { sessionToken } from "@/lib/auth-server";
import { SESSION_COOKIE, sessionCookieOptions } from "@/lib/session";
import { jedServerRequest } from "@/lib/jed-server";

export const runtime = "nodejs";

export async function POST() {
  const token = await sessionToken();
  if (token) {
    try {
      await jedServerRequest("/api/v1/logout", { method: "POST", token });
    } catch {
      // A sessão local deve ser encerrada mesmo se o servidor estiver indisponível.
    }
  }

  const response = NextResponse.json({ ok: true });
  response.headers.set("Cache-Control", "private, no-store");
  response.cookies.set(SESSION_COOKIE, "", { ...sessionCookieOptions(), maxAge: 0 });
  return response;
}
