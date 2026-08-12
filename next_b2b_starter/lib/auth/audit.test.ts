import { afterEach, beforeEach, describe, expect, it, vi, type Mock } from "vitest";

import { cookies } from "next/headers";

import {
  AUTH_AUDIT_EVENT_TYPES,
  AUTH_AUDIT_DETAILS,
  mapAuthErrorToDetail,
  recordAuthAudit,
  type AuthAuditDetail,
} from "@/lib/auth/audit";

vi.mock("next/headers", () => ({
  cookies: vi.fn(),
}));

const mockCookies = vi.mocked(cookies);
const mockCookieGet = vi.fn();

const SESSION_JWT = "header.payload.signature";

function cookieStoreWith(value: string | null) {
  mockCookieGet.mockReturnValue(value ? { value } : undefined);
  return { get: mockCookieGet };
}

describe("recordAuthAudit", () => {
  let fetchMock: Mock;

  beforeEach(() => {
    fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      status: 201,
      json: async () => ({}),
    });
    vi.stubGlobal("fetch", fetchMock);
    vi.spyOn(console, "warn").mockImplementation(() => {});
    vi.spyOn(console, "info").mockImplementation(() => {});
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
  });

  it("posts a tipo=sistema activity to the Go activities endpoint for every event type", async () => {
    for (const type of AUTH_AUDIT_EVENT_TYPES) {
      fetchMock.mockClear();
      mockCookies.mockResolvedValue(
        cookieStoreWith(SESSION_JWT) as never
      );

      await recordAuthAudit({
        type,
        memberId: "member-123",
        organizationId: "org-456",
      });

      expect(fetchMock).toHaveBeenCalledTimes(1);
      const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit];
      expect(url).toBe("http://localhost:8080/api/crm/actividades");
      expect(init.method).toBe("POST");
      expect(init.headers).toMatchObject({
        "Content-Type": "application/json",
        Authorization: `Bearer ${SESSION_JWT}`,
      });

      const body = JSON.parse(String(init.body));
      expect(body.tipo).toBe("sistema");
      expect(typeof body.asunto).toBe("string");
      expect(body.asunto.length).toBeGreaterThan(0);
      expect(body.contenido).toBe(JSON.stringify({ type }));
    }
  });

  it("includes the bounded detail in the payload and content when provided", async () => {
    mockCookies.mockResolvedValue(cookieStoreWith(SESSION_JWT) as never);

    await recordAuthAudit({
      type: "login_failed",
      memberId: "member-123",
      organizationId: "org-456",
      detail: "invalid_token",
    });

    const body = JSON.parse(
      String(fetchMock.mock.calls[0][1].body)
    ) as Record<string, unknown>;
    expect(body.metadata).toEqual({
      type: "login_failed",
      member_id: "member-123",
      organization_id: "org-456",
      detail: "invalid_token",
    });
    expect(body.contenido).toBe(
      JSON.stringify({ type: "login_failed", detail: "invalid_token" })
    );
  });

  it("audit payload contains ONLY type, member_id, organization_id, detail", async () => {
    mockCookies.mockResolvedValue(cookieStoreWith(SESSION_JWT) as never);

    await recordAuthAudit({
      type: "logout",
      memberId: "member-123",
      organizationId: "org-456",
    });

    const body = JSON.parse(
      String(fetchMock.mock.calls[0][1].body)
    ) as Record<string, unknown>;
    expect(Object.keys(body.metadata as object).sort()).toEqual([
      "member_id",
      "organization_id",
      "type",
    ]);
  });

  it("never includes token or session material in the payload", async () => {
    mockCookies.mockResolvedValue(cookieStoreWith(SESSION_JWT) as never);

    await recordAuthAudit({
      type: "login_succeeded",
      memberId: "member-123",
      organizationId: "org-456",
    });

    const rawBody = String(fetchMock.mock.calls[0][1].body);

    expect(rawBody).not.toContain(SESSION_JWT);
    expect(rawBody).not.toContain("signature");
    expect(rawBody).not.toContain("Bearer");
    expect(rawBody).not.toMatch(/token|jwt|session|password|secret/i);
  });

  it("swallows network failures (no throw) and logs a warning", async () => {
    mockCookies.mockResolvedValue(cookieStoreWith(SESSION_JWT) as never);
    fetchMock.mockRejectedValue(new Error("ECONNREFUSED"));

    await expect(
      recordAuthAudit({
        type: "logout",
        memberId: "member-123",
        organizationId: "org-456",
      })
    ).resolves.toBeUndefined();
    expect(console.warn).toHaveBeenCalled();
  });

  it("swallows non-2xx API responses (no throw) and logs a warning", async () => {
    mockCookies.mockResolvedValue(cookieStoreWith(SESSION_JWT) as never);
    fetchMock.mockResolvedValue({ ok: false, status: 500 });

    await expect(recordAuthAudit({ type: "login_succeeded" })).resolves.toBeUndefined();
    expect(console.warn).toHaveBeenCalled();
  });

  it("skips the POST (no throw) when no member session JWT is available", async () => {
    mockCookies.mockResolvedValue(cookieStoreWith(null) as never);

    await expect(
      recordAuthAudit({ type: "magic_link_requested", organizationId: "org-456" })
    ).resolves.toBeUndefined();
    expect(fetchMock).not.toHaveBeenCalled();
    expect(console.warn).toHaveBeenCalled();
  });

  it("drops a detail that is not part of the bounded enum", async () => {
    mockCookies.mockResolvedValue(cookieStoreWith(SESSION_JWT) as never);

    await recordAuthAudit({
      type: "login_failed",
      detail: "some.raw.error.body" as AuthAuditDetail,
    });

    const body = JSON.parse(
      String(fetchMock.mock.calls[0][1].body)
    ) as Record<string, unknown>;
    expect(body.metadata).toEqual({ type: "login_failed" });
    expect(body.contenido).toBe(JSON.stringify({ type: "login_failed" }));
  });
});

describe("mapAuthErrorToDetail", () => {
  it("maps known Stytch error codes to bounded details", () => {
    expect(mapAuthErrorToDetail({ error_type: "invalid_token" })).toBe("invalid_token");
    expect(mapAuthErrorToDetail({ error_type: "expired_token" })).toBe("expired_token");
    expect(mapAuthErrorToDetail({ error_type: "mfa_required" })).toBe("mfa_required");
    expect(mapAuthErrorToDetail({ error_type: "wrong_origin" })).toBe("wrong_origin");
  });

  it("collapses unknown/raw error bodies to internal_error", () => {
    expect(mapAuthErrorToDetail({ error_type: "unexpected_thing" })).toBe("internal_error");
    expect(mapAuthErrorToDetail(new Error("boom"))).toBe("internal_error");
    expect(mapAuthErrorToDetail("raw string")).toBe("internal_error");
    expect(mapAuthErrorToDetail(undefined)).toBe("internal_error");
  });

  it("exposes only the documented bounded details", () => {
    expect([...AUTH_AUDIT_DETAILS]).toEqual([
      "invalid_token",
      "expired_token",
      "mfa_required",
      "invalid_organization",
      "wrong_origin",
      "internal_error",
    ]);
  });
});
