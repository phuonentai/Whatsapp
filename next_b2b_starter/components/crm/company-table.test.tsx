import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { toast } from "sonner";
import { renderWithProviders } from "@/test/render";

const mocks = vi.hoisted(() => ({
  companies: vi.fn(),
  remove: vi.fn(),
  deleteCompany: vi.fn(),
}));

vi.mock("@/lib/hooks/queries/use-crm-queries", () => ({
  useCompaniesQuery: () => ({ data: mocks.companies(), isLoading: false }),
}));

vi.mock("@/lib/hooks/mutations/use-crm-mutations", () => ({
  useDeleteCompany: () => ({ mutateAsync: mocks.remove, isPending: false }),
  useCreateCompany: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useUpdateCompany: () => ({ mutateAsync: vi.fn(), isPending: false }),
}));

vi.mock("@/lib/hooks/use-entitlement", () => ({
  useFeature: () => true,
}));

vi.mock("@/lib/hooks/use-permissions", () => ({
  usePermissions: () => ({ hasPermission: () => true }),
}));

vi.mock("@/lib/api/api/repositories/crm-repository", () => ({
  crmRepository: {
    exportCompanies: vi.fn(),
    deleteCompany: mocks.deleteCompany,
  },
}));

import { CompanyTable } from "./company-table";

const COMPANY = { id: 1, name: "ACME SAS", nit: "900123456", sector: "Tecnología", created_at: "2026-01-01T00:00:00Z" };
const COMPANY_2 = { ...COMPANY, id: 2, name: "Beta Ltda", nit: "900654321", sector: "Salud", created_at: "2026-01-02T00:00:00Z" };

describe("CompanyTable", () => {
  beforeEach(() => {
    mocks.remove.mockReset();
    mocks.deleteCompany.mockReset();
    mocks.deleteCompany.mockResolvedValue(undefined);
    mocks.companies.mockReturnValue([COMPANY]);
  });

  it("renders rows from data", () => {
    renderWithProviders(<CompanyTable />);
    expect(screen.getByText("ACME SAS")).toBeInTheDocument();
    expect(screen.getByText("900123456")).toBeInTheDocument();
  });

  it("renders an empty table when there is no data", () => {
    mocks.companies.mockReturnValue([]);
    renderWithProviders(<CompanyTable />);
    expect(screen.queryByText("ACME SAS")).not.toBeInTheDocument();
    expect(screen.getByText("No hay empresas")).toBeInTheDocument();
  });

  it("deletes a company through the confirm dialog", async () => {
    const user = userEvent.setup();
    renderWithProviders(<CompanyTable />);
    await user.click(screen.getByRole("button", { name: /eliminar/i }));
    await user.click(screen.getByRole("button", { name: "Eliminar" }));
    expect(mocks.remove).toHaveBeenCalledWith(1);
  });

  it("sorts by name on header click and exposes aria-sort", async () => {
    const user = userEvent.setup();
    mocks.companies.mockReturnValue([COMPANY, COMPANY_2]);
    renderWithProviders(<CompanyTable />);

    // Default newest-first: Beta (newer) first.
    const rows = screen.getAllByRole("row");
    expect(within(rows[1]).getByText("Beta Ltda")).toBeInTheDocument();

    const nameHeader = screen.getByRole("columnheader", { name: /nombre/i });
    await user.click(nameHeader);
    expect(nameHeader).toHaveAttribute("aria-sort", "ascending");
    const ascRows = screen.getAllByRole("row");
    expect(within(ascRows[1]).getByText("ACME SAS")).toBeInTheDocument();
    expect(within(ascRows[2]).getByText("Beta Ltda")).toBeInTheDocument();

    await user.click(nameHeader);
    expect(nameHeader).toHaveAttribute("aria-sort", "descending");
    const descRows = screen.getAllByRole("row");
    expect(within(descRows[1]).getByText("Beta Ltda")).toBeInTheDocument();
  });

  it("bulk deletes selected companies sequentially with aggregate toast", async () => {
    const user = userEvent.setup();
    mocks.companies.mockReturnValue([COMPANY, COMPANY_2]);
    renderWithProviders(<CompanyTable />);

    await user.click(screen.getByRole("checkbox", { name: "Seleccionar ACME SAS" }));
    await user.click(screen.getByRole("checkbox", { name: "Seleccionar Beta Ltda" }));

    expect(screen.getByText("2 empresas seleccionadas")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Eliminar seleccionadas" }));
    await user.click(screen.getByRole("button", { name: "Eliminar" }));

    expect(mocks.deleteCompany).toHaveBeenCalledTimes(2);
    expect(mocks.deleteCompany).toHaveBeenCalledWith(1);
    expect(mocks.deleteCompany).toHaveBeenCalledWith(2);
    expect(toast.success).toHaveBeenCalledWith("2 eliminadas");
  });

  it("distinguishes no-results from empty data and clears filters", async () => {
    const user = userEvent.setup();
    mocks.companies.mockReturnValue([COMPANY]);
    renderWithProviders(<CompanyTable />);

    await user.type(screen.getByPlaceholderText("Buscar empresas por nombre, NIT o sector..."), "zzz-no-match");
    expect(screen.getByText("No hay resultados para la búsqueda")).toBeInTheDocument();
    expect(screen.queryByText("No hay empresas")).not.toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Limpiar filtros" }));
    expect(screen.getByText("ACME SAS")).toBeInTheDocument();
  });
});
