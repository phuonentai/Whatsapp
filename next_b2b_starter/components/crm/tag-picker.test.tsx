import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithProviders } from "@/test/render";

const mocks = vi.hoisted(() => ({
  tags: vi.fn(),
  entityTags: vi.fn(),
  attach: vi.fn(),
  detach: vi.fn(),
}));

vi.mock("@/lib/hooks/queries/use-crm-queries", () => ({
  useTagsQuery: () => ({ data: mocks.tags() }),
  useEntityTagsQuery: () => ({ data: mocks.entityTags() }),
}));

vi.mock("@/lib/hooks/mutations/use-crm-mutations", () => ({
  useTagEntity: () => ({ mutate: mocks.attach }),
  useUntagEntity: () => ({ mutate: mocks.detach }),
}));

import { TagPicker } from "./tag-picker";

describe("TagPicker", () => {
  beforeEach(() => {
    mocks.tags.mockReturnValue([
      { id: 1, nombre: "VIP" },
      { id: 2, nombre: "Nuevo" },
    ]);
    mocks.entityTags.mockReturnValue([{ id: 1, nombre: "VIP" }]);
  });

  it("renders attached tags and offers remaining tags", () => {
    renderWithProviders(<TagPicker entityType="contact" entityId={1} />);
    const attached = screen.getByTestId("entity-tag");
    expect(within(attached).getByText("VIP")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Asignar etiqueta" })).toBeInTheDocument();
  });

  it("detaches a tag via the Quitar action", async () => {
    const user = userEvent.setup();
    renderWithProviders(<TagPicker entityType="contact" entityId={1} />);
    await user.click(screen.getByRole("button", { name: "Quitar" }));
    expect(mocks.detach).toHaveBeenCalledWith(
      { entityType: "contact", entityId: 1, tagId: 1 },
      expect.anything()
    );
  });

  it("attaches a selected tag", async () => {
    const user = userEvent.setup();
    renderWithProviders(<TagPicker entityType="contact" entityId={1} />);
    await user.click(screen.getByRole("button", { name: "Asignar etiqueta" }));
    const select = screen.getByRole("combobox", { name: "Seleccionar etiqueta" });
    await user.selectOptions(select, "2");
    expect(mocks.attach).toHaveBeenCalledWith(
      { entityType: "contact", entityId: 1, tagId: 2 },
      expect.anything()
    );
  });

  it("shows a disabled state when no tags remain", () => {
    mocks.tags.mockReturnValue([{ id: 1, nombre: "VIP" }]);
    mocks.entityTags.mockReturnValue([{ id: 1, nombre: "VIP" }]);
    renderWithProviders(<TagPicker entityType="contact" entityId={1} />);
    const btn = screen.getByRole("button", { name: "Sin etiquetas disponibles" });
    expect(btn).toBeDisabled();
  });
});
