import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";

import {
  PostConnectSteps,
  POST_CONNECT_DISMISS_KEY,
  clearPostConnectDismissed,
} from "./post-connect-steps";

describe("PostConnectSteps", () => {
  beforeEach(() => {
    localStorage.clear();
    vi.restoreAllMocks();
  });

  it("renders the five next-steps items with links to inbox, compliance, and agent settings", () => {
    render(<PostConnectSteps />);

    expect(screen.getByText("Siguientes pasos")).toBeDefined();
    expect(screen.getByText("Envía un mensaje de prueba")).toBeDefined();
    expect(screen.getByText("Solo llegan mensajes nuevos")).toBeDefined();
    expect(screen.getByText("Consentimiento de datos (Ley 1581)")).toBeDefined();

    expect(screen.getByText("Ver la bandeja de entrada").closest("a")).toHaveAttribute(
      "href",
      "/dashboard/inbox"
    );
    expect(screen.getByText("Activar el asistente").closest("a")).toHaveAttribute(
      "href",
      "/dashboard/settings?view=ai"
    );
    expect(screen.getByText("Ir a cumplimiento →").closest("a")).toHaveAttribute(
      "href",
      "/dashboard/settings?view=compliance"
    );
  });

  it("dismisses persistently without invoking any API", () => {
    const fetchSpy = vi.spyOn(globalThis, "fetch");
    const { unmount } = render(<PostConnectSteps />);
    expect(fetchSpy).not.toHaveBeenCalled();

    fireEvent.click(screen.getByLabelText("Descartar"));

    expect(screen.queryByText("Siguientes pasos")).toBeNull();
    expect(localStorage.getItem(POST_CONNECT_DISMISS_KEY)).toBe("true");
    expect(fetchSpy).not.toHaveBeenCalled();

    // Stays dismissed on a fresh mount (persisted).
    unmount();
    render(<PostConnectSteps />);
    expect(screen.queryByText("Siguientes pasos")).toBeNull();
    expect(fetchSpy).not.toHaveBeenCalled();
  });

  it("does not render when already dismissed", () => {
    localStorage.setItem(POST_CONNECT_DISMISS_KEY, "true");

    const { container } = render(<PostConnectSteps />);
    expect(container.firstChild).toBeNull();
  });

  it("reappears after the dismissal flag is cleared (reactivation)", () => {
    const { unmount } = render(<PostConnectSteps />);
    fireEvent.click(screen.getByLabelText("Descartar"));
    expect(screen.queryByText("Siguientes pasos")).toBeNull();

    unmount();
    clearPostConnectDismissed();
    render(<PostConnectSteps />);
    expect(screen.getByText("Siguientes pasos")).toBeDefined();
  });
});
