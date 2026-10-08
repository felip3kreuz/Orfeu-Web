import { cookies } from "next/headers";
import { redirect } from "next/navigation";
import { JEDServerError, jedServerRequest } from "@/lib/jed-server";
import { normalizeRole, roleHome } from "@/lib/roles";
import { SESSION_COOKIE } from "@/lib/session";

export async function sessionToken() {
  const store = await cookies();
  return store.get(SESSION_COOKIE)?.value || "";
}

export async function currentUser() {
  const token = await sessionToken();
  if (!token) return null;

  try {
    const user = await jedServerRequest("/api/v1/me", { token });
    return { ...user, role: normalizeRole(user?.role) };
  } catch (error) {
    if (error instanceof JEDServerError && (error.status === 401 || error.status === 403)) {
      return null;
    }
    throw error;
  }
}

export async function requireRole(expectedRole) {
  const user = await currentUser();
  if (!user) redirect("/login");

  if (user.must_change_password) redirect("/alterar-senha");

  if (user.must_change_password) redirect("/alterar-senha");

  const expected = normalizeRole(expectedRole);
  if (user.role !== expected) {
    redirect(roleHome(user.role));
  }
  return user;
}
