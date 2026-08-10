import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { ConversationList } from "./conversation-list";
import type { Conversation } from "@/lib/models/conversation.model";

const base: Conversation = {
  id: 1,
  organizationId: 7,
  contactId: 10,
  channel: "whatsapp",
  status: "active",
  createdAt: "2026-08-01T00:00:00Z",
  updatedAt: "2026-08-01T00:00:00Z",
  contactPhone: "+573001234567",
  contactDisplayName: "",
};

function renderList(conversations: Conversation[]) {
  return render(
    <ConversationList
      conversations={conversations}
      selectedId={undefined}
      onSelect={vi.fn()}
      isLoading={false}
      statusFilter=""
      onStatusFilterChange={vi.fn()}
      channelFilter="all"
      onChannelFilterChange={vi.fn()}
    />
  );
}

describe("ConversationList channel rendering", () => {
  it("shows IG username and @-subtitle for instagram conversations", () => {
    renderList([
      {
        ...base,
        id: 2,
        channel: "instagram",
        contactInstagramUsername: "cliente.ig",
        contactDisplayName: "Cliente IG",
        contactAvatarUrl: "https://cdn.example/pic.jpg",
      },
    ]);

    expect(screen.getByText("Cliente IG")).toBeDefined();
    expect(screen.getByText("@cliente.ig")).toBeDefined();
  });

  it("shows phone for whatsapp conversations", () => {
    renderList([{ ...base, contactDisplayName: "+573001234567" }]);
    expect(screen.getByText("+573001234567")).toBeDefined();
  });

  it("renders channel-specific empty state for instagram filter", () => {
    render(
      <ConversationList
        conversations={[]}
        selectedId={undefined}
        onSelect={vi.fn()}
        isLoading={false}
        statusFilter=""
        onStatusFilterChange={vi.fn()}
        channelFilter="instagram"
        onChannelFilterChange={vi.fn()}
      />
    );
    expect(
      screen.getByText("No Instagram messages yet — connect Instagram in Settings to get started")
    ).toBeDefined();
  });
});
