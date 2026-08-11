"use client";

import { useState } from "react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Send } from "lucide-react";

interface ReplyInputProps {
  onSend: (content: string) => Promise<void>;
  isSending: boolean;
  conversationId: number;
  value?: string;
  onChange?: (value: string) => void;
  onSent?: () => void;
}

export function ReplyInput({ onSend, isSending, conversationId, value, onChange, onSent }: ReplyInputProps) {
  const [internalText, setInternalText] = useState("");

  const isControlled = value !== undefined && onChange !== undefined;
  const text = isControlled ? value : internalText;

  const setText = (next: string) => {
    if (isControlled) {
      onChange?.(next);
    } else {
      setInternalText(next);
    }
  };

  const handleSend = async () => {
    if (!text.trim() || isSending) return;
    const content = text.trim();
    try {
      await onSend(content);
      setText("");
      onSent?.();
    } catch {
      toast.error("No se pudo enviar el mensaje. Tu borrador se conservó.");
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  };

  return (
    <div className="border-t border-gray-200 bg-white px-4 py-3">
      <div className="flex items-center gap-2">
        <Input
          placeholder={
            conversationId
              ? "Type a message..."
              : "Select a conversation to reply"
          }
          value={text}
          onChange={(e) => setText(e.target.value)}
          onKeyDown={handleKeyDown}
          disabled={!conversationId || isSending}
          className="flex-1 rounded-full border-gray-300 bg-gray-50 px-4 focus:bg-white"
        />
        <Button
          onClick={handleSend}
          disabled={!text.trim() || !conversationId || isSending}
          size="icon"
          className="rounded-full bg-blue-600 hover:bg-blue-700"
        >
          <Send className="h-4 w-4" />
        </Button>
      </div>
    </div>
  );
}
