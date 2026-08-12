import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { UseMutationResult, UseQueryResult } from "@tanstack/react-query";
import { renderWithProviders } from "@/test/render";
import { AgentSuggestionsPanel } from "./agent-suggestions-panel";
import { usePendingSuggestionsQuery } from "@/lib/hooks/queries/use-pending-suggestions-query";
import { useMessagesQuery } from "@/lib/hooks/queries/use-messages-query";
import { useRejectSuggestion } from "@/lib/hooks/mutations/use-agent-suggestion-mutations";
import { useAiUsageQuery } from "@/lib/hooks/queries/use-ai-usage-query";
import type { AgentSuggestion } from "@/lib/models/agent.model";
import type { Message } from "@/lib/models/conversation.model";

type PendingQueryResult = UseQueryResult<AgentSuggestion[], Error>;
type MessagesQueryResult = UseQueryResult<Message[], Error>;
type RejectMutationResult = UseMutationResult<AgentSuggestion, Error, number>;

vi.mock("@/lib/hooks/queries/use-pending-suggestions-query", () => ({
  usePendingSuggestionsQuery: vi.fn(),
}));

vi.mock("@/lib/hooks/queries/use-messages-query", () => ({
  useMessagesQuery: vi.fn(),
}));

vi.mock("@/lib/hooks/mutations/use-agent-suggestion-mutations", () => ({
  useApproveSuggestion: vi.fn(),
  useRejectSuggestion: vi.fn(),
}));

vi.mock("@/lib/hooks/queries/use-ai-usage-query", () => ({
  useAiUsageQuery: vi.fn(() => ({ data: undefined })),
}));

const mockPendingQuery = vi.mocked(usePendingSuggestionsQuery);
const mockMessagesQuery = vi.mocked(useMessagesQuery);
const mockReject = vi.mocked(useRejectSuggestion);
const mockAiUsage = vi.mocked(useAiUsageQuery);

function makeSuggestion(overrides: Partial<AgentSuggestion>): AgentSuggestion {
  return {
    id: 1,
    organization_id: 7,
    conversation_id: 10,
    contact_id: 20,
    type: "reply",
    body: "Hola, te confirmamos tu pedido.",
    status: "pending",
    source: "copilot",
    created_at: "2026-08-01T10:00:00Z",
    updated_at: "2026-08-01T10:00:00Z",
    ...overrides,
  };
}

function makeMessage(overrides: Partial<Message>): Message {
  return {
    id: 1,
    organizationId: 7,
    conversationId: 10,
    contactId: 20,
    channel: "whatsapp",
    direction: "inbound",
    messageType: "text",
    content: "¿Cuánto cuesta el envío?",
    status: "received",
    createdAt: "2026-08-01T09:59:00Z",
    updatedAt: "2026-08-01T09:59:00Z",
    ...overrides,
  };
}

function pendingQueryResult(overrides: Partial<PendingQueryResult>) {
  mockPendingQuery.mockReturnValue(overrides as PendingQueryResult);
}

describe("AgentSuggestionsPanel", () => {
  beforeEach(() => {
    mockPendingQuery.mockReset();
    mockMessagesQuery.mockReset();
    mockReject.mockReset();
    mockAiUsage.mockReset();
    mockAiUsage.mockReturnValue({ data: undefined } as never);
    mockMessagesQuery.mockReturnValue({
      data: undefined,
      isLoading: false,
    } as MessagesQueryResult);
    mockReject.mockReturnValue({
      mutateAsync: vi.fn().mockResolvedValue(undefined),
      isPending: false,
    } as unknown as RejectMutationResult);
  });

  it("shows a skeleton while suggestions load", () => {
    pendingQueryResult({ data: undefined, isLoading: true, isError: false });

    renderWithProviders(<AgentSuggestionsPanel conversationId={10} />);

    expect(screen.getByTestId("suggestions-skeleton")).toBeDefined();
    expect(screen.queryByRole("button", { name: "Aprobar" })).toBeNull();
  });

  it("approving prefills the composer instead of sending silently", async () => {
    const user = userEvent.setup();
    const onApproveAsDraft = vi.fn();
    const rejectAsync = vi.fn().mockResolvedValue(undefined);
    mockReject.mockReturnValue({
      mutateAsync: rejectAsync,
      isPending: false,
    } as unknown as RejectMutationResult);
    pendingQueryResult({
      data: [makeSuggestion({ id: 1, body: "Primera sugerencia" })],
      isLoading: false,
      isError: false,
    });

    renderWithProviders(
      <AgentSuggestionsPanel conversationId={10} onApproveAsDraft={onApproveAsDraft} />
    );

    await user.click(screen.getByRole("button", { name: "Aprobar" }));

    // Prefill only: the body lands in the composer and NO approve/reject API
    // is called — nothing is sent silently.
    expect(onApproveAsDraft).toHaveBeenCalledWith("Primera sugerencia");
    expect(rejectAsync).not.toHaveBeenCalled();
  });

  it("marks suggestions with the AI ✦ badge and escalation with the human-only note", () => {
    pendingQueryResult({
      data: [
        makeSuggestion({ id: 1, body: "Borrador de IA" }),
        makeSuggestion({ id: 2, type: "escalation", body: "Cliente molesto por demora" }),
      ],
      isLoading: false,
      isError: false,
    });

    renderWithProviders(<AgentSuggestionsPanel conversationId={10} />);

    expect(screen.getAllByText(/✦/).length).toBeGreaterThanOrEqual(2);
    expect(screen.getByText("requiere juicio humano")).toBeDefined();
    // Escalations never auto-send: only the prefill (Aprobar) affordance exists.
    expect(screen.getAllByRole("button", { name: "Aprobar" })).toHaveLength(2);
  });

  it("shows a one-line notice when AI credits are exhausted", () => {
    pendingQueryResult({
      data: [makeSuggestion({ id: 1, body: "Sugerencia" })],
      isLoading: false,
      isError: false,
    });
    mockAiUsage.mockReturnValue({
      data: { credits_used: 100, credits_max: 100, credits_remaining: 0 },
    } as never);

    renderWithProviders(<AgentSuggestionsPanel conversationId={10} />);

    expect(screen.getByTestId("suggestions-credits-notice")).toBeDefined();
  });

  it("expands the conversation-context thread excerpt with aria-expanded", async () => {
    const user = userEvent.setup();
    pendingQueryResult({
      data: [makeSuggestion({ id: 1, body: "Sugerencia con contexto" })],
      isLoading: false,
      isError: false,
    });
    mockMessagesQuery.mockReturnValue({
      data: [
        makeMessage({ id: 1, direction: "inbound", content: "Hola, ¿tienen el modelo X?" }),
        makeMessage({ id: 2, direction: "outbound", content: "Sí, te compartimos precio." }),
      ],
      isLoading: false,
    } as MessagesQueryResult);

    renderWithProviders(<AgentSuggestionsPanel conversationId={10} />);

    const toggle = screen.getByRole("button", { name: "Ver contexto de la conversación" });
    expect(toggle.getAttribute("aria-expanded")).toBe("false");
    expect(screen.queryByTestId("suggestion-context")).toBeNull();

    await user.click(toggle);

    expect(toggle.getAttribute("aria-expanded")).toBe("true");
    const context = screen.getByTestId("suggestion-context");
    expect(within(context).getByText("Hola, ¿tienen el modelo X?")).toBeDefined();
    expect(within(context).getByText("Sí, te compartimos precio.")).toBeDefined();
    expect(within(context).getByText("Cliente")).toBeDefined();
    expect(within(context).getByText("Nosotros")).toBeDefined();

    await user.click(toggle);
    expect(toggle.getAttribute("aria-expanded")).toBe("false");
    expect(screen.queryByTestId("suggestion-context")).toBeNull();
  });
});
