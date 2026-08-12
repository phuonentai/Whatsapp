import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithProviders } from "@/test/render";

const mocks = vi.hoisted(() => ({
  catalog: vi.fn(),
  orgModules: vi.fn(),
  moduleState: vi.fn(),
  saveConfig: vi.fn(),
}));

vi.mock("@/lib/hooks/queries/use-modules-queries", () => ({
  useModulesCatalogQuery: () => ({ data: mocks.catalog(), isLoading: false }),
  useOrgModulesQuery: () => ({ data: mocks.orgModules() }),
}));

vi.mock("@/lib/hooks/use-entitlement", () => ({
  useModule: (key: string) => mocks.moduleState(key),
  // Plan source for the "Incluido en plan X" badge (design language).
  useEntitlementQuery: () => ({ data: { plan: "Pro" } }),
}));

vi.mock("@/lib/hooks/mutations/use-tickets-mutations", () => ({
  useSaveModuleConfig: () => ({ mutate: mocks.saveConfig, isPending: false }),
}));

import { ModulesSection } from "./modules-section";

const TICKETS_MODULE = {
  key: "tickets",
  name: "Tickets (Helpdesk)",
  description: "Cola de tickets de soporte.",
  features: ["tickets_module"],
  requires: [],
};

describe("ModulesSection", () => {
  beforeEach(() => {
    mocks.saveConfig.mockReset();
    mocks.catalog.mockReturnValue([TICKETS_MODULE]);
    mocks.orgModules.mockReturnValue([]);
    mocks.moduleState.mockReturnValue({ enabled: false, config: undefined });
  });

  it("lists modules with a disabled badge when not acquired", () => {
    renderWithProviders(<ModulesSection />);
    expect(screen.getByText("Tickets (Helpdesk)")).toBeInTheDocument();
    expect(screen.getByText("No adquirido")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Guardar configuración" })).not.toBeInTheDocument();
  });

  it("shows the config form and persists config for an enabled module", async () => {
    const user = userEvent.setup();
    mocks.moduleState.mockReturnValue({ enabled: true, config: undefined });
    renderWithProviders(<ModulesSection />);
    expect(screen.getByText("Incluido en plan Pro")).toBeInTheDocument();
    expect(screen.getByPlaceholderText("high:8,normal:24,low:48")).toBeInTheDocument();

    await user.type(screen.getByPlaceholderText("high:8,normal:24,low:48"), "high:8,low:48");
    await user.click(screen.getByRole("button", { name: "Guardar configuración" }));
    expect(mocks.saveConfig).toHaveBeenCalledWith({
      key: "tickets",
      config: { sla_hours: { high: 8, low: 48 } },
    });
  });

  it("seeds the config form from persisted org config", async () => {
    const config = { priorities: ["billing", "bug"] };
    mocks.orgModules.mockReturnValue([]);
    mocks.moduleState.mockReturnValue({ enabled: true, config: undefined });

    const { rerender } = renderWithProviders(<ModulesSection />);
    expect(screen.getByPlaceholderText("low,normal,high")).toHaveValue("");

    // Org config arrives after mount → the render-phase seed kicks in.
    mocks.orgModules.mockReturnValue([
      { key: "tickets", name: "Tickets (Helpdesk)", description: "d", features: [], config },
    ]);
    mocks.moduleState.mockReturnValue({ enabled: true, config });
    rerender(<ModulesSection />);

    expect(screen.getByPlaceholderText("low,normal,high")).toHaveValue("billing,bug");
  });
});
