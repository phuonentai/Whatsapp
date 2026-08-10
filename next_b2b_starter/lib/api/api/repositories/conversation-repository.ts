import { apiClient } from "../client/api-client";
import { ConversationDto, MessageDto } from "../dto/conversation.dto";
import type { Conversation, Message, ConversationStatus, Channel } from "@/lib/models/conversation.model";

class ConversationRepository {
  async listConversations(
    params?: { status?: string; channel?: Channel; limit?: number; offset?: number }
  ): Promise<Conversation[]> {
    const searchParams = new URLSearchParams();
    if (params?.status) searchParams.set("status", params.status);
    if (params?.channel) searchParams.set("channel", params.channel);
    if (params?.limit) searchParams.set("limit", String(params.limit));
    if (params?.offset) searchParams.set("offset", String(params.offset));
    const qs = searchParams.toString();
    const response = await apiClient.get<{ success: boolean; data: ConversationDto[] }>(
      `/crm/conversaciones${qs ? `?${qs}` : ""}`
    );
    return (response.data ?? []).map(this.toConversationModel);
  }

  async listMessages(conversationId: number, params?: { limit?: number; offset?: number }): Promise<Message[]> {
    const searchParams = new URLSearchParams();
    if (params?.limit) searchParams.set("limit", String(params.limit));
    if (params?.offset) searchParams.set("offset", String(params.offset));
    const qs = searchParams.toString();
    const response = await apiClient.get<{ success: boolean; data: MessageDto[] }>(
      `/crm/conversaciones/${conversationId}/mensajes${qs ? `?${qs}` : ""}`
    );
    return (response.data ?? []).map(this.toMessageModel);
  }

  async updateStatus(conversationId: number, status: ConversationStatus): Promise<Conversation> {
    const response = await apiClient.patch<{ success: boolean; data: ConversationDto }>(
      `/crm/conversaciones/${conversationId}/status`,
      { status }
    );
    return this.toConversationModel(response.data);
  }

  async sendMessage(conversationId: number, content: string): Promise<Message> {
    const response = await apiClient.post<{ success: boolean; data: MessageDto }>(
      `/crm/conversaciones/${conversationId}/mensajes`,
      { content }
    );
    return this.toMessageModel(response.data);
  }

  private toConversationModel(dto: ConversationDto): Conversation {
    return {
      id: dto.id,
      organizationId: dto.organization_id,
      contactId: dto.contact_id,
      channel: (dto.channel as Channel) ?? "whatsapp",
      status: dto.status as ConversationStatus,
      lastMessageAt: dto.last_message_at,
      metadata: dto.metadata,
      createdAt: dto.created_at,
      updatedAt: dto.updated_at,
      contactPhone: dto.contact_phone,
      contactDisplayName: dto.contact_display_name,
      contactInstagramUsername: dto.contact_instagram_username,
      contactAvatarUrl: dto.contact_avatar_url,
    };
  }

  private toMessageModel(dto: MessageDto): Message {
    return {
      id: dto.id,
      organizationId: dto.organization_id,
      conversationId: dto.conversation_id,
      contactId: dto.contact_id,
      channel: (dto.channel as Channel) ?? "whatsapp",
      providerMessageId: dto.provider_message_id,
      direction: dto.direction as "inbound" | "outbound",
      messageType: dto.message_type,
      content: dto.content,
      status: dto.status,
      chatTimestamp: dto.chat_timestamp,
      createdAt: dto.created_at,
      updatedAt: dto.updated_at,
    };
  }
}

export const conversationRepository = new ConversationRepository();
