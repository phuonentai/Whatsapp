import { afterEach, describe, expect, it, vi } from "vitest";
import {
  applyBackendStatus,
  fetchBackendSubscriptionStatus,
  type BackendSubscriptionStatus,
} from "./current-subscription";

const polarInactive = {
  isActive: false,
  status: null,
  reason: "NO_ACTIVE_SUBSCRIPTION" as string | undefined,
};

function backend(overrides: Partial<BackendSubscriptionStatus>): BackendSubscriptionStatus {
  return {
    has_active_subscription: false,
    reason: "",
    ...overrides,
  };
}

describe("applyBackendStatus", () => {
  it("maps an active backend subscription to an active state with reason cleared", () => {
    const mapped = applyBackendStatus(polarInactive, backend({ has_active_subscription: true, reason: "ok" }));

    expect(mapped.isActive).toBe(true);
    expect(mapped.status).toBe("active");
    expect(mapped.reason).toBeUndefined();
  });

  it("maps a past_due backend reason to the dunning state", () => {
    const mapped = applyBackendStatus(
      polarInactive,
      backend({ reason: "subscription status: past_due" })
    );

    expect(mapped.isActive).toBe(false);
    expect(mapped.status).toBe("past_due");
    expect(mapped.reason).toBe("NO_ACTIVE_SUBSCRIPTION");
  });

  it("maps a no-active-subscription backend reason to status none", () => {
    const mapped = applyBackendStatus(
      polarInactive,
      backend({ reason: "No active subscription found" })
    );

    expect(mapped.isActive).toBe(false);
    expect(mapped.status).toBe("none");
    expect(mapped.reason).toBe("NO_ACTIVE_SUBSCRIPTION");
  });

  it("keeps the Polar result when the backend reports an indecisive inactive state", () => {
    const mapped = applyBackendStatus(polarInactive, backend({ reason: "invoice quota exceeded" }));

    expect(mapped).toEqual(polarInactive);
  });
});

describe("fetchBackendSubscriptionStatus", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("returns null when the backend is unreachable (graceful degradation)", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockRejectedValue(new Error("ECONNREFUSED"))
    );

    const status = await fetchBackendSubscriptionStatus("jwt");
    expect(status).toBeNull();
  });

  it("returns null on a non-OK response", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({ ok: false, status: 500 })
    );

    const status = await fetchBackendSubscriptionStatus("jwt");
    expect(status).toBeNull();
  });

  it("returns the parsed status with the session JWT bearer", async () => {
    const response = {
      ok: true,
      json: async () => ({
        has_active_subscription: true,
        reason: "ok",
        checked_at: "2026-08-11T00:00:00Z",
      }),
    };
    const fetchMock = vi.fn().mockResolvedValue(response);
    vi.stubGlobal("fetch", fetchMock);

    const status = await fetchBackendSubscriptionStatus("session-jwt");
    expect(status?.has_active_subscription).toBe(true);

    const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(url).toContain("/api/subscriptions/status");
    expect((init.headers as Record<string, string>).Authorization).toBe("Bearer session-jwt");
  });
});
