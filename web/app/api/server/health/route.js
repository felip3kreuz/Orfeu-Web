import { NextResponse } from "next/server";
import { JEDServerError, jedServerRequest } from "@/lib/jed-server";

export const runtime = "nodejs";

export async function GET() {
  try {
    const upstream = await jedServerRequest("/health");
    return NextResponse.json({ ok: true, upstream });
  } catch (error) {
    const status = error instanceof JEDServerError ? error.status : 500;
    return NextResponse.json(
      {
        ok: false,
        error: error instanceof Error ? error.message : "JED Servidor indisponível.",
      },
      { status },
    );
  }
}
