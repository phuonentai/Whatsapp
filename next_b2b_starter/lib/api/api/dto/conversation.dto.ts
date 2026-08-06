export interface ConversationDto {
  id: number;
  organization_id: number;
  contact_id: number;
  status: string;
  last_message_at?: string;
  metadata?: Record<string, unknown>;
  created_at: string;
  updated_at: string;
  contact_phone: string;
  contact_display_name: string;
}

export interface MessageDto {
  id: number;
  organization_id: number;
  conversation_id: number;
  contact_id: number;
  whatsapp_message_id?: string;
  direction: string;
  message_type: string;
  content?: string;
  status: string;
  chat_timestamp?: string;
  created_at: string;
  updated_at: string;
}
