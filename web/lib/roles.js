export function normalizeRole(role) {
  const value = String(role || "").trim().toLowerCase();
  return value === "tutor" ? "mentor" : value;
}

export function roleHome(role) {
  switch (normalizeRole(role)) {
    case "admin":
      return "/admin";
    case "mentor":
      return "/mentor";
    case "aluno":
      return "/aluno";
    default:
      return "/login";
  }
}

export function roleLabel(role) {
  switch (normalizeRole(role)) {
    case "admin":
      return "Administrador";
    case "mentor":
      return "Mentor";
    case "aluno":
      return "Aluno";
    default:
      return "Usuário";
  }
}
