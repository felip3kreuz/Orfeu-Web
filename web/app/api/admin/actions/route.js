import { NextResponse } from "next/server";
import { currentUser, sessionToken } from "@/lib/auth-server";
import { JEDServerError, jedServerRequest } from "@/lib/jed-server";
import { normalizeRole } from "@/lib/roles";

export const runtime = "nodejs";

export async function POST(request) {
  try {
    const user = await currentUser();
    if (!user) throw new JEDServerError("Sessão inválida ou expirada.", 401);
    if (normalizeRole(user.role) !== "admin") throw new JEDServerError("Acesso permitido apenas para Administradores.", 403);
    const input = await request.json();
    const token = await sessionToken();
    const action = String(input?.action || "");
    let path;
    let body;

    switch (action) {
      case "user_status":
        path = "/api/v1/admin/user-status";
        body = { user_id: input.user_id, status: input.status };
        break;
      case "transfer_primary":
        path = "/api/v1/admin/transfer-primary";
        body = { user_id: input.user_id };
        break;
      case "resend_email":
        path = "/api/v1/admin/users/resend-email";
        body = { user_id: input.user_id };
        break;
      case "create_user":
        path = "/api/v1/admin/users/create";
        body = {
          name: input.name,
          email: input.email,
          role: input.role,
          institution: input.institution,
          institutional_id: input.institutional_id,
          class_id: input.class_id,
        };
        break;
      case "import_users":
        path = "/api/v1/admin/users/import";
        body = { users: Array.isArray(input.users) ? input.users : [] };
        break;
      case "create_class":
        path = "/api/v1/admin/classes";
        body = { name: input.name, mentor_id: input.mentor_id };
        break;
      case "assign_student_class":
        path = "/api/v1/admin/students/assign-class";
        body = { user_id: input.user_id, class_id: input.class_id || "" };
        break;
      case "delete_user":
        path = "/api/v1/admin/users/delete";
        body = { user_id: input.user_id };
        break;
      case "delete_class":
        path = "/api/v1/admin/classes/delete";
        body = { class_id: input.class_id };
        break;
      default:
        return NextResponse.json({ error: "Ação administrativa inválida." }, { status: 400 });
    }

    const result = await jedServerRequest(path, { method: "POST", token, body });
    return NextResponse.json({ result }, { headers: { "Cache-Control": "private, no-store" } });
  } catch (error) {
    const status = error instanceof JEDServerError ? error.status : 500;
    return NextResponse.json({ error: error instanceof Error ? error.message : "Falha na operação administrativa." }, { status });
  }
}
