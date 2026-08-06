"use client";

import type { Message } from "@/lib/models/conversation.model";
import { Skeleton } from "@/components/ui/skeleton";
import { cn } from "@/lib/utils";
import { useEffect, useRef } from "react";

interface MessageThreadProps {
  messages: Message[];
  isLoading: boolean;
}

function formatTime(dateStr?: string): string {
  if (!dateStr) return "";
  const date = new Date(dateStr);
  return date.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
}

function messageTypeLabel(type: string): string {
  switch (type) {
    case "image": return "🖼 Image";
    case "video": return "🎬 Video";
    case "audio": return "🎵 Audio";
    case "document": return "📄 Document";
    case "location": return "📍 Location";
    default: return type;
  }
}

export function MessageThread({ messages, isLoading }: MessageThreadProps) {
  const bottomRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages]);

  return (
    <div className="flex-1 overflow-y-auto bg-gray-50 px-4 py-6">
      {isLoading ? (
        <div className="space-y-4">
          {Array.from({ length: 4 }).map((_, i) => (
            <div key={i} className={cn("flex", i % 2 === 0 ? "justify-start" : "justify-end")}>
              <Skeleton className={cn("h-12 rounded-2xl", i % 2 === 0 ? "w-3/4" : "w-1/2")} />
            </div>
          ))}
        </div>
      ) : messages.length === 0 ? (
        <div className="flex h-full items-center justify-center">
          <p className="text-sm text-gray-400">No messages yet. Send a reply to start the conversation.</p>
        </div>
      ) : (
        <div className="space-y-3">
          {messages.map((msg) => (
            <div
              key={msg.id}
              className={cn(
                "flex",
                msg.direction === "inbound" ? "justify-start" : "justify-end"
              )}
            >
              <div
                className={cn(
                  "max-w-[75%] rounded-2xl px-4 py-2.5",
                  msg.direction === "inbound"
                    ? "bg-white text-gray-900 shadow-sm"
                    : "bg-blue-600 text-white"
                )}
              >
                {msg.messageType === "text" ? (
                  <p className="text-sm whitespace-pre-wrap">{msg.content}</p>
                ) : (
                  <div className="space-y-1">
                    <p className="text-sm font-medium">{messageTypeLabel(msg.messageType)}</p>
                    {msg.content && <p className="text-xs opacity-80">{msg.content}</p>}
                  </div>
                )}
                <p
                  className={cn(
                    "mt-1 text-[10px]",
                    msg.direction === "inbound" ? "text-gray-400" : "text-blue-200"
                  )}
                >
                  {formatTime(msg.chatTimestamp || msg.createdAt)}
                  {msg.direction === "outbound" && msg.status === "sent" && " · ✓"}
                  {msg.direction === "outbound" && msg.status === "delivered" && " · ✓✓"}
                  {msg.direction === "outbound" && msg.status === "read" && " · ✓✓"}
                </p>
              </div>
            </div>
          ))}
          <div ref={bottomRef} />
        </div>
      )}
    </div>
  );
}
