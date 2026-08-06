"use client";

import { Sparkles } from "lucide-react";
import type {
  ChatMessage as ChatMessageType,
  SimilarDocument,
} from "@/lib/models/cognitive.model";
import { ChatHelpers } from "@/lib/models/cognitive.model";
import { DocumentSources } from "./document-sources";

interface ChatMessageProps {
  message: ChatMessageType;
  sources?: SimilarDocument[];
}

export function ChatMessage({ message, sources }: ChatMessageProps) {
  const isUser = message.role === "user";

  if (isUser) {
    return (
      <div className="flex justify-end">
        <div
          className="max-w-[70%] rounded-lg rounded-br-sm px-3 py-2"
          style={{ backgroundColor: "#5b21b6", color: "white" }}
        >
          <p className="text-sm whitespace-pre-wrap">{message.content}</p>
          <p className="text-xs mt-1" style={{ color: "rgba(255,255,255,0.7)" }}>
            {ChatHelpers.formatTimestamp(message.createdAt)}
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="flex gap-2">
      <div
        className="h-7 w-7 rounded-full flex items-center justify-center flex-shrink-0"
        style={{ backgroundColor: "#ede9fe" }}
      >
        <Sparkles className="h-3.5 w-3.5" style={{ color: "#7c3aed" }} />
      </div>
      <div className="max-w-[70%]">
        <div
          className="rounded-lg rounded-tl-sm px-3 py-2"
          style={{ backgroundColor: "#f3f4f6" }}
        >
          <p className="text-sm whitespace-pre-wrap" style={{ color: "#111827" }}>{message.content}</p>
          <p className="text-xs mt-1" style={{ color: "#6b7280" }}>
            {ChatHelpers.formatTimestamp(message.createdAt)}
          </p>
        </div>
        {sources && sources.length > 0 && <DocumentSources sources={sources} />}
      </div>
    </div>
  );
}

export function TypingIndicator() {
  return (
    <div className="flex gap-2">
      <div
        className="h-7 w-7 rounded-full flex items-center justify-center flex-shrink-0"
        style={{ backgroundColor: "#ede9fe" }}
      >
        <Sparkles className="h-3.5 w-3.5" style={{ color: "#7c3aed" }} />
      </div>
      <div className="rounded-lg rounded-tl-sm px-3 py-2" style={{ backgroundColor: "#f3f4f6" }}>
        <div className="flex gap-1">
          <span className="h-2 w-2 rounded-full animate-bounce" style={{ backgroundColor: "#9ca3af", animationDelay: "0ms" }} />
          <span className="h-2 w-2 rounded-full animate-bounce" style={{ backgroundColor: "#9ca3af", animationDelay: "150ms" }} />
          <span className="h-2 w-2 rounded-full animate-bounce" style={{ backgroundColor: "#9ca3af", animationDelay: "300ms" }} />
        </div>
      </div>
    </div>
  );
}
