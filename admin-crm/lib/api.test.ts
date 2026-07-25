import { afterEach, describe, expect, it, vi } from "vitest";
import { api, APIError, apiURL, money } from "./api";

describe("admin API client", () => {
  afterEach(() => vi.restoreAllMocks());

  it("uses the dedicated same-origin gateway and includes credentials", async () => {
    const fetchMock = vi.spyOn(globalThis, "fetch").mockResolvedValue(new Response(JSON.stringify({ ok: true }), {
      status: 200,
      headers: { "Content-Type": "application/json" }
    }));
    await expect(api<{ ok: boolean }>("/api/v1/admin-crm/dashboard")).resolves.toEqual({ ok: true });
    expect(apiURL("/api/v1/admin-crm/dashboard")).toBe("/gateway/api/v1/admin-crm/dashboard");
    expect(fetchMock).toHaveBeenCalledWith("/gateway/api/v1/admin-crm/dashboard", expect.objectContaining({
      credentials: "include",
      cache: "no-store"
    }));
  });

  it("preserves structured backend errors", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValue(new Response(JSON.stringify({
      error: { code: "FORBIDDEN", message: "Permission is required." }
    }), { status: 403, headers: { "Content-Type": "application/json" } }));
    await expect(api("/api/v1/admin-crm/finance")).rejects.toEqual(
      expect.objectContaining<Partial<APIError>>({ status: 403, code: "FORBIDDEN", message: "Permission is required." })
    );
  });

  it("formats integer minor units without floating-point business logic", () => {
    expect(money(12345, "ZAR")).toContain("123");
  });
});
