// lib/api/api/repositories/cognitive-repository.ts

import { apiClient } from "../client/api-client";
import {
  ChatSessionDto,
  ChatMessageDto,
  ChatRequestDto,
  ChatResponseDto,
  ListSessionsResponseDto,
  SessionMessagesResponseDto,
  SimilarDocumentDto,
} from "../dto/cognitive.dto";
import {
  ChatSession,
  ChatMessage,
  ChatRequest,
  ChatResponse,
  SimilarDocument,
  ChatRole,
} from "@/lib/models/cognitive.model";

class CognitiveRepository {
  /**
   * Send a chat message
   */
  async chat(request: ChatRequest): Promise<ChatResponse> {
    const payload: ChatRequestDto = {
      session_id: request.sessionId,
      message: request.message,
      use_rag: request.useRag,
      max_documents: request.maxDocuments,
      context_history: request.contextHistory,
    };

    type ChatApiResponse =
      | ChatResponseDto
      | {
          data?: ChatResponseDto;
        };

    const response = await apiClient.post<ChatApiResponse>(
      "/example_cognitive/chat",
      payload
    );

    const dto: ChatResponseDto =
      "data" in response && response.data
        ? response.data
        : (response as ChatResponseDto);

    return {
      sessionId: dto.session_id,
      message: this.toMessage(dto.message),
      referencedDocs: dto.referenced_docs?.map((doc) => this.toSimilarDocument(doc)),
      tokensUsed: dto.tokens_used,
    };
  }

  /**
   * List chat sessions
   */
  async listSessions(): Promise<ChatSession[]> {
    type ListSessionsApiResponse =
      | ListSessionsResponseDto
      | {
          data?: ListSessionsResponseDto;
          sessions?: ChatSessionDto[];
        };

    const response = await apiClient.get<ListSessionsApiResponse>(
      "/example_cognitive/sessions"
    );

    const sessions: ChatSessionDto[] =
      "data" in response && response.data
        ? response.data.sessions
        : (response as { sessions?: ChatSessionDto[] }).sessions ??
          (response as ListSessionsResponseDto).sessions ??
          [];

    return sessions.map((dto) => this.toSession(dto));
  }

  /**
   * Get messages for a session
   */
  async getSessionMessages(sessionId: number): Promise<ChatMessage[]> {
    const response = await apiClient.get<ChatMessageDto[] | SessionMessagesResponseDto>(
      `/example_cognitive/sessions/${sessionId}/messages`
    );

    // Handle direct array response (actual API format)
    if (Array.isArray(response)) {
      return response.map((dto) => this.toMessage(dto));
    }

    // Handle wrapped response (fallback for backward compatibility)
    const messages: ChatMessageDto[] = response.messages ?? [];
    return messages.map((dto) => this.toMessage(dto));
  }

  /**
   * Transform DTO to ChatSession model
   */
  private toSession(dto: ChatSessionDto): ChatSession {
    return {
      id: dto.id,
      title: dto.title,
      createdAt: new Date(dto.created_at),
      updatedAt: new Date(dto.updated_at),
    };
  }

  /**
   * Transform DTO to ChatMessage model
   */
  private toMessage(dto: ChatMessageDto): ChatMessage {
    return {
      id: dto.id,
      sessionId: dto.session_id,
      role: dto.role as ChatRole,
      content: dto.content,
      referencedDocs: dto.referenced_docs,
      tokensUsed: dto.tokens_used,
      createdAt: new Date(dto.created_at),
    };
  }

  /**
   * Transform DTO to SimilarDocument model
   */
  private toSimilarDocument(dto: SimilarDocumentDto): SimilarDocument {
    return {
      id: dto.id,
      documentId: dto.document_id,
      contentPreview: dto.content_preview,
      similarityScore: dto.similarity_score,
    };
  }
}

export const cognitiveRepository = new CognitiveRepository();
