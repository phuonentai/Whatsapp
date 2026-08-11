import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithProviders } from "@/test/render";

const mocks = vi.hoisted(() => ({
  rows: vi.fn(),
  provision: vi.fn(),
}));

vi.mock("@/lib/hooks/queries/use-siigo-queries", () => ({
  useAdminConnectionsQuery: () => ({
    data: mocks.rows(),
    isLoading: false,
    error: null,
    refetch: vi.fn(),
  }),
  useSiigoStatusQuery: () => ({ data: undefined, isLoading: false, error: null, refetch: vi.fn() }),
  useSiigoNumerationQuery: () => ({ data: undefined, isLoading: false, refetch: vi.fn() }),
  useImportPreviewQuery: () => ({ data: undefined, isFetching: false, refetch: vi.fn() }),
}));

vi.mock("@/lib/hooks/mutations/use-siigo-mutations", () => ({
  useAdminProvision: () => ({ mutate: mocks.provision, isPending: false, error: null }),
  useSiigoConnect: () => ({ mutate: vi.fn(), isPending: false, error: null }),
  useRequestAssistedSetup: () => ({ mutate: vi.fn(), isPending: false, error: null }),
  useConfirmNumeration: () => ({ mutate: vi.fn(), isPending: false, error: null }),
  useImportConfirm: () => ({ mutate: vi.fn(), isPending: false, error: null }),
  useTestInvoice: () => ({ mutate: vi.fn(), isPending: false, error: null }),
  useActivateInvoicing: () => ({ mutate: vi.fn(), isPending: false, error: null }),
  usePauseInvoicing: () => ({ mutate: vi.fn(), isPending: false, error: null }),
  useResumeInvoicing: () => ({ mutate: vi.fn(), isPending: false, error: null }),
  useSiigoSync: () => ({ mutate: vi.fn(), isPending: false, error: null }),
}));

import { SiigoAdminView } from "./siigo-admin-view";

describe("SiigoAdminView", () => {
  beforeEach(() => {
    mocks.rows.mockReset();
    mocks.provision.mockReset();
  });

  it("shows connection rows with status, numeration, and import info", () => {
    mocks.rows.mockReturnValue([
      {
        organization_id: 7,
        provider: "siigo",
        status: "live",
        nit: "9001234567",
        numeration: { mode: "auto", prefijo: "FAC1", next_number: "124" },
        last_import_run: { kind: "delta", counts: { nuevos: 3 }, pulled_at: "2026-08-10T00:00:00Z" },
      },
      {
        organization_id: 9,
        provider: "siigo",
        status: "connected",
        nit: "9001112223",
      },
    ]);
    renderWithProviders(<SiigoAdminView />);
    expect(screen.getByText("7")).toBeInTheDocument();
    expect(screen.getByText("FAC1 124")).toBeInTheDocument();
    expect(screen.getByText(/delta · 3 nuevos/)).toBeInTheDocument();
    expect(screen.getByText("9")).toBeInTheDocument();
  });

  it("shows empty state without connections", () => {
    mocks.rows.mockReturnValue([]);
    renderWithProviders(<SiigoAdminView />);
    expect(screen.getByText(/Ninguna organización ha conectado Siigo/)).toBeInTheDocument();
  });

  it("renders provision form only for awaiting_setup and submits credentials", async () => {
    const user = userEvent.setup();
    mocks.rows.mockReturnValue([
      { organization_id: 12, provider: "none", status: "awaiting_setup" },
      { organization_id: 13, provider: "siigo", status: "live" },
    ]);
    renderWithProviders(<SiigoAdminView />);
    expect(screen.getByText("Configuración asistida")).toBeInTheDocument();
    expect(screen.getByPlaceholderText("client_id")).toBeInTheDocument();

    await user.type(screen.getByPlaceholderText("client_id"), "cid");
    await user.type(screen.getByPlaceholderText("client_secret"), "csec");
    await user.type(screen.getByPlaceholderText("NIT"), "9001234567");
    await user.click(screen.getByRole("button", { name: "Provisionar" }));
    expect(mocks.provision).toHaveBeenCalledWith({
      organization_id: 12,
      client_id: "cid",
      client_secret: "csec",
      nit: "9001234567",
    });
  });
});
