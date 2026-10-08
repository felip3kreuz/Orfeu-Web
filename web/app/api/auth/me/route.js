import { NextResponse } from "next/server";
import { currentUser } from "@/lib/auth-server";
import { SESSION_COOKIE, sessionCookieOptions } from "@/lib/session";
import { JEDServerError } from "@/lib/jed-server";

export const runtime = "nodejs";

export async function GET() {
  try {
    const user = await currentUser();
    if (!user) {
      const response = NextResponse.json({ error: "Sessão inválida ou expirada." }, { status: 401 });
      response.cookies.set(SESSION_COOKIE, "", { ...sessionCookieOptions(), maxAge: 0 });
      return response;
    }

    const response = NextResponse.json({ user });
    response.headers.set("Cache-Control", "private, no-store");
    return response;
  } catch (error) {
    const status = error instanceof JEDServerError ? error.status : 500;
    return NextResponse.json(
      { error: error instanceof Error ? error.message : "Falha ao consultar a sessão." },
      { status },
    );
  }
}
