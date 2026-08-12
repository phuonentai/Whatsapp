"use client";

import { useState } from "react";
import { Brain, Save, ShieldAlert, Sparkles } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { cn } from "@/lib/utils";
import { ui } from "@/lib/copy/ui";
import { useConversationContextQuery } from "@/lib/hooks/queries/use-conversation-context-query";
import { useCreateActivity } from "@/lib/hooks/mutations/use-crm-mutations";
import type { ConversationContext } from "@/lib/models/agent.model";

interface ConversationContextPanelProps {
  conversationId: number;
  /** Contact to attach the note to; the save action is hidden when absent. */
  contactId?: number;
  className?: string;
}

function formatDate(value?: string): string {
  if (!value) return "—";
  return new Date(value).toLocaleString([], {
    day: "2-digit",
    month: "short",
    hour: "2-digit",
    minute: "2-digit",
  });
}

function channelLabel(channel?: string): string {
  if (channel === "instagram") return "Instagram";
  return "WhatsApp";
}

/** Plain-text note body: summary + intent + key facts, omitting missing sections. */
function buildNoteContent(context: ConversationContext): string {
  const lines: string[] = [];
  if (context.summary) lines.push(`${ui.agent.contextSummaryLabel}: ${context.summary}`);
  if (context.detectedIntent) lines.push(`${ui.agent.contextIntentLabel}: ${context.detectedIntent}`);
  if (context.keyFacts.length > 0) {
    lines.push(`${ui.agent.contextFactsLabel}: ${context.keyFacts.join("; ")}`);
  }
  return lines.join("\n");
}

/**
 * AI-derived context for a conversation: summary, detected intent, and key
 * facts. Renders the "assistant is learning" state while loading or when the
 * backend reports context as unavailable, and a consent notice with
 * structural data only when analysis is consent-gated (Ley 1581).
 */
export function ConversationContextPanel({
  conversationId,
  contactId,
  className,
}: ConversationContextPanelProps) {
  const { data, isLoading, isError } = useConversationContextQuery(conversationId);
  const createActivity = useCreateActivity();
  const [saved, setSaved] = useState(false);

  const handleSaveNote = () => {
    if (contactId === undefined || !data) return;
    createActivity.mutate(
      {
        contact_id: contactId,
        tipo: "nota",
        asunto: ui.agent.noteSubject,
        contenido: buildNoteContent(data),
      },
      {
        onSuccess: () => {
          setSaved(true);
          toast.success(ui.agent.noteSaved);
        },
        onError: () => {
          toast.error(ui.agent.noteError);
        },
      }
    );
  };

  if (isLoading) {
    return (
      <div
        className={cn(
          "flex items-center gap-3 border-b border-gray-200 bg-gray-50 px-4 py-3",
          className
        )}
      >
        <Skeleton className="h-4 w-4 rounded-full" />
        <div className="flex-1 space-y-1.5">
          <Skeleton className="h-3 w-2/5" />
          <Skeleton className="h-3 w-4/5" />
        </div>
      </div>
    );
  }

  if (isError || !data || data.status === "unavailable") {
    return (
      <div className={cn("border-b border-gray-200 bg-gray-50 px-4 py-3", className)}>
        <div className="flex items-start gap-2">
          <Sparkles className="mt-0.5 h-4 w-4 shrink-0 text-indigo-500" />
          <div>
            <p className="text-sm font-medium text-gray-700">{ui.agent.contextLearningTitle}</p>
            <p className="text-xs text-gray-500">{ui.agent.contextLearningBody}</p>
          </div>
        </div>
      </div>
    );
  }

  if (data.consentGated || data.status === "structural") {
    return (
      <div className={cn("border-b border-gray-200 bg-gray-50 px-4 py-3", className)}>
        <div className="flex items-start gap-2">
          <ShieldAlert className="mt-0.5 h-4 w-4 shrink-0 text-amber-500" />
          <div className="flex-1">
            <p className="text-sm font-medium text-gray-700">{ui.agent.contextConsentTitle}</p>
            <p className="text-xs text-gray-500">{ui.agent.contextConsentBody}</p>
            {typeof data.messageCount === "number" && (
              <dl className="mt-2 grid grid-cols-2 gap-x-4 gap-y-1 text-xs text-gray-600 sm:grid-cols-4">
                <div>
                  <dt className="text-gray-400">{ui.agent.contextStructuralMessages}</dt>
                  <dd>{data.messageCount}</dd>
                </div>
                <div>
                  <dt className="text-gray-400">{ui.agent.contextStructuralChannel}</dt>
                  <dd>{channelLabel(data.channel)}</dd>
                </div>
                <div>
                  <dt className="text-gray-400">{ui.agent.contextStructuralFirst}</dt>
                  <dd>{formatDate(data.firstMessageAt)}</dd>
                </div>
                <div>
                  <dt className="text-gray-400">{ui.agent.contextStructuralLast}</dt>
                  <dd>{formatDate(data.lastMessageAt)}</dd>
                </div>
              </dl>
            )}
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className={cn("border-b border-gray-200 bg-gray-50 px-4 py-3", className)}>
      <div className="flex items-start gap-2">
        <Brain className="mt-0.5 h-4 w-4 shrink-0 text-indigo-500" />
        <div className="flex-1 space-y-2">
          <p className="text-xs font-semibold uppercase tracking-wide text-gray-400">
            {ui.agent.contextPanelTitle}
          </p>
          {data.summary && (
            <p className="text-sm text-gray-700">
              <span className="font-medium">{ui.agent.contextSummaryLabel}: </span>
              {data.summary}
            </p>
          )}
          {data.detectedIntent && (
            <span className="inline-flex items-center rounded-full bg-indigo-50 px-2 py-0.5 text-xs font-medium text-indigo-700">
              {ui.agent.contextIntentLabel}: {data.detectedIntent}
            </span>
          )}
          {data.keyFacts.length > 0 && (
            <ul className="list-inside list-disc space-y-0.5 text-sm text-gray-700">
              {data.keyFacts.map((fact) => (
                <li key={fact}>{fact}</li>
              ))}
            </ul>
          )}
          {contactId !== undefined && (
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={handleSaveNote}
              disabled={saved || createActivity.isPending}
            >
              <Save />
              {ui.agent.saveNote}
            </Button>
          )}
        </div>
      </div>
    </div>
  );
}
