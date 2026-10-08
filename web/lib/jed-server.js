const MAX_BODY_BYTES = 8 * 1024 * 1024;
const REQUEST_TIMEOUT_MS = 12_000;

export class JEDServerError extends Error {
  constructor(message, status = 502, code = "upstream_error") {
    super(message);
    this.name = "JEDServerError";
    this.status = status;
    this.code = code;
  }
}

export function jedServerURL() {
  let raw = String(process.env.JED_SERVER_URL || "").trim().replace(/\/+$/, "");
  if (!raw) {
    throw new JEDServerError(
      "JED Servidor não configurado. Defina JED_SERVER_URL na Vercel.",
      503,
      "server_not_configured",
    );
  }

  if (!/^https?:\/\//i.test(raw)) {
    raw = `https://${raw}`;
  }

  let parsed;
  try {
    parsed = new URL(raw);
  } catch {
    throw new JEDServerError("JED_SERVER_URL é inválida.", 503, "invalid_server_url");
  }

  if (process.env.NODE_ENV === "production" && parsed.protocol !== "https:") {
    throw new JEDServerError(
      "JED_SERVER_URL deve usar HTTPS em produção.",
      503,
      "https_required",
    );
  }

  return parsed.toString().replace(/\/$/, "");
}

async function readResponseBody(response) {
  const contentLength = Number(response.headers.get("content-length") || 0);
  if (contentLength > MAX_BODY_BYTES) {
    throw new JEDServerError("Resposta do JED Servidor excedeu o limite permitido.", 502);
  }

  const text = await response.text();
  if (text.length > MAX_BODY_BYTES) {
    throw new JEDServerError("Resposta do JED Servidor excedeu o limite permitido.", 502);
  }
  if (!text) return null;

  try {
    return JSON.parse(text);
  } catch {
    throw new JEDServerError("JED Servidor retornou JSON inválido.", 502);
  }
}

export async function jedServerRequest(path, { method = "GET", token = "", body } = {}) {
  const base = jedServerURL();
  const headers = {
    Accept: "application/json",
  };

  if (body !== undefined) {
    headers["Content-Type"] = "application/json";
  }
  if (token) {
    headers.Authorization = `Bearer ${token}`;
  }

  let response;
  try {
    response = await fetch(`${base}${path}`, {
      method,
      headers,
      body: body === undefined ? undefined : JSON.stringify(body),
      cache: "no-store",
      signal: AbortSignal.timeout(REQUEST_TIMEOUT_MS),
    });
  } catch (error) {
    if (error?.name === "TimeoutError") {
      throw new JEDServerError("Tempo esgotado ao conectar ao JED Servidor.", 504, "server_timeout");
    }
    throw new JEDServerError("Não foi possível conectar ao JED Servidor.", 502, "server_unreachable");
  }

  const payload = await readResponseBody(response);
  if (!response.ok) {
    const message = payload?.error || `JED Servidor respondeu HTTP ${response.status}.`;
    throw new JEDServerError(message, response.status, "server_rejected_request");
  }
  return payload;
}
