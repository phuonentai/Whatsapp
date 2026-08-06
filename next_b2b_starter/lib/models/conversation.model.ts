export type ConversationStatus = "active" | "closed" | "archived";

export interface Conversation {
  id: number;
  organizationId: number;
  contactId: number;
  status: ConversationStatus;
  lastMessageAt?: string;
  metadata?: Record<string, unknown>;
  createdAt: string;
  updatedAt: string;
  contactPhone: string;
  contactDisplayName: string;
}

export interface ConversationListResponse {
  success: boolean;
  data: Conversation[];
}

export interface Message {
  id: number;
  organizationId: number;
  conversationId: number;
  contactId: number;
  whatsappMessageId?: string;
  direction: "inbound" | "outbound";
  messageType: string;
  content?: string;
  status: string;
  chatTimestamp?: string;
  createdAt: string;
  updatedAt: string;
}

export interface MessageListResponse {
  success: boolean;
  data: Message[];
}
