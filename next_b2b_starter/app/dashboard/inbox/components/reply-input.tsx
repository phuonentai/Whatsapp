"use client";

import { useState } from "react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Loader2, Send, Sparkles, AlertTriangle } from "lucide-react";
import { ui } from "@/lib/copy/ui";
import {
  isAiCreditsExhausted,
  useRephraseMutation,
} from "@/lib/hooks/mutations/use-rephrase-mutation";
import { useAiUsageQuery } from "@/lib/hooks/queries/use-ai-usage-query";
import type { RephraseMode } from "@/lib/models/agent.model";

interface ReplyInputProps {
  onSend: (content: string) => Promise<void>;
  isSending: boolean;
  conversationId: number;
  value?: string;
  onChange?: (value: string) => void;
  onSent?: () => void;
  /** Writing assist (rephrase) is admin-only in the member tier. */
  showWritingAssist?: boolean;
}

const REPHRASE_ITEMS: Array<{ mode: RephraseMode; label: string }> = [
  { mode: "rephrase", label: ui.agent.rephrase },
  { mode: "formal", label: ui.agent.rephraseFormal },
  { mode: "casual", label: ui.agent.rephraseCasual },
  { mode: "summarize", label: ui.agent.rephraseSummarize },
];

export function ReplyInput({ onSend, isSending, conversationId, value, onChange, onSent, showWritingAssist = true }: ReplyInputProps) {
  const [internalText, setInternalText] = useState("");
  const rephraseMutation = useRephraseMutation();
  const isRephrasing = rephraseMutation.isPending;
  const { data: aiUsage } = useAiUsageQuery();
  // 402 only-IA: credits exhausted disables the AI transform menu (tooltip),
  // never the manual composer.
  const creditsExhausted =
    (aiUsage?.credits_max ?? 0) > 0 && (aiUsage?.credits_remaining ?? 0) <= 0;

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

  // Writing assist: the returned text replaces the draft for review.
  // Never auto-sends; on failure the draft is kept untouched.
  const handleRephrase = async (mode: RephraseMode) => {
    if (!text.trim() || isRephrasing || isSending) return;
    try {
      const result = await rephraseMutation.mutateAsync({ text, mode });
      setText(result.text);
    } catch (error) {
      if (isAiCreditsExhausted(error)) {
        toast.error(ui.agent.rephraseCreditsExhausted);
      } else {
        toast.error(ui.agent.rephraseError);
      }
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  };

  const assistDisabled =
    !text.trim() || !conversationId || isSending || isRephrasing || creditsExhausted;

  return (
    <div className="border-t border-gray-200 bg-white px-4 py-3">
      <div className="flex items-center gap-2">
        <Input
          placeholder={
            conversationId
              ? "Escribe un mensaje..."
              : "Selecciona una conversación para responder"
          }
          value={text}
          onChange={(e) => setText(e.target.value)}
          onKeyDown={handleKeyDown}
          disabled={!conversationId || isSending}
          className="flex-1 rounded-full border-gray-300 bg-gray-50 px-4 focus:bg-white"
        />
        {showWritingAssist && (
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button
                type="button"
                variant="ghost"
                size="icon"
                aria-label={ui.agent.rephraseTrigger}
                disabled={assistDisabled}
                title={creditsExhausted ? ui.inbox.creditsExhaustedTooltip : undefined}
                data-credits-exhausted={creditsExhausted || undefined}
                className="rounded-full text-gray-500 hover:text-blue-600"
              >
                {isRephrasing ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : (
                  <Sparkles className="h-4 w-4" />
                )}
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              {creditsExhausted ? (
                <DropdownMenuItem disabled className="flex items-center gap-2 text-amber-700">
                  <AlertTriangle className="h-3.5 w-3.5" />
                  {ui.inbox.creditsExhaustedTooltip}
                </DropdownMenuItem>
              ) : (
                REPHRASE_ITEMS.map((item) => (
                  <DropdownMenuItem
                    key={item.mode}
                    disabled={isRephrasing}
                    onClick={() => handleRephrase(item.mode)}
                  >
                    {item.label}
                  </DropdownMenuItem>
                ))
              )}
            </DropdownMenuContent>
          </DropdownMenu>
        )}
        <Button
          onClick={handleSend}
          aria-label="Enviar"
          disabled={!text.trim() || !conversationId || isSending}
          size="icon"
          className="rounded-full bg-emerald-500 hover:bg-emerald-600"
        >
          <Send className="h-4 w-4" />
        </Button>
      </div>
    </div>
  );
}
