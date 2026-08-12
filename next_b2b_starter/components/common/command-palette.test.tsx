import { describe, it, expect, beforeEach, vi } from "vitest";
import { act, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { CommandPalette } from "./command-palette";
import { KnowledgeContent } from "@/app/dashboard/knowledge/components/knowledge-content";
import { renderWithProviders } from "@/test/render";
import { crmRepository } from "@/lib/api/api/repositories/crm-repository";
import { useCommandPaletteStore } from "@/lib/stores/command-palette-store";
import type { ContactDto } from "@/lib/api/api/dto/crm.dto";

const pushMock = vi.fn();

vi.mock("next/navigation", () => ({
  useRouter: () => ({ push: pushMock }),
  usePathname: () => "/dashboard",
  useSearchParams: () => new URLSearchParams(),
}));

vi.mock("@/lib/api/api/repositories/crm-repository", () => ({
  crmRepository: {
    searchContacts: vi.fn(),
  },
}));

// Knowledge bridge: `knowledge-content.tsx` subscribes to the palette store's
// `aiNewChatSignal`. Mock its data hooks so the component renders standalone.
// Session data must be a stable reference across renders: the component's
// auto-select runs setState during render when the sessions reference changes.
vi.mock("@/lib/hooks/queries/use-sessions-query", () => {
  const sessions = [
    {
      id: 1,
      title: "Mi primera consulta",
      createdAt: new Date("2026-01-01T10:00:00Z"),
      updatedAt: new Date("2026-01-01T10:00:00Z"),
    },
  ];
  return {
    useSessionsQuery: () => ({ data: sessions, isLoading: false }),
    useSessionMessagesQuery: () => ({ data: [], isLoading: false }),
  };
});

vi.mock("@/lib/hooks/mutations/use-chat-stream", () => ({
  useChatStream: () => ({ sendMessage: vi.fn(), isStreaming: false }),
}));

vi.mock("@/lib/hooks/queries/use-documents-query", () => {
  const documentsData = { documents: [], total: 0, limit: 50, offset: 0 };
  return {
    useDocumentsQuery: () => ({
      data: documentsData,
      isLoading: false,
      isFetching: false,
      isError: false,
      refetch: vi.fn(),
    }),
  };
});

vi.mock("@/lib/hooks/mutations/use-upload-document", () => ({
  useUploadDocument: () => ({ mutateAsync: vi.fn(), isPending: false }),
}));

vi.mock("@/lib/hooks/mutations/use-delete-document", () => ({
  useDeleteDocument: () => ({ mutateAsync: vi.fn(), isPending: false }),
}));

const mockSearchContacts = vi.mocked(crmRepository.searchContacts);

function openPalette(mode: "command" | "search" = "command") {
  useCommandPaletteStore.getState().openPalette(mode);
}

function closePalette() {
  useCommandPaletteStore.getState().closePalette();
}

describe("CommandPalette", () => {
  beforeEach(() => {
    pushMock.mockReset();
    mockSearchContacts.mockReset();
    closePalette();
  });

  it("opens and focuses the input", async () => {
    const user = userEvent.setup();
    renderWithProviders(<CommandPalette />);

    openPalette();

    const input = await screen.findByPlaceholderText(
      "Type a command or search…"
    );
    expect(input).toBeDefined();
    expect(useCommandPaletteStore.getState().open).toBe(true);
  });

  it("filters navigation targets by fuzzy text match", async () => {
    const user = userEvent.setup();
    renderWithProviders(<CommandPalette />);
    openPalette();

    const input = await screen.findByPlaceholderText(
      "Type a command or search…"
    );
    await user.type(input, "contactos");

    expect(screen.getByText("CRM")).toBeDefined();
    expect(screen.queryByText("Inbox")).toBeNull();
  });

  it("navigates to the selected destination on Enter", async () => {
    const user = userEvent.setup();
    renderWithProviders(<CommandPalette />);
    openPalette();

    const input = await screen.findByPlaceholderText(
      "Type a command or search…"
    );
    await user.type(input, "inbox");
    await user.keyboard("{Enter}");

    await waitFor(() => {
      expect(pushMock).toHaveBeenCalledWith("/dashboard/inbox");
    });
    expect(useCommandPaletteStore.getState().open).toBe(false);
  });

  it("searches contacts in search mode and opens the contact on Enter", async () => {
    const user = userEvent.setup();
    const contact: ContactDto = {
      id: 42,
      organization_id: 7,
      phone_number: "+573001112233",
      display_name: "Maria Lopez",
      source: "whatsapp",
      lead_status: "nuevo",
      is_blocked: false,
      created_at: "2026-01-01T00:00:00Z",
      updated_at: "2026-01-01T00:00:00Z",
    };
    mockSearchContacts.mockResolvedValue({ items: [contact], total: 1 });

    renderWithProviders(<CommandPalette />);
    openPalette("search");

    const input = await screen.findByPlaceholderText("Search contacts…");
    await user.type(input, "maria");

    await waitFor(() => {
      expect(screen.getByText("Maria Lopez")).toBeDefined();
    });

    await user.keyboard("{Enter}");

    await waitFor(() => {
      expect(pushMock).toHaveBeenCalledWith(
        "/dashboard/crm?view=contactos&id=42"
      );
    });
    expect(mockSearchContacts).toHaveBeenCalledWith(
      "maria",
      expect.objectContaining({ limit: 20 })
    );
  });

  it("shows a no-results state when the search returns nothing", async () => {
    const user = userEvent.setup();
    mockSearchContacts.mockResolvedValue({ items: [], total: 0 });

    renderWithProviders(<CommandPalette />);
    openPalette("search");

    const input = await screen.findByPlaceholderText("Search contacts…");
    await user.type(input, "zzzzzz");

    await waitFor(() => {
      expect(screen.getByText("No results")).toBeDefined();
    });
  });

  it("shows the IA group when typing a matching AI keyword", async () => {
    const user = userEvent.setup();
    renderWithProviders(<CommandPalette />);
    openPalette();

    const input = await screen.findByPlaceholderText(
      "Type a command or search…"
    );
    await user.type(input, "asistente");

    expect(screen.getByText("IA")).toBeDefined();
    expect(screen.getByText("Preguntar al asistente")).toBeDefined();
  });

  it("navigates to the knowledge page when selecting 'Preguntar al asistente'", async () => {
    const user = userEvent.setup();
    renderWithProviders(<CommandPalette />);
    openPalette();

    const input = await screen.findByPlaceholderText(
      "Type a command or search…"
    );
    await user.type(input, "preguntar");
    await user.keyboard("{Enter}");

    await waitFor(() => {
      expect(pushMock).toHaveBeenCalledWith("/dashboard/knowledge");
    });
    expect(useCommandPaletteStore.getState().open).toBe(false);
  });

  it("navigates AND increments the new-chat signal when selecting 'Nueva conversación de IA'", async () => {
    const user = userEvent.setup();
    renderWithProviders(<CommandPalette />);
    openPalette();

    const input = await screen.findByPlaceholderText(
      "Type a command or search…"
    );
    await user.type(input, "nueva conversación");
    await user.keyboard("{Enter}");

    await waitFor(() => {
      expect(pushMock).toHaveBeenCalledWith("/dashboard/knowledge");
    });
    expect(useCommandPaletteStore.getState().open).toBe(false);
    expect(useCommandPaletteStore.getState().aiNewChatSignal).toBe(1);
  });

  it("navigates to the CRM campaigns view when selecting the audience action", async () => {
    const user = userEvent.setup();
    renderWithProviders(<CommandPalette />);
    openPalette();

    const input = await screen.findByPlaceholderText(
      "Type a command or search…"
    );
    await user.type(input, "audiencia");
    await user.keyboard("{Enter}");

    await waitFor(() => {
      expect(pushMock).toHaveBeenCalledWith("/dashboard/crm?view=campanas");
    });
    expect(useCommandPaletteStore.getState().open).toBe(false);
  });

  it("keeps existing navigation commands working alongside the IA group", async () => {
    const user = userEvent.setup();
    renderWithProviders(<CommandPalette />);
    openPalette();

    const input = await screen.findByPlaceholderText(
      "Type a command or search…"
    );
    // "knowledge" matches both the AI assistant action and the Knowledge Base
    // nav destination; both must stay discoverable.
    await user.type(input, "knowledge");

    expect(screen.getByText("Preguntar al asistente")).toBeDefined();
    expect(screen.getByText("Knowledge Base")).toBeDefined();

    await user.keyboard("{Enter}");

    await waitFor(() => {
      expect(pushMock).toHaveBeenCalledWith("/dashboard/knowledge");
    });
  });

  it("resets the knowledge chat when the new-AI-chat signal increments", async () => {
    const user = userEvent.setup();
    renderWithProviders(<KnowledgeContent />);

    // Fresh chat state initially: "Nueva conversación" header title + sidebar button.
    expect(screen.getAllByText("Nueva conversación")).toHaveLength(2);

    // Select the existing session: the header title mirrors the session title.
    await user.click(screen.getByText("Mi primera consulta"));
    await waitFor(() => {
      expect(screen.getAllByText("Nueva conversación")).toHaveLength(1);
    });
    expect(screen.getAllByText("Mi primera consulta")).toHaveLength(2);

    // Palette's new-chat action → signal increments → handleNewChat resets.
    act(() => {
      useCommandPaletteStore.getState().requestNewAiChat();
    });

    await waitFor(() => {
      expect(screen.getAllByText("Mi primera consulta")).toHaveLength(1);
    });
    expect(screen.getAllByText("Nueva conversación")).toHaveLength(2);
  });
});
