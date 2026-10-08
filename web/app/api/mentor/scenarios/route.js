import { NextResponse } from "next/server";
import { currentUser, sessionToken } from "@/lib/auth-server";
import { JEDServerError, jedServerRequest } from "@/lib/jed-server";
import { normalizeRole } from "@/lib/roles";

export const runtime = "nodejs";
export const dynamic = "force-dynamic";

const fallbackScenarios = [
  ["base-estavel", "Mercado estável", 1, 1, 0.08, 0, "media", 1, "intermediario"],
  ["base-aquecido", "Mercado aquecido", 1.10, 1.04, 0.10, -0.05, "media", 1, "intermediario"],
  ["base-desaceleracao", "Desaceleração econômica", 0.90, 0.92, 0.12, 0.08, "media", 1, "intermediario"],
  ["base-concorrencia", "Concorrência intensa", 0.96, 0.94, 0.10, 0.05, "alta", 1.12, "avancado"],
].map(([id, nome, alcance, conversao, oscilacao, evento_negativo_extra, concorrencia_nivel, concorrencia_indice, dificuldade]) => ({
  id, builtin: true, scenario: { nome, alcance, conversao, oscilacao, evento_negativo_extra, duracao: 12, concorrencia_nivel, concorrencia_indice, dificuldade },
}));

async function requireMentor() {
  const user = await currentUser();
  if (!user) throw new JEDServerError("Sessão inválida ou expirada.", 401);
  if (normalizeRole(user.role) !== "mentor") throw new JEDServerError("Acesso permitido apenas para Mentores.", 403);
  return user;
}

function fail(error, fallback) {
  const status = error instanceof JEDServerError ? error.status : 500;
  return NextResponse.json({ error: error instanceof Error ? error.message : fallback }, { status, headers: { "Cache-Control": "private, no-store" } });
}

export async function GET() {
  try {
    await requireMentor();
    const token = await sessionToken();
    const scenarios = await jedServerRequest("/api/v1/scenarios", { token });
    return NextResponse.json({ scenarios: Array.isArray(scenarios) ? scenarios : [] }, { headers: { "Cache-Control": "private, no-store" } });
  } catch (error) {
    if (error instanceof JEDServerError && error.status === 404) {
      return NextResponse.json({ scenarios: fallbackScenarios, server_upgrade_required: true }, { headers: { "Cache-Control": "private, no-store" } });
    }
    return fail(error, "Falha ao carregar cenários.");
  }
}

export async function POST(request) {
  try {
    await requireMentor();
    const input = await request.json();
    const name = String(input?.nome || "").trim();
    if (!name) return NextResponse.json({ error: "Informe o nome do cenário." }, { status: 400 });
    const token = await sessionToken();
    const created = await jedServerRequest("/api/v1/scenarios", {
      method: "POST",
      token,
      body: {
        nome: name,
        alcance: Number(input?.alcance || 1),
        conversao: Number(input?.conversao || 1),
        oscilacao: Number(input?.oscilacao || 0.08),
        evento_negativo_extra: Number(input?.evento_negativo_extra || 0),
        duracao: Math.max(1, Math.trunc(Number(input?.duracao || 12))),
        concorrencia_nivel: String(input?.concorrencia_nivel || "media"),
        concorrencia_indice: Number(input?.concorrencia_indice || 1),
        dificuldade: String(input?.dificuldade || "intermediario"),
        observacoes: String(input?.observacoes || ""),
      },
    });
    return NextResponse.json({ scenario: created }, { status: 201, headers: { "Cache-Control": "private, no-store" } });
  } catch (error) {
    return fail(error, "Falha ao criar cenário.");
  }
}
