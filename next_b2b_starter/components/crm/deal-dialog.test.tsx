import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithProviders } from "@/test/render";

const mocks = vi.hoisted(() => ({
  create: vi.fn(),
  update: vi.fn(),
  companies: vi.fn(),
  contacts: vi.fn(),
}));

vi.mock("@/lib/hooks/mutations/use-crm-mutations", () => ({
  useCreateDeal: () => ({ mutateAsync: mocks.create, isPending: false }),
  useUpdateDeal: () => ({ mutateAsync: mocks.update, isPending: false }),
}));

vi.mock("@/lib/hooks/queries/use-crm-queries", () => ({
  useCompaniesQuery: () => ({ data: mocks.companies() }),
  useContactsQuery: () => ({ data: mocks.contacts() }),
}));

import { DealDialog } from "./deal-dialog";

const STAGES = [
  { id: 1, nombre: "Prospección", orden: 1, pipeline_id: 1, created_at: "2026-01-01T00:00:00Z", updated_at: "2026-01-01T00:00:00Z" },
  { id: 2, nombre: "Negociación", orden: 2, pipeline_id: 1, created_at: "2026-01-01T00:00:00Z", updated_at: "2026-01-01T00:00:00Z" },
];

describe("DealDialog", () => {
  beforeEach(() => {
    mocks.create.mockReset();
    mocks.update.mockReset();
    mocks.create.mockResolvedValue({});
    mocks.companies.mockReturnValue([{ id: 1, name: "ACME" }]);
    mocks.contacts.mockReturnValue([{ id: 1, phone_number: "+573123456789" }]);
  });

  it("blocks submit when the name is required", async () => {
    const user = userEvent.setup();
    renderWithProviders(
      <DealDialog open onOpenChange={vi.fn()} pipelineId={1} stages={STAGES} />
    );
    await user.click(screen.getByRole("button", { name: "Guardar" }));
    expect(await screen.findByText("El nombre es requerido")).toBeInTheDocument();
    expect(mocks.create).not.toHaveBeenCalled();
  });

  it("submits valid data and calls the create action once", async () => {
    const user = userEvent.setup();
    renderWithProviders(<DealDialog open onOpenChange={vi.fn()} pipelineId={1} stages={STAGES} />);
    await user.type(screen.getByLabelText("Nombre"), "Trato grande");
    await user.click(screen.getByRole("button", { name: "Guardar" }));
    expect(mocks.create).toHaveBeenCalledTimes(1);
    expect(mocks.create).toHaveBeenCalledWith(
      expect.objectContaining({ nombre: "Trato grande" })
    );
  });

  it("cancel closes without submitting", async () => {
    const user = userEvent.setup();
    const onOpenChange = vi.fn();
    renderWithProviders(
      <DealDialog open onOpenChange={onOpenChange} pipelineId={1} stages={STAGES} />
    );
    await user.type(screen.getByLabelText("Nombre"), "Trato");
    await user.click(screen.getByRole("button", { name: "Cancelar" }));
    expect(onOpenChange).toHaveBeenCalledWith(false);
    expect(mocks.create).not.toHaveBeenCalled();
  });
});
