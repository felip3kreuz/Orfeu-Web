import { redirect } from "next/navigation";
import { currentUser } from "@/lib/auth-server";
import { roleHome } from "@/lib/roles";
import FirstAccessPasswordForm from "@/components/first-access-password-form";
import { JEDChrome, PublicScreenTitle } from "@/components/jed-shell";

export const dynamic = "force-dynamic";
export const metadata = { title: "Primeiro acesso · Orfeu" };

export default async function ChangePasswordPage() {
  const user = await currentUser();
  if (!user) redirect("/login");
  if (!user.must_change_password) redirect(roleHome(user.role));
  return <div className="public-orbit-page"><JEDChrome /><main className="orbit-frame"><PublicScreenTitle title="PRIMEIRO ACESSO" subtitle="substituição obrigatória da senha temporária" /><FirstAccessPasswordForm user={user} /></main></div>;
}
