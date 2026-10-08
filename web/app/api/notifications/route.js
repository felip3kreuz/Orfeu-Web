import { NextResponse } from "next/server";
import { currentUser, sessionToken } from "@/lib/auth-server";
import { jedServerRequest, JEDServerError } from "@/lib/jed-server";

export const runtime = "nodejs";
export const dynamic = "force-dynamic";

export async function GET() {
  try {
    if (!await currentUser()) throw new JEDServerError("Sessão inválida.", 401);
    const token = await sessionToken();
    const notifications = await jedServerRequest("/api/v1/notifications", { token });
    return NextResponse.json({ notifications }, { headers: { "Cache-Control": "private, no-store" } });
  } catch (error) {
    return NextResponse.json({ error: error.message || "Falha ao consultar notificações." }, { status: error instanceof JEDServerError ? error.status : 500 });
  }
}

export async function POST(request) {
  try {
    if (!await currentUser()) throw new JEDServerError("Sessão inválida.", 401);
    const { notification_id: id } = await request.json();
    if (typeof id !== "string" || !/^nt-[a-f0-9]{24}$/.test(id)) return NextResponse.json({ error: "ID inválido." }, { status: 400 });
    const token = await sessionToken();
    const notification = await jedServerRequest(`/api/v1/notifications/${id}/read`, { method: "POST", token });
    return NextResponse.json({ notification }, { headers: { "Cache-Control": "private, no-store" } });
  } catch (error) {
    return NextResponse.json({ error: error.message || "Falha ao marcar como lida." }, { status: error instanceof JEDServerError ? error.status : 500 });
  }
}
