import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithProviders } from "@/test/render";

const mocks = vi.hoisted(() => ({
  contacts: vi.fn(),
  remove: vi.fn(),
}));

vi.mock("@/lib/hooks/queries/use-crm-queries", () => ({
  useContactsQuery: () => ({ data: mocks.contacts(), isLoading: false }),
}));

vi.mock("@/lib/hooks/mutations/use-crm-mutations", () => ({
  useDeleteContact: () => ({ mutateAsync: mocks.remove, isPending: false }),
  useCreateContact: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useUpdateContact: () => ({ mutateAsync: vi.fn(), isPending: false }),
}));

vi.mock("@/lib/hooks/use-entitlement", () => ({
  useFeature: () => true,
}));

vi.mock("@/lib/hooks/use-permissions", () => ({
  usePermissions: () => ({ hasPermission: () => true }),
}));

vi.mock("@/lib/api/api/repositories/crm-repository", () => ({
  crmRepository: { exportContacts: vi.fn() },
}));

import { ContactTable } from "./contact-table";

const CONTACT = {
  id: 1,
  phone_number: "+573123456789",
  display_name: "Juan Pérez",
  email: "juan@example.com",
  lead_status: "nuevo",
};

describe("ContactTable", () => {
  beforeEach(() => {
    mocks.remove.mockReset();
    mocks.contacts.mockReturnValue([CONTACT]);
  });

  it("renders rows from data", () => {
    renderWithProviders(<ContactTable />);
    expect(screen.getByText("Juan Pérez")).toBeInTheDocument();
    expect(screen.getByText("+573123456789")).toBeInTheDocument();
    expect(screen.getByText("juan@example.com")).toBeInTheDocument();
  });

  it("renders an empty table when there is no data", () => {
    mocks.contacts.mockReturnValue([]);
    renderWithProviders(<ContactTable />);
    expect(screen.getByRole("columnheader", { name: "Nombre" })).toBeInTheDocument();
    expect(screen.queryByText("+573123456789")).not.toBeInTheDocument();
  });

  it("filters rows by search input", async () => {
    const user = userEvent.setup();
    mocks.contacts.mockReturnValue([
      CONTACT,
      { ...CONTACT, id: 2, display_name: "Ana Torres", phone_number: "+573999999999" },
    ]);
    renderWithProviders(<ContactTable />);
    await user.type(screen.getByPlaceholderText("Buscar contactos..."), "Ana");
    expect(screen.getByText("Ana Torres")).toBeInTheDocument();
    expect(screen.queryByText("Juan Pérez")).not.toBeInTheDocument();
  });

  it("deletes a contact through the confirm dialog", async () => {
    const user = userEvent.setup();
    renderWithProviders(<ContactTable />);
    await user.click(screen.getByRole("button", { name: /eliminar/i }));
    await user.click(screen.getByRole("button", { name: "Eliminar" }));
    expect(mocks.remove).toHaveBeenCalledWith(1);
  });
});
