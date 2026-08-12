import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { ConversationHeader } from "./conversation-header";
import type { Conversation } from "@/lib/models/conversation.model";

// Mocks de hooks externos: el header contrata con useMemberDirectoryQuery y
// useUpdateConversationAssignee (conversation-row-scoping).
vi.mock("@/lib/hooks/queries/use-member-directory-query", () => ({
  useMemberDirectoryQuery: () => ({
    data: undefined,
    isLoading: false,
    isError: true,
    refetch: vi.fn(),
    isRefetching: false,
  }),
}));
vi.mock("@/lib/hooks/mutations/use-update-conversation-assignee", () => ({
  useUpdateConversationAssignee: () => ({ mutate: vi.fn(), isPending: false }),
}));
vi.mock("@/lib/hooks/use-entitlement", () => ({
  useModule: () => ({ enabled: false }),
}));
vi.mock("@/lib/hooks/queries/use-conversation-context-query", () => ({
  useConversationContextQuery: () => ({ data: undefined }),
}));
vi.mock("@/lib/hooks/mutations/use-tickets-mutations", () => ({
  useCreateTicket: () => ({ mutate: vi.fn(), isPending: false }),
}));

const base: Conversation = {
  id: 1,
  organizationId: 7,
  contactId: 10,
  channel: "whatsapp",
  status: "active",
  createdAt: "2026-08-01T00:00:00Z",
  updatedAt: "2026-08-01T00:00:00Z",
  contactPhone: "+573001234567",
  contactDisplayName: "Cliente",
};

describe("ConversationHeader assignee picker (conversation-row-scoping)", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("oculta el picker en free tier (flag off) aunque haya permiso", () => {
    render(
      <ConversationHeader
        conversation={base}
        onToggleStatus={vi.fn()}
        isUpdating={false}
        canManage
        canReassign
        rowScopingEnabled={false}
      />
    );
    // Sin flag: sin chip de assignee ni controles de scope (hide, no ghost).
    expect(screen.queryByText("Sin asignar")).toBeNull();
  });

  it("oculta el picker sin permiso inbox:reassign (hide, no ghost)", () => {
    render(
      <ConversationHeader
        conversation={base}
        onToggleStatus={vi.fn()}
        isUpdating={false}
        canManage
        canReassign={false}
        rowScopingEnabled
      />
    );
    expect(screen.queryByText("Sin asignar")).toBeNull();
  });

  it("muestra estado de retry cuando el directorio no está disponible (503)", () => {
    render(
      <ConversationHeader
        conversation={base}
        onToggleStatus={vi.fn()}
        isUpdating={false}
        canManage
        canReassign
        rowScopingEnabled
      />
    );
    // Abrir el picker: con el directorio en error, el picker ofrece retry y
    // el thread/composer (encabezado) permanecen operativos.
    fireEvent.click(screen.getByText("Sin asignar"));
    expect(screen.getByText("El directorio de miembros no está disponible.")).toBeDefined();
    expect(screen.getByText("Reintentar")).toBeDefined();
    // El resto del encabezado sigue operativo (estado de la conversación).
    expect(screen.getByText("active")).toBeDefined();
  });
});
