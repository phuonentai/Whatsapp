export interface ConversationDto {
  id: number;
  organization_id: number;
  contact_id: number;
  channel: string;
  status: string;
  last_message_at?: string;
  metadata?: Record<string, unknown>;
  created_at: string;
  updated_at: string;
  contact_phone: string;
  contact_display_name: string;
  contact_instagram_username?: string;
  contact_avatar_url?: string;
  assignee_stytch_member_id?: string | null;
}

export interface MessageDto {
  id: number;
  organization_id: number;
  conversation_id: number;
  contact_id: number;
  channel: string;
  provider_message_id?: string;
  direction: string;
  message_type: string;
  content?: string;
  status: string;
  chat_timestamp?: string;
  created_at: string;
  updated_at: string;
}
