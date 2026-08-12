import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "@/test/render";
import type { PolarPlan } from "@/lib/polar/plans";
import type { SubscriptionGateState } from "@/lib/polar/current-subscription";
import { ui } from "@/lib/copy/ui";

import { PlansModal } from "./plans-modal";

const mocks = vi.hoisted(() => ({
  products: undefined as PolarPlan[] | undefined,
}));

vi.mock("@/lib/hooks/queries/use-products-query", () => ({
  useProductsQuery: () => ({
    data: mocks.products,
    isLoading: false,
    error: null,
  }),
}));

vi.mock("@/lib/actions/billing/create-checkout", () => ({
  createCheckout: vi.fn(),
}));

vi.mock("@/lib/actions/billing/create-mp-checkout", () => ({
  createMercadoPagoCheckout: vi.fn(),
}));

const monthStarter: PolarPlan = {
  id: "plan-starter-month",
  name: "Starter",
  description: "Para empezar",
  price: 100,
  interval: "month",
  productId: "prod-starter",
  priceId: "price-starter",
  includedSeats: 5,
  includedInvoices: 100,
  benefits: ["Beneficio A"],
  metadata: { ai_credits_max: 250 },
};

const monthGrowth: PolarPlan = {
  id: "plan-growth-month",
  name: "Growth",
  description: "Para crecer",
  price: 200,
  interval: "month",
  productId: "prod-growth",
  priceId: "price-growth",
  includedSeats: null,
  includedInvoices: null,
  benefits: [],
};

const yearStarter: PolarPlan = {
  ...monthStarter,
  id: "plan-starter-year",
  name: "Starter Anual",
  price: 1000,
  interval: "year",
};

const yearGrowth: PolarPlan = {
  ...monthGrowth,
  id: "plan-growth-year",
  name: "Growth Anual",
  price: 2000,
  interval: "year",
};

const activeSubscriptionState: SubscriptionGateState = {
  isAuthenticated: true,
  isActive: true,
  status: "active",
  productId: "prod-starter",
  planId: null,
  meterId: null,
  subscription: {
    id: "sub_1",
    status: "active",
    currentPeriodStart: "2026-01-01T00:00:00Z",
    currentPeriodEnd: "2026-02-01T00:00:00Z",
    cancelAtPeriodEnd: false,
    customerId: "cus_1",
    productId: "prod-starter",
    productName: "Starter",
    productMetadata: null,
    trialEnd: null,
    trialStart: null,
    recurringInterval: "month",
    metadata: null,
    customFieldData: null,
    customerCancellationReason: null,
    customerCancellationComment: null,
  },
  usage: null,
  backendAvailable: true,
};

describe("PlansModal", () => {
  beforeEach(() => {
    mocks.products = undefined;
  });

  it("renders plan cards with seats, invoices and AI credits from metadata", () => {
    mocks.products = [monthStarter, monthGrowth];

    renderWithProviders(<PlansModal open onOpenChange={vi.fn()} />);

    // Plan names appear in both the card and the comparison header
    expect(screen.getAllByText("Starter").length).toBeGreaterThan(0);
    expect(screen.getAllByText("Growth").length).toBeGreaterThan(0);

    // Seats line: "5 puestos incluidos"
    const seatsLine = screen.getByText(ui.billing.seatsIncluded).closest("li");
    expect(within(seatsLine!).getByText("5")).toBeInTheDocument();

    // Invoices line: "100 facturas por mes"
    const invoicesLine = screen.getByText(ui.billing.invoicesPerMonth).closest("li");
    expect(within(invoicesLine!).getByText("100")).toBeInTheDocument();

    // AI-credit line from metadata.ai_credits_max: "250 créditos por período"
    const aiLine = screen.getByText(ui.billing.aiCreditsPerPeriod).closest("li");
    expect(within(aiLine!).getByText("250")).toBeInTheDocument();

    // Growth has no metadata / null seats+invoices: no per-card lines rendered
    expect(screen.getAllByText(ui.billing.aiCreditsPerPeriod)).toHaveLength(1);
  });

  it("renders the comparison with values and a '—' fallback for missing ones", () => {
    mocks.products = [monthStarter, monthGrowth];

    renderWithProviders(<PlansModal open onOpenChange={vi.fn()} />);

    const table = screen.getByRole("table");
    expect(screen.getByText(ui.billing.comparisonTitle)).toBeInTheDocument();
    expect(within(table).getByText(ui.billing.comparisonSeats)).toBeInTheDocument();
    expect(within(table).getByText(ui.billing.comparisonInvoices)).toBeInTheDocument();
    expect(within(table).getByText(ui.billing.comparisonAiCredits)).toBeInTheDocument();

    // Starter row values
    expect(within(table).getAllByText("5")).toHaveLength(1);
    expect(within(table).getAllByText("100")).toHaveLength(1);
    expect(within(table).getAllByText("250")).toHaveLength(1);

    // Growth has no seats, invoices or AI credits -> "—" for each column
    expect(within(table).getAllByText("—")).toHaveLength(3);
  });

  it("shows both interval options when the catalog has both and filters plans on switch", async () => {
    mocks.products = [monthStarter, monthGrowth, yearStarter, yearGrowth];

    renderWithProviders(<PlansModal open onOpenChange={vi.fn()} />);

    const toggle = screen.getByRole("group", { name: ui.billing.intervalToggleAria });
    expect(within(toggle).getByText(ui.billing.intervalMonthly)).toBeInTheDocument();
    expect(within(toggle).getByText(ui.billing.intervalAnnual)).toBeInTheDocument();

    // Month plans shown by default (name appears in card + comparison header)
    expect(screen.getAllByText("Starter").length).toBeGreaterThan(0);
    expect(screen.getAllByText("Growth").length).toBeGreaterThan(0);
    expect(screen.queryByText("Starter Anual")).not.toBeInTheDocument();

    await userEvent.click(within(toggle).getByText(ui.billing.intervalAnnual));

    expect(screen.getAllByText("Starter Anual").length).toBeGreaterThan(0);
    expect(screen.getAllByText("Growth Anual").length).toBeGreaterThan(0);
    expect(screen.queryByText("Starter")).not.toBeInTheDocument();
    expect(screen.queryByText("Growth")).not.toBeInTheDocument();
  });

  it("hides the interval toggle when the catalog has a single interval", () => {
    mocks.products = [monthStarter, monthGrowth];

    renderWithProviders(<PlansModal open onOpenChange={vi.fn()} />);

    expect(
      screen.queryByRole("group", { name: ui.billing.intervalToggleAria })
    ).not.toBeInTheDocument();
  });

  it("marks the current plan when the subscription matches a product", () => {
    mocks.products = [monthStarter, monthGrowth];

    renderWithProviders(
      <PlansModal open onOpenChange={vi.fn()} subscriptionState={activeSubscriptionState} />
    );

    // "Plan actual" badge + disabled current-plan button on the Starter card
    expect(screen.getAllByText(ui.billing.currentPlan)).toHaveLength(2);
    const starterCard = screen
      .getAllByText("Starter")
      .map((el) => el.closest("article"))
      .find((el) => el !== null);
    expect(starterCard).not.toBeNull();
    expect(within(starterCard!).getAllByText(ui.billing.currentPlan)).toHaveLength(2);
  });

  it("coerces string-typed numeric metadata (\"500\") for AI credits", () => {
    mocks.products = [
      {
        ...monthStarter,
        id: "plan-string-credits",
        name: "String Credits",
        metadata: { ai_credits_max: "500" },
      },
      monthGrowth,
    ];

    renderWithProviders(<PlansModal open onOpenChange={vi.fn()} />);

    // Card line: "500 créditos por período"
    const aiLine = screen.getByText(ui.billing.aiCreditsPerPeriod).closest("li");
    expect(within(aiLine!).getByText("500")).toBeInTheDocument();

    // Comparison column shows the coerced value, not "—"
    const table = screen.getByRole("table");
    expect(within(table).getByText("500")).toBeInTheDocument();
  });

  it("renders the MP option when enabled and promotes it when Polar is unconfigured", () => {
    mocks.products = [monthStarter, monthGrowth];

    const unconfiguredState: SubscriptionGateState = {
      ...activeSubscriptionState,
      isActive: false,
      reason: "POLAR_UNCONFIGURED",
      productId: null,
      subscription: null,
    };

    renderWithProviders(
      <PlansModal
        open
        onOpenChange={vi.fn()}
        mercadopagoEnabled
        subscriptionState={unconfiguredState}
      />
    );

    // MP option rendered on every non-current card
    const mpButtons = screen.getAllByText(ui.billing.mpCard);
    expect(mpButtons).toHaveLength(2);
    // MP CTA is primary (dark) while Polar is demoted (outline)
    expect(mpButtons[0].className).toContain("bg-gray-900");
    const polarButtons = screen.getAllByText(ui.billing.internationalCard);
    expect(polarButtons).toHaveLength(2);
    expect(polarButtons[0].className).toContain("bg-white");
  });

  it("does not render the MP option when disabled", () => {
    mocks.products = [monthStarter, monthGrowth];

    renderWithProviders(<PlansModal open onOpenChange={vi.fn()} />);

    expect(screen.queryByText(ui.billing.mpCard)).not.toBeInTheDocument();
  });
});
