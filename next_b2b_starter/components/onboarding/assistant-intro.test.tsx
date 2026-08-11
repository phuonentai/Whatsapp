import { describe, it, expect, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";

import { AssistantIntro } from "./assistant-intro";

describe("AssistantIntro", () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it("renders the assistant explanation and links to agent settings", () => {
    render(<AssistantIntro />);

    expect(screen.getByText("Conoce a tu asistente")).toBeDefined();
    expect(screen.getByText("Base de conocimiento")).toBeDefined();
    expect(screen.getByText("Configurar asistente").closest("a")).toHaveAttribute(
      "href",
      "/dashboard/settings?view=ai"
    );
  });

  it("hides after dismissal and persists the flag", () => {
    const { unmount } = render(<AssistantIntro />);
    expect(screen.getByText("Conoce a tu asistente")).toBeDefined();

    fireEvent.click(screen.getByText("Más tarde"));

    expect(screen.queryByText("Conoce a tu asistente")).toBeNull();
    expect(localStorage.getItem("ai-onboarding.assistant-intro-dismissed")).toBe("true");

    unmount();
    render(<AssistantIntro />);
    expect(screen.queryByText("Conoce a tu asistente")).toBeNull();
  });

  it("does not render when already dismissed", () => {
    localStorage.setItem("ai-onboarding.assistant-intro-dismissed", "true");

    const { container } = render(<AssistantIntro />);
    expect(container.firstChild).toBeNull();
  });
});
