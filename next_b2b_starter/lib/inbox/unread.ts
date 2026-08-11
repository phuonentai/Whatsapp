import type { Conversation } from "@/lib/models/conversation.model";

export function isConversationUnread(
  conversation: Conversation,
  lastSeenAt: Record<number, number>
): boolean {
  if (!conversation.lastMessageAt) return false;
  const lastMessageTs = new Date(conversation.lastMessageAt).getTime();
  if (Number.isNaN(lastMessageTs)) return false;
  const seen = lastSeenAt[conversation.id];
  if (seen == null) return true;
  return lastMessageTs > seen;
}
