import { NextResponse } from "next/server";
export async function POST() { return NextResponse.json({ error: "Cadastro de Mentor é realizado exclusivamente por Administradores." }, { status: 403 }); }
