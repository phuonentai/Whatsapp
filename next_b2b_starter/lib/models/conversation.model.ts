export type ConversationStatus = "active" | "closed" | "archived";

export type Channel = "whatsapp" | "instagram";

export interface Conversation {
  id: number;
  organizationId: number;
  contactId: number;
  channel: Channel;
  status: ConversationStatus;
  lastMessageAt?: string;
  metadata?: Record<string, unknown>;
  createdAt: string;
  updatedAt: string;
  contactPhone: string;
  contactDisplayName: string;
  contactInstagramUsername?: string;
  contactAvatarUrl?: string;
  /** stytch_member_id del miembro asignado (conversation-row-scoping);
   * undefined = conversación sin asignar (cola). */
  assigneeStytchMemberId?: string;
}

/** Vistas de scope de la bandeja (parámetro `scope` de GET /crm/conversaciones). */
export type ConversationScopeView = "" | "mine" | "queue" | "all";

export interface ConversationListResponse {
  success: boolean;
  data: Conversation[];
}

export interface Message {
  id: number;
  organizationId: number;
  conversationId: number;
  contactId: number;
  channel: Channel;
  providerMessageId?: string;
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
