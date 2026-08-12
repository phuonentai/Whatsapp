import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen } from "@testing-library/react";
import { renderWithProviders } from "@/test/render";
import { ChatMessage } from "./chat-message";
import { useDocuments } from "@/lib/hooks/queries/use-documents-query";
import type { ChatMessage as ChatMessageType } from "@/lib/models/cognitive.model";

vi.mock("@/lib/hooks/queries/use-documents-query", () => ({
  useDocuments: vi.fn(() => []),
}));

const mockUseDocuments = vi.mocked(useDocuments);

function makeMessage(overrides: Partial<ChatMessageType>): ChatMessageType {
  return {
    id: 1,
    sessionId: 1,
    role: "assistant",
    content: "",
    tokensUsed: 0,
    createdAt: new Date("2026-01-01T10:00:00Z"),
    ...overrides,
  };
}

describe("ChatMessage", () => {
  beforeEach(() => {
    mockUseDocuments.mockReset();
    mockUseDocuments.mockReturnValue([]);
  });

  it("renders assistant markdown through the shared renderer", () => {
    renderWithProviders(
      <ChatMessage
        message={makeMessage({ content: "# Resumen\n\n**Punto clave** con [enlace](https://ejemplo.com)." })}
      />
    );

    expect(screen.getByRole("heading", { level: 1, name: "Resumen" })).toBeDefined();
    expect(screen.getByText("Punto clave")).toHaveProperty("tagName", "STRONG");
    expect(screen.getByRole("link", { name: "enlace" })).toBeDefined();
  });

  it("exposes the assistant message container as aria-live polite", () => {
    const { container } = renderWithProviders(
      <ChatMessage message={makeMessage({ content: "Hola" })} />
    );

    const liveRegion = container.querySelector('[aria-live="polite"]');
    expect(liveRegion).not.toBeNull();
  });

  it("offers a copy button on assistant messages", () => {
    renderWithProviders(<ChatMessage message={makeMessage({ content: "Hola" })} />);

    expect(screen.getByRole("button", { name: "Copiar" })).toBeDefined();
  });

  it("keeps raw text rendering for user messages", () => {
    const { container } = renderWithProviders(
      <ChatMessage
        message={makeMessage({ role: "user", content: "**no es markdown** en el usuario" })}
      />
    );

    // User bubble must not contain a STRONG element.
    expect(container.querySelector("strong")).toBeNull();
    expect(screen.getByText("**no es markdown** en el usuario")).toBeDefined();
  });
});
