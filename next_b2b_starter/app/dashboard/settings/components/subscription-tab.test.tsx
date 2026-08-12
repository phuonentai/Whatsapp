import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "@/test/render";
import { ui } from "@/lib/copy/ui";

const mocks = vi.hoisted(() => ({
  searchParams: vi.fn(),
  pathname: vi.fn(),
  replace: vi.fn(),
  products: vi.fn(),
  aiUsage: vi.fn(),
}));

vi.mock("next/navigation", () => ({
  useRouter: () => ({
    replace: mocks.replace,
    push: vi.fn(),
    back: vi.fn(),
    refresh: vi.fn(),
  }),
  usePathname: () => mocks.pathname(),
  useSearchParams: () => mocks.searchParams(),
}));

vi.mock("@/lib/hooks/queries/use-products-query", () => ({
  useProductsQuery: () => ({ data: mocks.products(), isLoading: false, error: null }),
}));

vi.mock("@/lib/hooks/queries/use-ai-usage-query", () => ({
  useAiUsageQuery: () => ({ data: mocks.aiUsage(), isLoading: false, error: null }),
}));

vi.mock("@/lib/actions/billing/cancel-subscription", () => ({
  cancelSubscription: vi.fn(),
}));

vi.mock("@/lib/actions/billing/cancel-mp-subscription", () => ({
  cancelMPSubscription: vi.fn(),
}));

vi.mock("@/lib/actions/billing/create-checkout", () => ({
  createCheckout: vi.fn(),
}));

vi.mock("@/lib/actions/billing/create-mp-checkout", () => ({
  createMercadoPagoCheckout: vi.fn(),
}));

import { SubscriptionTab } from "./subscription-tab";
import type { SubscriptionGateState } from "@/lib/polar/current-subscription";
import type { PolarPlan } from "@/lib/polar/plans";

const ACTIVE_STATE: SubscriptionGateState = {
  isAuthenticated: true,
  isActive: true,
  status: "active",
  reason: undefined,
  productId: "prod-1",
  meterId: "meter-1",
  planId: "plan-pro",
  subscription: {
    id: "sub-1",
    status: "active",
    currentPeriodStart: "2026-08-01T00:00:00Z",
    currentPeriodEnd: "2026-09-01T00:00:00Z",
    cancelAtPeriodEnd: false,
    customerId: "cust-1",
    productId: "prod-1",
    productName: "Pro",
    productMetadata: null,
    trialEnd: null,
    trialStart: null,
    recurringInterval: "month",
    metadata: null,
    customFieldData: null,
    customerCancellationReason: null,
    customerCancellationComment: null,
  },
  usage: {
    meterId: "meter-1",
    customerId: "cust-1",
    included: 10,
    used: 9,
    remaining: 1,
    periodStart: "2026-08-01T00:00:00Z",
    periodEnd: "2026-09-01T00:00:00Z",
  },
  backendAvailable: true,
};

const PRO_PLAN: PolarPlan = {
  id: "plan-pro",
  name: "Pro",
  description: null,
  price: 49,
  interval: "month",
  productId: "prod-1",
  priceId: "price-1",
  includedSeats: 5,
  includedInvoices: 10,
  benefits: ["AI credits"],
};

function renderTab(params: Record<string, string> = {}) {
  mocks.searchParams.mockReturnValue(new URLSearchParams(params));
  mocks.pathname.mockReturnValue("/dashboard/settings");
  mocks.replace.mockClear();
  return renderWithProviders(
    <SubscriptionTab state={null} isLoading={false} error={null} onRefresh={vi.fn()} />
  );
}

function renderActiveTab(params: Record<string, string> = {}) {
  mocks.searchParams.mockReturnValue(new URLSearchParams(params));
  mocks.pathname.mockReturnValue("/dashboard/settings");
  mocks.replace.mockClear();
  return renderWithProviders(
    <SubscriptionTab state={ACTIVE_STATE} isLoading={false} error={null} onRefresh={vi.fn()} />
  );
}

describe("SubscriptionTab checkout-outcome banners", () => {
  beforeEach(() => {
    mocks.replace.mockClear();
    mocks.products.mockReturnValue([]);
    mocks.aiUsage.mockReturnValue(undefined);
  });

  it("renders the success banner when payment_verified is present", () => {
    renderTab({ view: "subscription", payment_verified: "true" });

    expect(screen.getByText(ui.billing.paymentVerifiedTitle)).toBeInTheDocument();
    expect(screen.getByText(ui.billing.paymentVerifiedBody)).toBeInTheDocument();
    expect(screen.queryByText(ui.billing.paymentErrorTitle)).not.toBeInTheDocument();
  });

  it("renders the error banner when payment_error is present", () => {
    renderTab({ view: "subscription", payment_error: "true" });

    expect(screen.getByText(ui.billing.paymentErrorTitle)).toBeInTheDocument();
    expect(screen.getByText(ui.billing.paymentErrorBody)).toBeInTheDocument();
    expect(screen.queryByText(ui.billing.paymentVerifiedTitle)).not.toBeInTheDocument();
  });

  it("renders no banners without checkout params", () => {
    renderTab({ view: "subscription" });

    expect(screen.queryByText(ui.billing.paymentVerifiedTitle)).not.toBeInTheDocument();
    expect(screen.queryByText(ui.billing.paymentErrorTitle)).not.toBeInTheDocument();
  });

  it("clears checkout params (preserving view) when the success banner is dismissed", async () => {
    renderTab({ view: "subscription", payment_verified: "true" });

    await userEvent.click(screen.getByRole("button", { name: ui.billing.understood }));

    expect(mocks.replace).toHaveBeenCalledTimes(1);
    expect(mocks.replace).toHaveBeenCalledWith("/dashboard/settings?view=subscription");
  });

  it("clears checkout params when the error banner is dismissed", async () => {
    renderTab({ view: "subscription", payment_error: "true" });

    await userEvent.click(screen.getByRole("button", { name: ui.billing.understood }));

    expect(mocks.replace).toHaveBeenCalledTimes(1);
    expect(mocks.replace).toHaveBeenCalledWith("/dashboard/settings?view=subscription");
  });

  it("opens the plans modal from the error banner retry action", async () => {
    renderTab({ view: "subscription", payment_error: "true" });

    await userEvent.click(screen.getByRole("button", { name: ui.billing.retryCheckout }));

    expect(screen.getByText(ui.billing.plansTitle)).toBeInTheDocument();
    expect(screen.getByText(ui.billing.noPlans)).toBeInTheDocument();
  });

  it("clears checkout params from the URL after the banner renders, so refresh does not re-show it", () => {
    window.history.replaceState(
      null,
      "",
      "/dashboard/settings?view=subscription&payment_verified=true"
    );

    renderTab({ view: "subscription", payment_verified: "true" });

    // Banner still visible for this render...
    expect(screen.getByText(ui.billing.paymentVerifiedTitle)).toBeInTheDocument();
    // ...but the URL no longer carries the param, so a refresh won't re-show it.
    expect(window.location.search).not.toContain("payment_verified");
    expect(window.location.search).toContain("view=subscription");
  });
});

describe("SubscriptionTab usage bars (amber threshold >=80%)", () => {
  beforeEach(() => {
    mocks.replace.mockClear();
    mocks.products.mockReturnValue([PRO_PLAN]);
  });

  it("shows the amber threshold chip and bar when invoice usage is >= 80%", () => {
    mocks.aiUsage.mockReturnValue(undefined);
    renderActiveTab();

    // 9/10 invoices used = 90% >= 80% → amber chip with text.
    expect(screen.getByText("Uso alto: 90% del límite")).toBeInTheDocument();
    expect(screen.getByText(ui.settings.usageNearLimitBody)).toBeInTheDocument();
  });

  it("shows the amber threshold chip when AI credits usage is >= 80%", () => {
    mocks.aiUsage.mockReturnValue({
      credits_used: 85,
      credits_max: 100,
      credits_remaining: 15,
      tokens_input: 1000,
      tokens_output: 500,
      period_end: "2026-09-01T00:00:00Z",
    });
    renderActiveTab();

    // Invoice usage is 9/10 (90%) and AI credits 85/100 (85%): both at limit.
    expect(screen.getAllByText(ui.settings.usageNearLimitBody).length).toBeGreaterThan(0);
    expect(screen.getByText("Uso alto: 85% del límite")).toBeInTheDocument();
  });

  it("keeps a neutral gray chip when usage is below the threshold", () => {
    mocks.aiUsage.mockReturnValue(undefined);
    const below: SubscriptionGateState = {
      ...ACTIVE_STATE,
      usage: {
        ...ACTIVE_STATE.usage!,
        included: 10,
        used: 4,
        remaining: 6,
      },
    };
    mocks.searchParams.mockReturnValue(new URLSearchParams());
    mocks.pathname.mockReturnValue("/dashboard/settings");
    renderWithProviders(
      <SubscriptionTab state={below} isLoading={false} error={null} onRefresh={vi.fn()} />
    );

    // 4/10 = 40% < 80% → neutral chip, no amber copy.
    expect(screen.getByText("Uso: 40% del límite")).toBeInTheDocument();
    expect(screen.queryByText(/Uso alto:/)).not.toBeInTheDocument();
    expect(screen.queryByText(ui.settings.usageNearLimitBody)).not.toBeInTheDocument();
  });
});
