// lib/api/api/dto/cognitive.dto.ts

export type ChatRole = "user" | "assistant" | "system";

export interface ChatSessionDto {
  id: number;
  title: string;
  created_at: string;
  updated_at: string;
}

export interface ChatMessageDto {
  id: number;
  session_id: number;
  role: ChatRole;
  content: string;
  referenced_docs?: number[];
  tokens_used: number;
  created_at: string;
}

export interface SimilarDocumentDto {
  id: number;
  document_id: number;
  content_preview: string;
  similarity_score: number;
}

export interface ChatRequestDto {
  session_id?: number;
  message: string;
  use_rag?: boolean;
  max_documents?: number;
  context_history?: number;
}

export interface ChatResponseDto {
  session_id: number;
  message: ChatMessageDto;
  referenced_docs?: SimilarDocumentDto[];
  tokens_used: number;
}

export interface ListSessionsResponseDto {
  sessions: ChatSessionDto[];
}

export interface SessionMessagesResponseDto {
  messages: ChatMessageDto[];
}
