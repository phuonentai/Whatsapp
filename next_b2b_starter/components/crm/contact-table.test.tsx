import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { toast } from "sonner";
import { renderWithProviders } from "@/test/render";

const mocks = vi.hoisted(() => ({
  contacts: vi.fn(),
  remove: vi.fn(),
  deleteContact: vi.fn(),
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
  crmRepository: {
    exportContacts: vi.fn(),
    deleteContact: mocks.deleteContact,
  },
}));

import { ContactTable } from "./contact-table";

const CONTACT = {
  id: 1,
  phone_number: "+573123456789",
  display_name: "Juan Pérez",
  email: "juan@example.com",
  lead_status: "nuevo",
  created_at: "2026-01-01T00:00:00Z",
};

const CONTACT_2 = {
  ...CONTACT,
  id: 2,
  display_name: "Ana Torres",
  phone_number: "+573999999999",
  email: "ana@example.com",
  created_at: "2026-01-02T00:00:00Z",
};

const manyContacts = Array.from({ length: 120 }, (_, i) => ({
  ...CONTACT,
  id: i + 1,
  display_name: `Contacto ${i + 1}`,
  phone_number: `+5731000000${String(i).padStart(3, "0")}`,
}));

describe("ContactTable", () => {
  beforeEach(() => {
    mocks.remove.mockReset();
    mocks.deleteContact.mockReset();
    mocks.deleteContact.mockResolvedValue(undefined);
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
    expect(screen.getByText("No hay contactos")).toBeInTheDocument();
  });

  it("filters rows by search input", async () => {
    const user = userEvent.setup();
    mocks.contacts.mockReturnValue([CONTACT, CONTACT_2]);
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

  it("sorts by column on header click and exposes aria-sort", async () => {
    const user = userEvent.setup();
    mocks.contacts.mockReturnValue([CONTACT, CONTACT_2]);
    renderWithProviders(<ContactTable />);

    // Default sort is newest-first (created_at desc): Ana (newer) first.
    const rows = screen.getAllByRole("row");
    expect(within(rows[1]).getByText("Ana Torres")).toBeInTheDocument();

    const nameHeader = screen.getByRole("columnheader", { name: /nombre/i });
    await user.click(nameHeader);
    expect(nameHeader).toHaveAttribute("aria-sort", "ascending");

    const ascRows = screen.getAllByRole("row");
    expect(within(ascRows[1]).getByText("Ana Torres")).toBeInTheDocument();
    expect(within(ascRows[2]).getByText("Juan Pérez")).toBeInTheDocument();

    // Second click toggles to descending.
    await user.click(nameHeader);
    expect(nameHeader).toHaveAttribute("aria-sort", "descending");
    const descRows = screen.getAllByRole("row");
    expect(within(descRows[1]).getByText("Juan Pérez")).toBeInTheDocument();
  });

  it("selects rows, shows the bulk bar, and bulk deletes sequentially", async () => {
    const user = userEvent.setup();
    mocks.contacts.mockReturnValue([CONTACT, CONTACT_2]);
    renderWithProviders(<ContactTable />);

    await user.click(screen.getByRole("checkbox", { name: "Seleccionar Juan Pérez" }));
    await user.click(screen.getByRole("checkbox", { name: "Seleccionar Ana Torres" }));

    expect(screen.getByText("2 contactos seleccionados")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Eliminar seleccionados" }));
    await user.click(screen.getByRole("button", { name: "Eliminar" }));

    expect(mocks.deleteContact).toHaveBeenCalledTimes(2);
    expect(mocks.deleteContact).toHaveBeenCalledWith(1);
    expect(mocks.deleteContact).toHaveBeenCalledWith(2);
    expect(toast.success).toHaveBeenCalledWith("2 eliminados");
  });

  it("select-all selects the whole visible page and clears on second click", async () => {
    const user = userEvent.setup();
    mocks.contacts.mockReturnValue([CONTACT, CONTACT_2]);
    renderWithProviders(<ContactTable />);

    await user.click(screen.getByRole("checkbox", { name: "Seleccionar todos" }));
    expect(screen.getByText("2 contactos seleccionados")).toBeInTheDocument();

    await user.click(screen.getByRole("checkbox", { name: "Seleccionar todos" }));
    expect(screen.queryByText("2 contactos seleccionados")).not.toBeInTheDocument();
  });

  it("virtualizes large tables while preserving scroll height and selection", async () => {
    const user = userEvent.setup();
    // jsdom has no layout, so the virtualizer measures a 0-height scroll
    // element and renders nothing. Feed it a real viewport size via a fake
    // ResizeObserver that reports dimensions on observe().
    class FakeResizeObserver {
      private cb: (entries: { borderBoxSize: { inlineSize: number; blockSize: number }[] }[]) => void;
      constructor(cb: (entries: { borderBoxSize: { inlineSize: number; blockSize: number }[] }[]) => void) {
        this.cb = cb;
      }
      observe() {
        this.cb([{ borderBoxSize: [{ inlineSize: 800, blockSize: 600 }] }]);
      }
      unobserve() {}
      disconnect() {}
    }
    window.ResizeObserver = FakeResizeObserver as unknown as typeof ResizeObserver;

    mocks.contacts.mockReturnValue(manyContacts);
    renderWithProviders(<ContactTable />);

    // Only the visible window is mounted, not all 120 rows.
    const bodyRows = document.querySelectorAll("tbody tr");
    expect(bodyRows.length).toBeGreaterThan(0);
    expect(bodyRows.length).toBeLessThan(120);
    // Scroll height is preserved via the tbody's explicit total height.
    const tbody = document.querySelector("tbody");
    expect(tbody).toHaveStyle({ height: `${120 * 52}px`, position: "relative" });
    // Rows are keyboard-reachable.
    expect(bodyRows[0]).toHaveAttribute("tabindex", "0");

    // Selection still works across the full (virtual) page.
    await user.click(screen.getByRole("checkbox", { name: "Seleccionar todos" }));
    expect(screen.getByText("120 contactos seleccionados")).toBeInTheDocument();
  });

  it("distinguishes no-results from empty data and clears filters", async () => {
    const user = userEvent.setup();
    mocks.contacts.mockReturnValue([CONTACT]);
    renderWithProviders(<ContactTable />);

    await user.type(screen.getByPlaceholderText("Buscar contactos..."), "zzz-no-match");
    expect(screen.getByText("No hay resultados para la búsqueda")).toBeInTheDocument();
    expect(screen.queryByText("No hay contactos")).not.toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Limpiar filtros" }));
    expect(screen.getByText("Juan Pérez")).toBeInTheDocument();
  });
});
