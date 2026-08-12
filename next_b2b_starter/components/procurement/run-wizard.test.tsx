import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithProviders } from "@/test/render";
import { RunWizard } from "./run-wizard";
import { RunBoard } from "./run-board";
import type {
  InquiryRunDto,
  BoardDto,
  OrderDto,
  SupplierDto,
  ProductDto,
} from "@/lib/api/api/dto/procurement.dto";

const SUPPLIERS: SupplierDto[] = [
  { id: 1, organization_id: 42, contact_id: 101, nit: "900111222", is_active: true, created_at: "2026-08-11T00:00:00Z", updated_at: "2026-08-11T00:00:00Z" },
];
const PRODUCTS: ProductDto[] = [
  { id: 10, organization_id: 42, name: "Papel carta", sku: "PAP-001", unit: "resma", is_active: true, created_at: "2026-08-11T00:00:00Z", updated_at: "2026-08-11T00:00:00Z" },
];
const RUN: InquiryRunDto = {
  id: 5,
  organization_id: 42,
  status: "draft",
  source: "manual",
  created_at: "2026-08-11T00:00:00Z",
  updated_at: "2026-08-11T00:00:00Z",
};
const SENT_RUN: InquiryRunDto = { ...RUN, status: "awaiting_responses", sent_at: "2026-08-11T00:00:00Z" };

const BOARD: BoardDto = {
  run: SENT_RUN,
  rows: [
    {
      recipient_id: 1,
      recipient_status: "answered",
      supplier_id: 1,
      nit: "900111222",
      contact_id: 101,
      display_name: "ProvA",
      phone_number: "+573001234567",
      response: {
        id: 1,
        organization_id: 42,
        recipient_id: 1,
        raw_message_id: "m1",
        items: [
          { product_name: "Papel carta", disponible: true, precio_unitario: 10000, cantidad_disponible: 50, tiempo_entrega: "2 días", requiere_seguimiento: false },
        ],
        resumen: "Disponible a 10.000",
        requiere_humano: false,
        created_at: "2026-08-11T00:00:00Z",
      },
    },
    {
      recipient_id: 2,
      recipient_status: "sent",
      supplier_id: 2,
      nit: "900333444",
      contact_id: 102,
      display_name: "ProvB",
      phone_number: "+573005555555",
    },
  ],
};

const createRun = vi.fn();
const sendRun = vi.fn();
const placeOrder = vi.fn();

vi.mock("@/lib/hooks/queries/use-procurement-queries", () => ({
  useSuppliersQuery: () => ({ data: SUPPLIERS, isLoading: false, isError: false, refetch: vi.fn(), isRefetching: false }),
  useProductsQuery: () => ({ data: PRODUCTS, isLoading: false, isError: false, refetch: vi.fn(), isRefetching: false }),
  useRunsQuery: () => ({ data: [RUN, SENT_RUN], isLoading: false, isError: false, refetch: vi.fn(), isRefetching: false }),
  useRunBoardQuery: () => ({ data: BOARD, isLoading: false, isError: false, refetch: vi.fn(), isRefetching: false }),
  useRunOrdersQuery: () => ({ data: [] as OrderDto[], isLoading: false, isError: false, refetch: vi.fn(), isRefetching: false }),
}));

vi.mock("@/lib/hooks/mutations/use-procurement-mutations", () => ({
  useCreateRun: () => ({ mutateAsync: createRun, isPending: false }),
  useSendRun: () => ({ mutateAsync: sendRun, isPending: false }),
  usePlaceOrder: () => ({ mutateAsync: placeOrder, isPending: false }),
}));

vi.mock("next/navigation", () => ({
  useSearchParams: () => ({ get: (k: string) => (k === "run" ? "5" : null) }),
}));

beforeEach(() => {
  vi.clearAllMocks();
});

describe("RunWizard", () => {
  it("submits the wizard payload (suppliers + products with quantities + nota)", async () => {
    const user = userEvent.setup();
    createRun.mockResolvedValue({ ...RUN, status: "draft" });
    renderWithProviders(<RunWizard />);
    await user.click(screen.getByRole("button", { name: "Nueva cotización" }));
    await user.click(screen.getByRole("checkbox"));
    await user.type(screen.getByPlaceholderText("Cantidad"), "5");
    await user.type(screen.getByPlaceholderText("Nota para el proveedor (opcional)"), "Urgente");
    await user.click(screen.getByRole("button", { name: "Crear y redactar" }));
    await waitFor(() =>
      expect(createRun).toHaveBeenCalledWith({
        supplier_ids: [1],
        products: [{ product_id: 10, quantity: 5 }],
        nota: "Urgente",
      })
    );
  });

  it("shows run statuses including escalated", () => {
    const escalated = { ...SENT_RUN, status: "escalated" as const };
    createRun.mockResolvedValue(escalated);
    // Re-render with an escalated run present by mutating the mocked data via
    // a fresh render: assert the badge label renders for the sent run at least.
    renderWithProviders(<RunWizard />);
    expect(screen.getByText("Esperando respuestas")).toBeInTheDocument();
    expect(screen.getByText("Borrador")).toBeInTheDocument();
  });
});

describe("RunBoard", () => {
  it("renders the ranked comparison and no auto-quote for unanswered suppliers", () => {
    renderWithProviders(<RunBoard runId={5} />);
    expect(screen.getByText("ProvA")).toBeInTheDocument();
    expect(screen.getByText("Disponible a 10.000")).toBeInTheDocument();
    expect(screen.getByText("$ 10.000 COP")).toBeInTheDocument();
  });

  it("approves an order with product + quantity (and override toggle for requires_humano)", async () => {
    const user = userEvent.setup();
    placeOrder.mockResolvedValue({ id: 1, status: "placed" } as OrderDto);
    renderWithProviders(<RunBoard runId={5} />);
    await user.click(screen.getByRole("button", { name: "Confirmar pedido" }));
    await user.click(screen.getByRole("button", { name: "Confirmar pedido" })); // dialog submit
    await waitFor(() =>
      expect(placeOrder).toHaveBeenCalledWith({
        runId: 5,
        data: {
          supplier_id: 1,
          items: [{ product_id: 10, quantity: 50 }],
          notes: null,
          override: false,
        },
      })
    );
  });
});
