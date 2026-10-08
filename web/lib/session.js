export const SESSION_COOKIE = "jed_server_session";
export const SESSION_MAX_AGE = 12 * 60 * 60;

export function sessionCookieOptions() {
  return {
    httpOnly: true,
    sameSite: "lax",
    secure: process.env.NODE_ENV === "production",
    path: "/",
    maxAge: SESSION_MAX_AGE,
  };
}
