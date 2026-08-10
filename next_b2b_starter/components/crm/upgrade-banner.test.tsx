import { describe, it, expect } from "vitest";
import { screen } from "@testing-library/react";
import { renderWithProviders } from "@/test/render";
import { UpgradeBanner } from "./upgrade-banner";

describe("UpgradeBanner", () => {
  it("shows the gated feature, required plan and upgrade link", () => {
    renderWithProviders(<UpgradeBanner feature="Negocios" plan="Pro" />);
    expect(screen.getByText("Negocios es una funcionalidad Pro")).toBeInTheDocument();
    expect(screen.getByText("Actualiza tu plan para acceder a esta funcionalidad.")).toBeInTheDocument();
    const link = screen.getByRole("link", { name: "Actualizar a Pro" });
    expect(link).toHaveAttribute("href", "/dashboard/settings?view=suscripcion");
  });
});
