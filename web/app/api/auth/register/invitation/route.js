import { NextResponse } from "next/server";
export async function POST() { return NextResponse.json({ error: "Ativação por convite foi desativada. Contas são cadastradas por Administradores." }, { status: 403 }); }
