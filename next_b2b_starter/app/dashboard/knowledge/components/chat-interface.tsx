"use client";

import { useState, useRef, useEffect } from "react";
import { Send, Sparkles, Plus, Coins, Upload, AlertTriangle } from "lucide-react";
import Link from "next/link";
import { Skeleton } from "@/components/ui/skeleton";
import { ChatMessage, TypingIndicator } from "./chat-message";
import type {
  ChatMessage as ChatMessageType,
  SimilarDocument,
} from "@/lib/models/cognitive.model";
import { ChatHelpers } from "@/lib/models/cognitive.model";
import { tpl, ui } from "@/lib/copy/ui";
import { cn } from "@/lib/utils";

interface ChatInterfaceProps {
  messages: ChatMessageType[];
  sessionTitle?: string;
  isLoading?: boolean;
  isSending?: boolean;
  streamingMessageId?: number | null;
  onSendMessage: (message: string, useRag: boolean) => Promise<void>;
  onNewChat: () => void;
  messageSources?: Record<number, SimilarDocument[]>;
  onSourceClick?: (documentId: number) => void;
  credits?: { used: number; max: number; remaining: number };
  indexingCount?: number;
  canManage?: boolean;
  onAddDocument?: () => void;
}

function EmptyState({ onSuggestionClick }: { onSuggestionClick?: (text: string) => void }) {
  const suggestions = ["Resumen de documentos", "Fechas clave", "Temas principales"];

  return (
    <div className="flex-1 flex flex-col items-center justify-center p-6">
      <div
        className="h-12 w-12 rounded-full flex items-center justify-center"
        style={{ backgroundColor: "#ede9fe" }}
      >
        <Sparkles className="h-6 w-6" style={{ color: "#7c3aed" }} />
      </div>
      <h3 className="mt-4 text-base font-medium" style={{ color: "#111827" }}>
        ¿En qué te ayudo?
      </h3>
      <p className="mt-1 text-sm" style={{ color: "#6b7280" }}>
        Pregunta sobre tus documentos
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
  streamingMessageId = null,
  onSendMessage,
  onNewChat,
  messageSources,
  onSourceClick,
  credits,
  indexingCount = 0,
  canManage = true,
  onAddDocument,
}: ChatInterfaceProps) {
  const [input, setInput] = useState("");
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages]);

  const creditsExhausted = (credits?.max ?? 0) > 0 && (credits?.remaining ?? 0) <= 0;
  const creditsAmber =
    (credits?.max ?? 0) > 0 && (credits?.used ?? 0) / credits!.max >= 0.8;

  const handleSend = async () => {
    if (!input.trim() || isSending || creditsExhausted) return;
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

  const title = sessionTitle ? ChatHelpers.truncateTitle(sessionTitle) : ui.knowledge.newChat;

  return (
    <div className="h-full flex flex-col min-h-0">
      {/* Header */}
      <div className="h-14 px-4 border-b border-gray-200 flex items-center justify-between flex-shrink-0 bg-white">
        <div className="flex items-center gap-3 min-w-0">
          <div
            className="h-8 w-8 rounded-full flex items-center justify-center flex-shrink-0"
            style={{ backgroundColor: "#ede9fe" }}
          >
            <Sparkles className="h-4 w-4" style={{ color: "#7c3aed" }} />
          </div>
          <div className="min-w-0">
            <p className="text-sm font-medium truncate" style={{ color: "#111827" }}>{title}</p>
            <p className="text-xs" style={{ color: "#16a34a" }}>En línea</p>
          </div>
        </div>

        <div className="flex items-center gap-2 flex-shrink-0">
          {indexingCount > 0 && (
            <span
              className="hidden sm:inline-flex items-center gap-1.5 rounded-full px-2.5 py-1 text-[11px] font-medium"
              style={{ backgroundColor: "#fffbeb", color: "#b45309" }}
            >
              <span className="h-1.5 w-1.5 rounded-full animate-pulse" style={{ backgroundColor: "#f59e0b" }} />
              {tpl(ui.knowledge.indexingCount, { n: indexingCount })}
            </span>
          )}

          {credits && credits.max > 0 && (
            <span
              className={cn(
                "inline-flex items-center gap-1.5 rounded-full px-2.5 py-1 text-[11px] font-medium",
                creditsAmber ? "bg-amber-100 text-amber-800" : "bg-gray-100 text-gray-600"
              )}
              title={tpl(ui.knowledge.creditsUsed, { used: credits.used, max: credits.max })}
            >
              <Coins className="h-3.5 w-3.5" />
              {tpl(ui.knowledge.creditsUsed, { used: credits.used, max: credits.max })}
            </span>
          )}

          {canManage && onAddDocument && (
            <button
              onClick={onAddDocument}
              className="flex items-center gap-1.5 px-3 py-1.5 rounded-md text-sm hover:bg-gray-100"
              style={{ color: "#374151" }}
            >
              <Upload className="h-4 w-4" />
              <span className="hidden md:inline">{ui.knowledge.addDocument}</span>
            </button>
          )}

          <button
            onClick={onNewChat}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-md text-sm hover:bg-gray-100"
            style={{ color: "#374151" }}
            aria-label={ui.knowledge.newChat}
          >
            <Plus className="h-4 w-4" />
          </button>
        </div>
      </div>

      {/* Messages */}
      {isLoading ? (
        <LoadingSkeleton />
      ) : messages.length === 0 ? (
        <EmptyState onSuggestionClick={(text) => setInput(text)} />
      ) : (
        <div
          aria-live="polite"
          className="flex-1 min-h-0 overflow-y-auto p-4 space-y-3 bg-white"
        >
          {messages.map((message) => (
            <ChatMessage
              key={message.id}
              message={message}
              sources={messageSources?.[message.id]}
              streaming={streamingMessageId === message.id}
              onSourceClick={onSourceClick}
            />
          ))}
          {isSending && streamingMessageId === null && <TypingIndicator />}
          <div ref={messagesEndRef} />
        </div>
      )}

      {/* Composer */}
      <div className="p-4 border-t border-gray-200 flex-shrink-0 bg-white">
        {creditsExhausted && (
          <div
            role="status"
            className="mb-2 flex items-center gap-2 rounded-lg border border-amber-200 px-3 py-2 text-xs"
            style={{ backgroundColor: "#fffbeb", color: "#b45309" }}
          >
            <AlertTriangle className="h-3.5 w-3.5 flex-shrink-0" />
            <span>{ui.knowledge.creditsExhaustedBanner}</span>
            <Link
              href="/dashboard/settings?view=subscription"
              className="ml-auto font-medium underline underline-offset-2"
            >
              {ui.knowledge.upgradeCta}
            </Link>
          </div>
        )}
        <div className="flex gap-2 items-center">
          <textarea
            ref={textareaRef}
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder={creditsExhausted ? ui.knowledge.creditsExhaustedBanner : ui.knowledge.newChat}
            disabled={isSending || creditsExhausted}
            rows={1}
            aria-label={ui.knowledge.newChat}
            className="flex-1 resize-none rounded-lg border border-gray-200 px-3 py-2.5 text-sm focus:outline-none focus:ring-2 focus:ring-violet-500 focus:border-transparent disabled:opacity-50"
          />
          <button
            onClick={handleSend}
            disabled={!input.trim() || isSending || creditsExhausted}
            aria-label="Enviar"
            className="h-10 w-10 rounded-lg flex-shrink-0 flex items-center justify-center disabled:opacity-50"
            style={{
              backgroundColor: input.trim() && !creditsExhausted ? "#5b21b6" : "#f3f4f6",
              color: input.trim() && !creditsExhausted ? "white" : "#9ca3af",
            }}
          >
            <Send className="h-4 w-4" />
          </button>
        </div>
      </div>
    </div>
  );
}
