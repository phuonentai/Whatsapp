// lib/models/cognitive.model.ts

export type ChatRole = "user" | "assistant" | "system";

export interface ChatSession {
  id: number;
  title: string;
  createdAt: Date;
  updatedAt: Date;
}

export interface ChatMessage {
  id: number;
  sessionId: number;
  role: ChatRole;
  content: string;
  referencedDocs?: number[];
  tokensUsed: number;
  createdAt: Date;
}

export interface SimilarDocument {
  id: number;
  documentId: number;
  contentPreview: string;
  similarityScore: number;
}

export interface ChatRequest {
  sessionId?: number;
  message: string;
  useRag?: boolean;
  maxDocuments?: number;
  contextHistory?: number;
}

export interface ChatResponse {
  sessionId: number;
  message: ChatMessage;
  referencedDocs?: SimilarDocument[];
  tokensUsed: number;
}

export const ChatHelpers = {
  getRoleConfig: (role: ChatRole) => {
    const configs: Record<ChatRole, { label: string; bgColor: string; align: "left" | "right" }> = {
      user: {
        label: "You",
        bgColor: "bg-blue-600 text-white",
        align: "right",
      },
      assistant: {
        label: "AI",
        bgColor: "bg-gray-100 text-gray-900",
        align: "left",
      },
      system: {
        label: "System",
        bgColor: "bg-amber-100 text-amber-900",
        align: "left",
      },
    };
    return configs[role] || configs.assistant;
  },

  formatTimestamp: (date: Date): string => {
    return new Intl.DateTimeFormat("en-US", {
      hour: "numeric",
      minute: "2-digit",
      hour12: true,
    }).format(date);
  },

  truncateTitle: (title: string, maxLength: number = 30): string => {
    if (title.length <= maxLength) return title;
    return `${title.substring(0, maxLength)}...`;
  },
};
