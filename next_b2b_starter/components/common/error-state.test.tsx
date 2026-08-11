import { describe, it, expect, vi } from "vitest";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithProviders } from "@/test/render";
import { ErrorState } from "./error-state";

describe("ErrorState", () => {
  it("renders title and description", () => {
    renderWithProviders(
      <ErrorState title="Error al cargar los contactos" description="Inténtalo de nuevo." />
    );
    expect(screen.getByRole("alert")).toHaveTextContent("Error al cargar los contactos");
    expect(screen.getByText("Inténtalo de nuevo.")).toBeInTheDocument();
  });

  it("calls onRetry when retry button is clicked", async () => {
    const user = userEvent.setup();
    const onRetry = vi.fn();
    renderWithProviders(<ErrorState title="Error" onRetry={onRetry} />);
    await user.click(screen.getByRole("button", { name: /reintentar/i }));
    expect(onRetry).toHaveBeenCalledTimes(1);
  });

  it("shows retrying state and disables the button", () => {
    renderWithProviders(<ErrorState title="Error" onRetry={vi.fn()} isRetrying />);
    const button = screen.getByRole("button", { name: /reintentando/i });
    expect(button).toBeDisabled();
  });
});
