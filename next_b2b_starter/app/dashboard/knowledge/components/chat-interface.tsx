"use client";

import { useState, useRef, useEffect } from "react";
import { Send, Sparkles, Plus } from "lucide-react";
import { Skeleton } from "@/components/ui/skeleton";
import { ChatMessage, TypingIndicator } from "./chat-message";
import type {
  ChatMessage as ChatMessageType,
  SimilarDocument,
} from "@/lib/models/cognitive.model";
import { ChatHelpers } from "@/lib/models/cognitive.model";

interface ChatInterfaceProps {
  messages: ChatMessageType[];
  sessionTitle?: string;
  isLoading?: boolean;
  isSending?: boolean;
  onSendMessage: (message: string, useRag: boolean) => Promise<void>;
  onNewChat: () => void;
  messageSources?: Record<number, SimilarDocument[]>;
  documentCount?: number;
}

function EmptyState({ onSuggestionClick }: { onSuggestionClick?: (text: string) => void }) {
  const suggestions = [
    "Summarize documents",
    "Find key dates",
    "Main topics"
  ];

  return (
    <div className="flex-1 flex flex-col items-center justify-center p-6">
      <div
        className="h-12 w-12 rounded-full flex items-center justify-center"
        style={{ backgroundColor: "#ede9fe" }}
      >
        <Sparkles className="h-6 w-6" style={{ color: "#7c3aed" }} />
      </div>
      <h3 className="mt-4 text-base font-medium" style={{ color: "#111827" }}>
        How can I help?
      </h3>
      <p className="mt-1 text-sm" style={{ color: "#6b7280" }}>
        Ask about your documents
      </p>
      <div className="mt-4 flex flex-wrap gap-2 justify-center">
        {suggestions.map((text, i) => (
          <button
            key={i}
            onClick={() => onSuggestionClick?.(text)}
            className="px-3 py-1.5 text-xs rounded-full border hover:bg-gray-50"
            style={{ borderColor: "#e5e7eb", color: "#4b5563" }}
          >
            {text}
          </button>
        ))}
      </div>
    </div>
  );
}

function LoadingSkeleton() {
  return (
    <div className="flex-1 p-4 space-y-4">
      <div className="flex justify-end">
        <Skeleton className="h-10 w-48 rounded-lg" />
      </div>
      <div className="flex gap-2">
        <Skeleton className="h-8 w-8 rounded-full flex-shrink-0" />
        <Skeleton className="h-16 w-64 rounded-lg" />
      </div>
    </div>
  );
}

export function ChatInterface({
  messages,
  sessionTitle,
  isLoading = false,
  isSending = false,
  onSendMessage,
  onNewChat,
  messageSources,
}: ChatInterfaceProps) {
  const [input, setInput] = useState("");
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages]);

  const handleSend = async () => {
    if (!input.trim() || isSending) return;
    const message = input.trim();
    setInput("");
    await onSendMessage(message, true);
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  };

  const title = sessionTitle ? ChatHelpers.truncateTitle(sessionTitle) : "New Chat";

  return (
    <div className="h-full flex flex-col min-h-0">
      {/* Header */}
      <div className="h-14 px-4 border-b border-gray-200 flex items-center justify-between flex-shrink-0 bg-white">
        <div className="flex items-center gap-3">
          <div
            className="h-8 w-8 rounded-full flex items-center justify-center"
            style={{ backgroundColor: "#ede9fe" }}
          >
            <Sparkles className="h-4 w-4" style={{ color: "#7c3aed" }} />
          </div>
          <div>
            <p className="text-sm font-medium" style={{ color: "#111827" }}>{title}</p>
            <p className="text-xs" style={{ color: "#16a34a" }}>Online</p>
          </div>
        </div>
        <button
          onClick={onNewChat}
          className="flex items-center gap-1.5 px-3 py-1.5 rounded-md text-sm hover:bg-gray-100"
          style={{ color: "#374151" }}
        >
          <Plus className="h-4 w-4" />
          New
        </button>
      </div>

      {/* Messages */}
      {isLoading ? (
        <LoadingSkeleton />
      ) : messages.length === 0 ? (
        <EmptyState onSuggestionClick={(text) => setInput(text)} />
      ) : (
        <div className="flex-1 min-h-0 overflow-y-auto p-4 space-y-3 bg-white">
          {messages.map((message) => (
            <ChatMessage
              key={message.id}
              message={message}
              sources={messageSources?.[message.id]}
            />
          ))}
          {isSending && <TypingIndicator />}
          <div ref={messagesEndRef} />
        </div>
      )}

      {/* Input */}
      <div className="p-4 border-t border-gray-200 flex-shrink-0 bg-white">
        <div className="flex gap-2 items-center">
          <textarea
            ref={textareaRef}
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder="Type a message..."
            disabled={isSending}
            rows={1}
            className="flex-1 resize-none rounded-lg border border-gray-200 px-3 py-2.5 text-sm focus:outline-none focus:ring-2 focus:ring-violet-500 focus:border-transparent disabled:opacity-50"
          />
          <button
            onClick={handleSend}
            disabled={!input.trim() || isSending}
            className="h-10 w-10 rounded-lg flex-shrink-0 flex items-center justify-center disabled:opacity-50"
            style={{
              backgroundColor: input.trim() ? "#5b21b6" : "#f3f4f6",
              color: input.trim() ? "white" : "#9ca3af",
            }}
          >
            <Send className="h-4 w-4" />
          </button>
        </div>
      </div>
    </div>
  );
}
