import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { toast } from "sonner";
import type { UseMutationResult, UseQueryResult } from "@tanstack/react-query";
import { renderWithProviders } from "@/test/render";
import type { ActivityDto } from "@/lib/api/api/dto/crm.dto";
import { ConversationContextPanel } from "./conversation-context-panel";
import { useConversationContextQuery } from "@/lib/hooks/queries/use-conversation-context-query";
import { useCreateActivity } from "@/lib/hooks/mutations/use-crm-mutations";
import type { ConversationContext } from "@/lib/models/agent.model";

vi.mock("@/lib/hooks/queries/use-conversation-context-query", () => ({
  useConversationContextQuery: vi.fn(),
}));

vi.mock("@/lib/hooks/mutations/use-crm-mutations", () => ({
  useCreateActivity: vi.fn(),
}));

type CreateActivityMutation = UseMutationResult<ActivityDto, Error, Partial<ActivityDto>>;

const mockQuery = vi.mocked(useConversationContextQuery);
const mockUseCreateActivity = vi.mocked(useCreateActivity);
const mockCreateActivityMutate = vi.fn();

function makeContext(overrides: Partial<ConversationContext> = {}): ConversationContext {
  return {
    conversationId: 10,
    summary: "El cliente pidió una cotización para 5 unidades.",
    detectedIntent: "cotización",
    keyFacts: ["Pidió cotización", "Cliente de Bogotá"],
    sourceCursor: 42,
    consentGated: false,
    status: "available",
    channel: "whatsapp",
    messageCount: 4,
    ...overrides,
  };
}

function queryResult(
  data?: ConversationContext,
  isLoading = false,
  isError = false
): UseQueryResult<ConversationContext, Error> {
  return { data, isLoading, isError, isFetching: false, isPending: isLoading } as unknown as UseQueryResult<
    ConversationContext,
    Error
  >;
}

function mockMutation({ isPending = false } = {}) {
  mockUseCreateActivity.mockReturnValue({
    mutate: mockCreateActivityMutate,
    isPending,
  } as unknown as CreateActivityMutation);
}

describe("ConversationContextPanel", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockMutation();
  });

  it("renders summary, intent, and key facts when context is available", () => {
    mockQuery.mockReturnValue(queryResult(makeContext()));
    renderWithProviders(<ConversationContextPanel conversationId={10} />);
    expect(screen.getByText(/El cliente pidió una cotización para 5 unidades\./)).toBeInTheDocument();
    expect(screen.getByText(/Intención: cotización/)).toBeInTheDocument();
    expect(screen.getByText("Pidió cotización")).toBeInTheDocument();
  });

  it("renders a skeleton while loading", () => {
    mockQuery.mockReturnValue(queryResult(undefined, true));
    const { container } = renderWithProviders(<ConversationContextPanel conversationId={10} />);
    expect(container.querySelector(".animate-pulse")).not.toBeNull();
    expect(screen.queryByText("El asistente está aprendiendo…")).not.toBeInTheDocument();
  });

  it("renders the learning state when context is unavailable", () => {
    mockQuery.mockReturnValue(queryResult(makeContext({ status: "unavailable" })));
    renderWithProviders(<ConversationContextPanel conversationId={10} />);
    expect(screen.getByText("El asistente está aprendiendo…")).toBeInTheDocument();
  });

  it("renders consent-gated structural context with counts and channel", () => {
    mockQuery.mockReturnValue(
      queryResult(
        makeContext({
          consentGated: true,
          status: "structural",
          summary: undefined,
          detectedIntent: undefined,
          keyFacts: [],
          messageCount: 4,
          channel: "whatsapp",
        })
      )
    );
    renderWithProviders(<ConversationContextPanel conversationId={10} />);
    expect(screen.getByText("Contexto estructural")).toBeInTheDocument();
    expect(screen.getByText("4")).toBeInTheDocument();
    expect(screen.getByText("WhatsApp")).toBeInTheDocument();
    expect(screen.queryByText(/El cliente pidió una cotización/)).not.toBeInTheDocument();
  });

  it("renders the learning state on query error", () => {
    mockQuery.mockReturnValue(queryResult(undefined, false, true));
    renderWithProviders(<ConversationContextPanel conversationId={10} />);
    expect(screen.getByText("El asistente está aprendiendo…")).toBeInTheDocument();
  });

  it("renders the save action only on full context with a contact id", () => {
    mockQuery.mockReturnValue(queryResult(makeContext()));
    renderWithProviders(<ConversationContextPanel conversationId={10} contactId={42} />);
    expect(screen.getByRole("button", { name: "Guardar como nota" })).toBeInTheDocument();
  });

  it("hides the save action when contact id is missing", () => {
    mockQuery.mockReturnValue(queryResult(makeContext()));
    renderWithProviders(<ConversationContextPanel conversationId={10} />);
    expect(screen.queryByRole("button", { name: "Guardar como nota" })).not.toBeInTheDocument();
  });

  it("hides the save action in loading, unavailable, consent-gated, and error states", () => {
    const contactId = 42;

    mockQuery.mockReturnValue(queryResult(undefined, true));
    const loading = renderWithProviders(<ConversationContextPanel conversationId={10} contactId={contactId} />);
    expect(screen.queryByRole("button", { name: "Guardar como nota" })).not.toBeInTheDocument();
    loading.unmount();

    mockQuery.mockReturnValue(queryResult(makeContext({ status: "unavailable" })));
    const unavailable = renderWithProviders(<ConversationContextPanel conversationId={10} contactId={contactId} />);
    expect(screen.queryByRole("button", { name: "Guardar como nota" })).not.toBeInTheDocument();
    unavailable.unmount();

    mockQuery.mockReturnValue(
      queryResult(
        makeContext({
          consentGated: true,
          status: "structural",
          summary: undefined,
          detectedIntent: undefined,
          keyFacts: [],
        })
      )
    );
    const consent = renderWithProviders(<ConversationContextPanel conversationId={10} contactId={contactId} />);
    expect(screen.queryByRole("button", { name: "Guardar como nota" })).not.toBeInTheDocument();
    consent.unmount();

    mockQuery.mockReturnValue(queryResult(undefined, false, true));
    renderWithProviders(<ConversationContextPanel conversationId={10} contactId={contactId} />);
    expect(screen.queryByRole("button", { name: "Guardar como nota" })).not.toBeInTheDocument();
  });

  it("creates the activity with the correct contact id and content when saving", async () => {
    const user = userEvent.setup();
    mockQuery.mockReturnValue(queryResult(makeContext()));
    renderWithProviders(<ConversationContextPanel conversationId={10} contactId={42} />);

    await user.click(screen.getByRole("button", { name: "Guardar como nota" }));

    expect(mockCreateActivityMutate).toHaveBeenCalledWith(
      {
        contact_id: 42,
        tipo: "nota",
        asunto: "Resumen IA de la conversación",
        contenido:
          "Resumen: El cliente pidió una cotización para 5 unidades.\n" +
          "Intención: cotización\n" +
          "Datos clave: Pidió cotización; Cliente de Bogotá",
      },
      expect.any(Object)
    );
  });

  it("omits missing context sections from the note content", async () => {
    const user = userEvent.setup();
    mockQuery.mockReturnValue(
      queryResult(makeContext({ summary: undefined, detectedIntent: undefined, keyFacts: ["Solo un dato"] }))
    );
    renderWithProviders(<ConversationContextPanel conversationId={10} contactId={42} />);

    await user.click(screen.getByRole("button", { name: "Guardar como nota" }));

    expect(mockCreateActivityMutate).toHaveBeenCalledWith(
      {
        contact_id: 42,
        tipo: "nota",
        asunto: "Resumen IA de la conversación",
        contenido: "Datos clave: Solo un dato",
      },
      expect.any(Object)
    );
  });

  it("shows an error toast and keeps the action available for retry on failure", async () => {
    const user = userEvent.setup();
    mockQuery.mockReturnValue(queryResult(makeContext()));
    mockCreateActivityMutate.mockImplementation((_payload: unknown, options?: { onError?: (error: Error) => void }) => {
      options?.onError?.(new Error("boom"));
    });
    renderWithProviders(<ConversationContextPanel conversationId={10} contactId={42} />);

    await user.click(screen.getByRole("button", { name: "Guardar como nota" }));

    expect(toast.error).toHaveBeenCalledWith("No se pudo guardar la nota. Inténtalo de nuevo.");
    expect(toast.success).not.toHaveBeenCalled();
    expect(screen.getByRole("button", { name: "Guardar como nota" })).toBeEnabled();
  });

  it("disables the action and confirms via toast after a successful save", async () => {
    const user = userEvent.setup();
    mockQuery.mockReturnValue(queryResult(makeContext()));
    mockCreateActivityMutate.mockImplementation((_payload: unknown, options?: { onSuccess?: () => void }) => {
      options?.onSuccess?.();
    });
    renderWithProviders(<ConversationContextPanel conversationId={10} contactId={42} />);

    await user.click(screen.getByRole("button", { name: "Guardar como nota" }));

    expect(toast.success).toHaveBeenCalledWith("Nota guardada en el contacto");
    expect(screen.getByRole("button", { name: "Guardar como nota" })).toBeDisabled();
  });

  it("disables the action while the mutation is pending", () => {
    mockQuery.mockReturnValue(queryResult(makeContext()));
    mockMutation({ isPending: true });
    renderWithProviders(<ConversationContextPanel conversationId={10} contactId={42} />);
    expect(screen.getByRole("button", { name: "Guardar como nota" })).toBeDisabled();
  });
});
