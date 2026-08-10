import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithProviders } from "@/test/render";

const mocks = vi.hoisted(() => ({
  create: vi.fn(),
  update: vi.fn(),
}));

vi.mock("@/lib/hooks/mutations/use-crm-mutations", () => ({
  useCreateCompany: () => ({ mutateAsync: mocks.create, isPending: false }),
  useUpdateCompany: () => ({ mutateAsync: mocks.update, isPending: false }),
}));

import { CompanyDialog } from "./company-dialog";

describe("CompanyDialog", () => {
  beforeEach(() => {
    mocks.create.mockReset();
    mocks.create.mockResolvedValue({});
  });

  it("blocks submit when the name is required", async () => {
    const user = userEvent.setup();
    renderWithProviders(<CompanyDialog open onOpenChange={vi.fn()} />);
    await user.click(screen.getByRole("button", { name: "Guardar" }));
    expect(await screen.findByText("El nombre es requerido")).toBeInTheDocument();
    expect(mocks.create).not.toHaveBeenCalled();
  });

  it("submits valid data and calls the create action once", async () => {
    const user = userEvent.setup();
    const onOpenChange = vi.fn();
    renderWithProviders(<CompanyDialog open onOpenChange={onOpenChange} />);
    await user.type(screen.getByLabelText("Nombre"), "ACME");
    await user.type(screen.getByLabelText("NIT"), "900123456");
    await user.click(screen.getByRole("button", { name: "Guardar" }));
    expect(mocks.create).toHaveBeenCalledTimes(1);
    expect(mocks.create).toHaveBeenCalledWith(
      expect.objectContaining({ name: "ACME", nit: "900123456" })
    );
  });

  it("cancel closes without submitting", async () => {
    const user = userEvent.setup();
    const onOpenChange = vi.fn();
    renderWithProviders(<CompanyDialog open onOpenChange={onOpenChange} />);
    await user.type(screen.getByLabelText("Nombre"), "ACME");
    await user.click(screen.getByRole("button", { name: "Cancelar" }));
    expect(onOpenChange).toHaveBeenCalledWith(false);
    expect(mocks.create).not.toHaveBeenCalled();
  });
});
