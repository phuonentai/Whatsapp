import { describe, it, expect } from "vitest";
import { isConversationUnread } from "./unread";
import type { Conversation } from "@/lib/models/conversation.model";

function makeConversation(overrides: Partial<Conversation>): Conversation {
  return {
    id: 1,
    organizationId: 1,
    contactId: 1,
    channel: "whatsapp",
    status: "active",
    contactPhone: "+573000000000",
    contactDisplayName: "Cliente",
    createdAt: "2024-01-01T00:00:00Z",
    updatedAt: "2024-01-01T00:00:00Z",
    ...overrides,
  } as Conversation;
}

describe("isConversationUnread", () => {
  it("is unread when never seen and has a last message", () => {
    const conv = makeConversation({ id: 5, lastMessageAt: "2024-06-01T10:00:00Z" });
    expect(isConversationUnread(conv, {})).toBe(true);
  });

  it("is not unread when never seen and has no last message", () => {
    const conv = makeConversation({ id: 5, lastMessageAt: undefined });
    expect(isConversationUnread(conv, {})).toBe(false);
  });

  it("is unread when last message is newer than lastSeenAt", () => {
    const conv = makeConversation({ id: 5, lastMessageAt: "2024-06-01T10:00:00Z" });
    expect(isConversationUnread(conv, { 5: new Date("2024-06-01T09:00:00Z").getTime() })).toBe(true);
  });

  it("is read when last message is older than lastSeenAt", () => {
    const conv = makeConversation({ id: 5, lastMessageAt: "2024-06-01T09:00:00Z" });
    expect(isConversationUnread(conv, { 5: new Date("2024-06-01T10:00:00Z").getTime() })).toBe(false);
  });

  it("ignores an invalid timestamp", () => {
    const conv = makeConversation({ id: 5, lastMessageAt: "not-a-date" });
    expect(isConversationUnread(conv, {})).toBe(false);
  });
});
