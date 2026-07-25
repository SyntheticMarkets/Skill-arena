const API_URL = (process.env.NEXT_PUBLIC_SKILL_ARENA_API_URL || "/gateway").replace(/\/$/, "");

export function apiURL(path: string) {
  return `${API_URL}${path}`;
}

export class APIError extends Error {
  status: number;
  code: string;

  constructor(status: number, code: string, message: string) {
    super(message);
    this.status = status;
    this.code = code;
  }
}

export async function api<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers);
  if (init.body && !headers.has("Content-Type")) headers.set("Content-Type", "application/json");
  const response = await fetch(apiURL(path), {
    ...init,
    headers,
    credentials: "include",
    cache: "no-store"
  });
  if (response.status === 204) return undefined as T;
  const contentType = response.headers.get("content-type") || "";
  const body = contentType.includes("application/json") ? await response.json() : null;
  if (!response.ok) {
    throw new APIError(response.status, body?.error?.code || "request_failed", body?.error?.message || "The operation could not be completed.");
  }
  return body as T;
}

export function money(minor: number, currency = "ZAR") {
  return new Intl.NumberFormat("en-ZA", { style: "currency", currency }).format(minor / 100);
}

export function dateTime(value?: string) {
  if (!value) return "Not recorded";
  return new Intl.DateTimeFormat("en-ZA", { dateStyle: "medium", timeStyle: "short" }).format(new Date(value));
}
