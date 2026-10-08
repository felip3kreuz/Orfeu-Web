import { NextResponse } from "next/server";
import { currentUser, sessionToken } from "@/lib/auth-server";
import { JEDServerError, jedServerRequest } from "@/lib/jed-server";
import { normalizeRole } from "@/lib/roles";

export const runtime = "nodejs";
export const dynamic = "force-dynamic";

export async function POST(request) {
  try {
    const user = await currentUser();
    if (!user) throw new JEDServerError("Sessão inválida ou expirada.", 401);
    if (normalizeRole(user.role) !== "mentor") throw new JEDServerError("Acesso permitido apenas para Mentores.", 403);

    const input = await request.json();
    const companyID = String(input?.company_id || "").trim();
    const status = String(input?.status || "").trim().toLowerCase();
    const comment = String(input?.comment || "").trim();
    const withReservations = Boolean(input?.com_ressalvas);
    const requestID = String(input?.request_id || "").trim();
    if (!companyID) return NextResponse.json({ error: "Empresa não informada." }, { status: 400 });
    if (!['aprovado', 'reprovado'].includes(status)) return NextResponse.json({ error: "Classificação deve ser APROVADO ou REPROVADO." }, { status: 400 });
    if (withReservations && status !== "aprovado") return NextResponse.json({ error: "Ressalvas requerem aprovação." }, { status: 400 });
    if ((withReservations || status === "reprovado") && !comment) return NextResponse.json({ error: "Esta classificação exige um comentário." }, { status: 400 });
    if ([...comment].length > 4000) return NextResponse.json({ error: "Comentário deve ter no máximo 4000 caracteres." }, { status: 400 });

    const token = await sessionToken();
    const company = await jedServerRequest("/api/v1/mentor/companies/evaluate", {
      method: "POST",
      token,
      body: { company_id: companyID, status, comment, com_ressalvas: withReservations, request_id: requestID },
    });
    return NextResponse.json({ company }, { headers: { "Cache-Control": "private, no-store" } });
  } catch (error) {
    const status = error instanceof JEDServerError ? error.status : 500;
    return NextResponse.json({ error: error instanceof Error ? error.message : "Falha ao salvar avaliação." }, { status, headers: { "Cache-Control": "private, no-store" } });
  }
}
