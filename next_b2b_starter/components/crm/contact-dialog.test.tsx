import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithProviders } from "@/test/render";

const mocks = vi.hoisted(() => ({
  create: vi.fn(),
  update: vi.fn(),
}));

vi.mock("@/lib/hooks/mutations/use-crm-mutations", () => ({
  useCreateContact: () => ({ mutateAsync: mocks.create, isPending: false }),
  useUpdateContact: () => ({ mutateAsync: mocks.update, isPending: false }),
}));

import { ContactDialog } from "./contact-dialog";

const VALID_PHONE = "+573123456789";

describe("ContactDialog", () => {
  beforeEach(() => {
    mocks.create.mockReset();
    mocks.update.mockReset();
    mocks.create.mockResolvedValue({});
    mocks.update.mockResolvedValue({});
  });

  it("blocks submit with an empty required phone field", async () => {
    const user = userEvent.setup();
    renderWithProviders(<ContactDialog open onOpenChange={vi.fn()} />);
    await user.click(screen.getByRole("button", { name: "Guardar" }));
    expect(await screen.findByText("El teléfono es requerido")).toBeInTheDocument();
    expect(mocks.create).not.toHaveBeenCalled();
  });

  it("rejects an invalid phone format", async () => {
    const user = userEvent.setup();
    renderWithProviders(<ContactDialog open onOpenChange={vi.fn()} />);
    await user.type(screen.getByLabelText("Teléfono"), "123");
    await user.click(screen.getByRole("button", { name: "Guardar" }));
    expect(await screen.findByText("Teléfono inválido")).toBeInTheDocument();
    expect(mocks.create).not.toHaveBeenCalled();
  });

  it("submits valid data and calls the create action once", async () => {
    const user = userEvent.setup();
    const onOpenChange = vi.fn();
    renderWithProviders(<ContactDialog open onOpenChange={onOpenChange} />);
    await user.type(screen.getByLabelText("Teléfono"), VALID_PHONE);
    await user.type(screen.getByLabelText("Nombre"), "Juan");
    await user.click(screen.getByRole("button", { name: "Guardar" }));
    expect(mocks.create).toHaveBeenCalledTimes(1);
    expect(mocks.create).toHaveBeenCalledWith({
      phone_number: VALID_PHONE,
      display_name: "Juan",
      email: "",
      lead_status: "nuevo",
    });
  });

  it("cancel closes the dialog without submitting", async () => {
    const user = userEvent.setup();
    const onOpenChange = vi.fn();
    renderWithProviders(<ContactDialog open onOpenChange={onOpenChange} />);
    await user.type(screen.getByLabelText("Teléfono"), VALID_PHONE);
    await user.click(screen.getByRole("button", { name: "Cancelar" }));
    expect(onOpenChange).toHaveBeenCalledWith(false);
    expect(mocks.create).not.toHaveBeenCalled();
  });
});
