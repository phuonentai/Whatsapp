import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("@/lib/auth/stytch/server", () => ({
  getMemberSession: vi.fn(),
}));
vi.mock("@/lib/auth/server-permissions", () => ({
  getServerPermissions: vi.fn(),
}));

import { getMemberSession } from "@/lib/auth/stytch/server";
import { getServerPermissions } from "@/lib/auth/server-permissions";
import { createMercadoPagoCheckout } from "./create-mp-checkout";

const sessionMock = vi.mocked(getMemberSession);
const permissionsMock = vi.mocked(getServerPermissions);

const ORIGINAL_ENV = {
  NEXT_PUBLIC_MERCADOPAGO_PLAN_ID: process.env.NEXT_PUBLIC_MERCADOPAGO_PLAN_ID,
  NEXT_PUBLIC_MERCADOPAGO_CHECKOUT_PLAN_ID:
    process.env.NEXT_PUBLIC_MERCADOPAGO_CHECKOUT_PLAN_ID,
  NEXT_PUBLIC_APP_BASE_URL: process.env.NEXT_PUBLIC_APP_BASE_URL,
};

function bodyOf(call: [string, RequestInit]) {
  return JSON.parse(call[1].body as string);
}

describe("createMercadoPagoCheckout", () => {
  beforeEach(() => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: true,
        json: async () => ({
          checkoutUrl: "https://checkout.mercadopago.com/preapproval/xyz",
          message: "Checkout initiated",
          checkedAt: "2026-08-11T00:00:00Z",
        }),
      })
    );
    sessionMock.mockResolvedValue({ session_jwt: "jwt" } as never);
    permissionsMock.mockResolvedValue({ canManageSubscriptions: true } as never);

    process.env.NEXT_PUBLIC_MERCADOPAGO_PLAN_ID = "plan-gate";
    delete process.env.NEXT_PUBLIC_MERCADOPAGO_CHECKOUT_PLAN_ID;
    process.env.NEXT_PUBLIC_APP_BASE_URL = "https://app.example.com";
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
    for (const [key, value] of Object.entries(ORIGINAL_ENV)) {
      if (value === undefined) {
        delete process.env[key];
      } else {
        process.env[key] = value;
      }
    }
  });

  it("prefers NEXT_PUBLIC_MERCADOPAGO_CHECKOUT_PLAN_ID over params.planId", async () => {
    process.env.NEXT_PUBLIC_MERCADOPAGO_CHECKOUT_PLAN_ID = "plan-env";

    await createMercadoPagoCheckout({ planId: "plan-param" });

    const fetchMock = vi.mocked(fetch);
    expect(bodyOf(fetchMock.mock.calls[0] as [string, RequestInit]).plan_id).toBe("plan-env");
  });

  it("falls back to params.planId when the env plan id is unset", async () => {
    await createMercadoPagoCheckout({ planId: "plan-param" });

    const fetchMock = vi.mocked(fetch);
    expect(bodyOf(fetchMock.mock.calls[0] as [string, RequestInit]).plan_id).toBe("plan-param");
  });

  it("falls back to \"default\" when neither env nor param provides a plan id", async () => {
    await createMercadoPagoCheckout();

    const fetchMock = vi.mocked(fetch);
    expect(bodyOf(fetchMock.mock.calls[0] as [string, RequestInit]).plan_id).toBe("default");
  });

  it("sends an explicit app-origin return URL pointing at the subscription view", async () => {
    await createMercadoPagoCheckout({ planId: "plan-param" });

    const fetchMock = vi.mocked(fetch);
    const body = bodyOf(fetchMock.mock.calls[0] as [string, RequestInit]);
    expect(body.back_url).toBe(
      "https://app.example.com/dashboard/settings?view=subscription"
    );
  });
});
