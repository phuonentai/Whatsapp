import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithProviders } from "@/test/render";

const mocks = vi.hoisted(() => ({
  companies: vi.fn(),
  remove: vi.fn(),
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
  crmRepository: { exportCompanies: vi.fn() },
}));

import { CompanyTable } from "./company-table";

const COMPANY = { id: 1, name: "ACME SAS", nit: "900123456", sector: "Tecnología" };

describe("CompanyTable", () => {
  beforeEach(() => {
    mocks.remove.mockReset();
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
  });

  it("deletes a company through the confirm dialog", async () => {
    const user = userEvent.setup();
    renderWithProviders(<CompanyTable />);
    await user.click(screen.getByRole("button", { name: /eliminar/i }));
    await user.click(screen.getByRole("button", { name: "Eliminar" }));
    expect(mocks.remove).toHaveBeenCalledWith(1);
  });
});
