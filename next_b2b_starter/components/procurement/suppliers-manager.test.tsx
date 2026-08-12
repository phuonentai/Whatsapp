import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithProviders } from "@/test/render";
import { SuppliersManager } from "./suppliers-manager";
import { ProductsManager } from "./products-manager";
import type { SupplierDto, ProductDto } from "@/lib/api/api/dto/procurement.dto";

const SUPPLIER: SupplierDto = {
  id: 1,
  organization_id: 42,
  contact_id: 101,
  nit: "900111222",
  display_name: "Distribuidora Andina",
  phone_number: "+573001112223",
  is_active: true,
  created_at: "2026-08-11T00:00:00Z",
  updated_at: "2026-08-11T00:00:00Z",
};

const PRODUCT: ProductDto = {
  id: 10,
  organization_id: 42,
  name: "Papel carta",
  sku: "PAP-001",
  unit: "resma",
  is_active: true,
  created_at: "2026-08-11T00:00:00Z",
  updated_at: "2026-08-11T00:00:00Z",
};

const createSupplier = vi.fn();
const updateSupplier = vi.fn();
const createProduct = vi.fn();
const updateProduct = vi.fn();

vi.mock("@/lib/hooks/queries/use-procurement-queries", () => ({
  useSuppliersQuery: () => ({ data: [SUPPLIER], isLoading: false, isError: false, refetch: vi.fn(), isRefetching: false }),
  useProductsQuery: () => ({ data: [PRODUCT], isLoading: false, isError: false, refetch: vi.fn(), isRefetching: false }),
}));

vi.mock("@/lib/hooks/mutations/use-procurement-mutations", () => ({
  useCreateSupplier: () => ({ mutateAsync: createSupplier, isPending: false }),
  useUpdateSupplier: () => ({ mutateAsync: updateSupplier, isPending: false }),
  useCreateProduct: () => ({ mutateAsync: createProduct, isPending: false }),
  useUpdateProduct: () => ({ mutateAsync: updateProduct, isPending: false }),
}));

beforeEach(() => {
  vi.clearAllMocks();
});

describe("SuppliersManager", () => {
  it("renders the org-scoped supplier list", () => {
    renderWithProviders(<SuppliersManager />);
    expect(screen.getByText("Distribuidora Andina")).toBeInTheDocument();
    expect(screen.getByText("900111222")).toBeInTheDocument();
    expect(screen.getByText("Activo")).toBeInTheDocument();
  });

  it("submits the create-supplier payload (NIT + phone + optional fields)", async () => {
    const user = userEvent.setup();
    createSupplier.mockResolvedValue(SUPPLIER);
    renderWithProviders(<SuppliersManager />);
    await user.click(screen.getByRole("button", { name: "Nuevo proveedor" }));
    await user.type(screen.getByLabelText("Nombre comercial"), "Distribuidora Andina");
    await user.type(screen.getByLabelText("NIT"), "900999888");
    await user.type(screen.getByLabelText("Teléfono"), "+573009999999");
    await user.type(screen.getByLabelText("Días de entrega"), "3");
    await user.click(screen.getByRole("button", { name: "Guardar" }));
    await waitFor(() =>
      expect(createSupplier).toHaveBeenCalledWith({
        nit: "900999888",
        phone: "+573009999999",
        display_name: "Distribuidora Andina",
        delivery_days: 3,
        min_order_amount: null,
        notes: null,
      })
    );
  });

  it("deactivates a supplier via the toggle", async () => {
    const user = userEvent.setup();
    updateSupplier.mockResolvedValue(SUPPLIER);
    renderWithProviders(<SuppliersManager />);
    await user.click(screen.getByRole("button", { name: "Desactivar" }));
    await waitFor(() =>
      expect(updateSupplier).toHaveBeenCalledWith({
        id: 1,
        data: expect.objectContaining({ is_active: false }),
      })
    );
  });
});

describe("ProductsManager", () => {
  it("renders products and deactivates", async () => {
    const user = userEvent.setup();
    updateProduct.mockResolvedValue(PRODUCT);
    renderWithProviders(<ProductsManager />);
    expect(screen.getByText("Papel carta")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Desactivar" }));
    await waitFor(() =>
      expect(updateProduct).toHaveBeenCalledWith({
        id: 10,
        data: expect.objectContaining({ is_active: false }),
      })
    );
  });

  it("submits the create-product payload", async () => {
    const user = userEvent.setup();
    createProduct.mockResolvedValue(PRODUCT);
    renderWithProviders(<ProductsManager />);
    await user.click(screen.getByRole("button", { name: "Nuevo producto" }));
    await user.type(screen.getByLabelText("Nombre"), "Esfero negro");
    await user.type(screen.getByLabelText("SKU"), "ESF-002");
    await user.click(screen.getByRole("button", { name: "Guardar" }));
    await waitFor(() =>
      expect(createProduct).toHaveBeenCalledWith({ name: "Esfero negro", sku: "ESF-002", unit: "und" })
    );
  });
});
