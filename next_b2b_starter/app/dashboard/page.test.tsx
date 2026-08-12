import { describe, expect, it, vi, beforeEach } from "vitest";

const mocks = vi.hoisted(() => ({
  redirect: vi.fn(),
  verifyPayment: vi.fn(),
  verifyMercadoPagoPayment: vi.fn(),
}));

vi.mock("next/navigation", () => ({
  redirect: (url: string) => {
    mocks.redirect(url);
    throw new Error("NEXT_REDIRECT");
  },
}));

vi.mock("@/lib/actions/billing/verify-payment", () => ({
  verifyPayment: (...args: unknown[]) => mocks.verifyPayment(...args),
}));

vi.mock("@/lib/actions/billing/verify-mp-payment", () => ({
  verifyMercadoPagoPayment: (...args: unknown[]) =>
    mocks.verifyMercadoPagoPayment(...args),
}));

vi.mock("./components/dashboard-home", () => ({
  DashboardHome: () => null,
}));

import DashboardPage from "./page";

function pageWith(params: Record<string, string>) {
  return DashboardPage({ searchParams: Promise.resolve(params) });
}

describe("DashboardPage checkout callback routing", () => {
  beforeEach(() => {
    mocks.redirect.mockClear();
    mocks.verifyPayment.mockReset();
    mocks.verifyMercadoPagoPayment.mockReset();
  });

  it("routes a preapproval-only return to the subscription view without a banner or verification", async () => {
    await expect(pageWith({ preapproval_id: "pre-123" })).rejects.toThrow(
      "NEXT_REDIRECT"
    );

    expect(mocks.redirect).toHaveBeenCalledTimes(1);
    expect(mocks.redirect).toHaveBeenCalledWith("/dashboard/settings?view=subscription");
    expect(mocks.verifyMercadoPagoPayment).not.toHaveBeenCalled();
    expect(mocks.verifyPayment).not.toHaveBeenCalled();
  });

  it("still verifies a payment when preapproval_id arrives alongside payment_id", async () => {
    mocks.verifyMercadoPagoPayment.mockResolvedValue({
      success: true,
      data: { has_active_subscription: true },
    });

    await expect(
      pageWith({ preapproval_id: "pre-123", payment_id: "pay-456" })
    ).rejects.toThrow("NEXT_REDIRECT");

    expect(mocks.verifyMercadoPagoPayment).toHaveBeenCalledWith({ paymentId: "pay-456" });
    expect(mocks.redirect).toHaveBeenCalledWith(
      "/dashboard/settings?view=subscription&payment_verified=true"
    );
  });

  it("verifies a payment_id without a preapproval_id", async () => {
    mocks.verifyMercadoPagoPayment.mockResolvedValue({
      success: false,
      error: "verification failed",
    });

    await expect(pageWith({ payment_id: "pay-456" })).rejects.toThrow(
      "NEXT_REDIRECT"
    );

    expect(mocks.verifyMercadoPagoPayment).toHaveBeenCalledWith({ paymentId: "pay-456" });
    expect(mocks.redirect).toHaveBeenCalledWith(
      "/dashboard/settings?view=subscription&payment_error=true"
    );
  });

  it("renders the dashboard home when no checkout params are present", async () => {
    const element = await pageWith({});
    expect(element).not.toBeNull();
    expect(mocks.redirect).not.toHaveBeenCalled();
  });
});
