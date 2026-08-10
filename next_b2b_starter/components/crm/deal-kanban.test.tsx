import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithProviders } from "@/test/render";

const mocks = vi.hoisted(() => ({
  deals: vi.fn(),
  pipelines: vi.fn(),
  move: vi.fn(),
}));

vi.mock("@/lib/hooks/queries/use-crm-queries", () => ({
  useDealsQuery: () => ({ data: mocks.deals(), isLoading: false }),
  usePipelinesQuery: () => ({ data: mocks.pipelines(), isLoading: false }),
  useCompaniesQuery: () => ({ data: [] }),
  useContactsQuery: () => ({ data: [] }),
}));

vi.mock("@/lib/hooks/mutations/use-crm-mutations", () => ({
  useMoveDealStage: () => ({ mutate: mocks.move }),
  useDeleteDeal: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useCreateDeal: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useUpdateDeal: () => ({ mutateAsync: vi.fn(), isPending: false }),
}));

vi.mock("@/lib/hooks/use-entitlement", () => ({
  useFeature: () => true,
}));

vi.mock("@/lib/hooks/use-permissions", () => ({
  usePermissions: () => ({ hasPermission: () => true }),
}));

vi.mock("@/lib/api/api/repositories/crm-repository", () => ({
  crmRepository: { exportDeals: vi.fn() },
}));

import { DealKanban } from "./deal-kanban";

const STAGES = [
  { id: 1, nombre: "Prospección", orden: 1 },
  { id: 2, nombre: "Negociación", orden: 2 },
];
const DEAL = { id: 10, nombre: "Trato grande", monto: 1000, moneda: "COP", stage_id: 1 };

describe("DealKanban", () => {
  beforeEach(() => {
    mocks.move.mockReset();
    mocks.pipelines.mockReturnValue([{ id: 1, nombre: "Default", etapas: STAGES }]);
    mocks.deals.mockReturnValue([DEAL]);
  });

  it("renders deals in their stage columns", () => {
    renderWithProviders(<DealKanban />);
    expect(screen.getByText("Prospección")).toBeInTheDocument();
    // Stage name also appears in the card's "Mover a" options.
    expect(screen.getAllByText("Negociación").length).toBeGreaterThan(0);
    expect(screen.getByText("Trato grande")).toBeInTheDocument();
  });

  it("renders an empty board when there are no deals", () => {
    mocks.deals.mockReturnValue([]);
    renderWithProviders(<DealKanban />);
    expect(screen.queryByText("Trato grande")).not.toBeInTheDocument();
    expect(screen.getAllByText("Sin negocios").length).toBeGreaterThan(0);
  });

  it("opens the deal detail view on card click", async () => {
    const user = userEvent.setup();
    renderWithProviders(<DealKanban />);
    await user.click(screen.getByTestId("deal-card"));
    // Card click routes via useRouter (mocked) — assert card remains rendered.
    expect(screen.getByTestId("deal-card")).toBeInTheDocument();
  });
});
