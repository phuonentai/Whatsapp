"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import { ErrorState } from "@/components/common/error-state";
import { Skeleton } from "@/components/ui/skeleton";
import { ChevronDown, Loader2, Sparkles, AlertTriangle } from "lucide-react";
import { usePendingSuggestionsQuery } from "@/lib/hooks/queries/use-pending-suggestions-query";
import { useMessagesQuery } from "@/lib/hooks/queries/use-messages-query";
import { useRejectSuggestion } from "@/lib/hooks/mutations/use-agent-suggestion-mutations";
import { useAiUsageQuery } from "@/lib/hooks/queries/use-ai-usage-query";
import type { AgentSuggestion } from "@/lib/models/agent.model";
import { cn } from "@/lib/utils";
import { ui } from "@/lib/copy/ui";

interface AgentSuggestionsPanelProps {
  conversationId?: number;
  /**
   * Approve NEVER sends silently: the suggestion body is prefilled into the
   * composer for explicit human review/send (copilot philosophy).
   */
  onApproveAsDraft?: (body: string) => void;
}

export function AgentSuggestionsPanel({ conversationId, onApproveAsDraft }: AgentSuggestionsPanelProps) {
  const { data: suggestions, isLoading, isError, refetch, isRefetching } = usePendingSuggestionsQuery({
    enabled: Boolean(conversationId),
  });
  const { data: messages, isLoading: isMessagesLoading } = useMessagesQuery(conversationId);
  const rejectMutation = useRejectSuggestion();
  const { data: aiUsage } = useAiUsageQuery();
  const creditsExhausted =
    (aiUsage?.credits_max ?? 0) > 0 && (aiUsage?.credits_remaining ?? 0) <= 0;

  const [editingId, setEditingId] = useState<number | null>(null);
  const [editedBody, setEditedBody] = useState("");
  const [pendingIds, setPendingIds] = useState<Set<number>>(new Set());
  const [contextOpen, setContextOpen] = useState(false);

  if (!conversationId) return null;

  if (isLoading) {
    return (
      <div
        data-testid="suggestions-skeleton"
        aria-busy="true"
        className="border-t border-gray-200 bg-violet-50/60 px-4 py-3"
      >
        <div className="mb-2 flex items-center gap-2 text-xs font-semibold uppercase tracking-wider text-violet-700">
          <Sparkles className="h-3.5 w-3.5" />
          {ui.agent.panelTitle}
        </div>
        <div className="space-y-2">
          <Skeleton className="h-14 w-full rounded-lg" />
          <Skeleton className="h-14 w-full rounded-lg" />
        </div>
      </div>
    );
  }

  if (isError) {
    return (
      <div className="border-t border-gray-200 bg-red-50/40 px-4 py-3">
        <ErrorState
          title={ui.agent.panelErrorTitle}
          description={ui.agent.panelErrorDesc}
          onRetry={() => refetch()}
          isRetrying={isRefetching}
        />
      </div>
    );
  }

  const conversationSuggestions = (suggestions ?? []).filter(
    (s) => s.conversation_id === conversationId && s.status === "pending"
  );

  if (conversationSuggestions.length === 0) return null;

  const handleReject = (suggestion: AgentSuggestion) => {
    setPendingIds((prev) => new Set(prev).add(suggestion.id));
    rejectMutation.mutateAsync(suggestion.id).finally(() => {
      setPendingIds((prev) => {
        const next = new Set(prev);
        next.delete(suggestion.id);
        return next;
      });
    });
  };

  const handleUseDraft = (suggestion: AgentSuggestion, body?: string) => {
    setEditingId(null);
    setEditedBody("");
    onApproveAsDraft?.(body ?? suggestion.body);
  };

  return (
    <div
      role="status"
      aria-live="polite"
      className="border-t border-gray-200 bg-violet-50/60 px-4 py-3"
    >
      <div className="mb-2 flex items-center justify-between gap-2">
        <div className="flex items-center gap-2 text-xs font-semibold uppercase tracking-wider text-violet-700">
          <Sparkles className="h-3.5 w-3.5" />
          {ui.agent.panelTitle}
        </div>
        <button
          type="button"
          onClick={() => setContextOpen((v) => !v)}
          aria-expanded={contextOpen}
          className="flex items-center gap-1 rounded-md px-2 py-1 text-xs font-medium text-violet-700 transition-colors hover:bg-violet-100"
        >
          {contextOpen ? ui.inbox.hideContext : ui.inbox.showContext}
          <ChevronDown
            className={cn("h-3.5 w-3.5 transition-transform duration-200", contextOpen && "rotate-180")}
          />
        </button>
      </div>

      {creditsExhausted && (
        <p
          data-testid="suggestions-credits-notice"
          className="mb-2 flex items-center gap-1.5 rounded-lg border border-amber-200 bg-amber-50 px-3 py-2 text-xs text-amber-800"
        >
          <AlertTriangle className="h-3.5 w-3.5 flex-shrink-0" />
          {ui.inbox.creditsExhaustedPanelNotice}
        </p>
      )}

      {contextOpen && (
        <div
          data-testid="suggestion-context"
          className="mb-3 max-h-44 overflow-y-auto rounded-lg border border-violet-100 bg-white/80 p-3"
        >
          <p className="mb-2 text-[11px] font-semibold uppercase tracking-wider text-gray-500">
            {ui.inbox.contextThreadLabel}
          </p>
          {isMessagesLoading ? (
            <p className="text-xs text-gray-500">{ui.inbox.contextLoading}</p>
          ) : (messages ?? []).length === 0 ? (
            <p className="text-xs text-gray-400">{ui.inbox.threadEmpty}</p>
          ) : (
            <ul className="space-y-1.5">
              {(messages ?? []).slice(-5).map((msg) => (
                <li key={msg.id} className="flex gap-2 text-xs">
                  <span
                    className={cn(
                      "flex-shrink-0 font-medium",
                      msg.direction === "inbound" ? "text-violet-700" : "text-gray-500"
                    )}
                  >
                    {msg.direction === "inbound" ? ui.inbox.contextInbound : ui.inbox.contextOutbound}
                  </span>
                  <span className="line-clamp-2 text-gray-700">
                    {msg.content || msg.messageType}
                  </span>
                </li>
              ))}
            </ul>
          )}
        </div>
      )}

      <div className="space-y-2">
        {conversationSuggestions.map((suggestion) => {
          const isPending = pendingIds.has(suggestion.id);
          const isEscalation = suggestion.type === "escalation";
          return (
            <div
              key={suggestion.id}
              className={cn(
                "rounded-lg border bg-white p-3 shadow-sm",
                isEscalation ? "border-amber-300" : "border-violet-200"
              )}
            >
              <div className="mb-1.5 flex items-center gap-1.5">
                <span
                  className={cn(
                    "inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-[10px] font-semibold",
                    isEscalation
                      ? "bg-amber-100 text-amber-800"
                      : "bg-violet-100 text-violet-700"
                  )}
                >
                  ✦ {isEscalation ? ui.inbox.escalationHumanNote : ui.inbox.aiDraftMarker}
                </span>
                {isEscalation && (
                  <span className="text-[10px] font-medium text-amber-700">
                    {ui.inbox.escalationHumanNote}
                  </span>
                )}
              </div>

              {isEscalation ? (
                <>
                  <p className="text-sm font-medium text-amber-800 whitespace-pre-wrap">
                    {suggestion.body}
                  </p>
                  <div className="mt-2 flex gap-2">
                    <Button
                      size="sm"
                      variant="outline"
                      className="border-amber-300 text-amber-800 hover:bg-amber-50"
                      onClick={() => handleUseDraft(suggestion)}
                      title={ui.inbox.approveAsDraftHint}
                    >
                      {ui.inbox.approveAsDraft}
                    </Button>
                    <Button
                      size="sm"
                      variant="ghost"
                      className="text-red-600 hover:bg-red-50"
                      onClick={() => handleReject(suggestion)}
                      disabled={isPending}
                    >
                      {isPending && <Loader2 className="mr-1 h-3 w-3 animate-spin" />}
                      {ui.agent.reject}
                    </Button>
                  </div>
                </>
              ) : editingId === suggestion.id ? (
                <div className="space-y-2">
                  <textarea
                    className="min-h-20 w-full rounded-md border border-gray-300 px-3 py-2 text-sm"
                    value={editedBody}
                    onChange={(e) => setEditedBody(e.target.value)}
                  />
                  <div className="flex gap-2">
                    <Button
                      size="sm"
                      className="bg-violet-700 text-white hover:bg-violet-800"
                      onClick={() => handleUseDraft(suggestion, editedBody)}
                      disabled={!editedBody.trim()}
                    >
                      {ui.inbox.useDraft}
                    </Button>
                    <Button
                      size="sm"
                      variant="outline"
                      onClick={() => { setEditingId(null); setEditedBody(""); }}
                    >
                      {ui.common.cancel}
                    </Button>
                  </div>
                </div>
              ) : (
                <>
                  <p className="text-sm text-gray-800 whitespace-pre-wrap">{suggestion.body}</p>
                  <div className="mt-2 flex gap-2">
                    <Button
                      size="sm"
                      className="bg-violet-700 text-white hover:bg-violet-800"
                      onClick={() => handleUseDraft(suggestion)}
                      title={ui.inbox.approveAsDraftHint}
                    >
                      {ui.inbox.approveAsDraft}
                    </Button>
                    <Button
                      size="sm"
                      variant="outline"
                      onClick={() => { setEditingId(suggestion.id); setEditedBody(suggestion.body); }}
                    >
                      {ui.agent.edit}
                    </Button>
                    <Button
                      size="sm"
                      variant="ghost"
                      className="text-red-600 hover:bg-red-50"
                      onClick={() => handleReject(suggestion)}
                      disabled={isPending}
                    >
                      {isPending && <Loader2 className="mr-1 h-3 w-3 animate-spin" />}
                      {ui.agent.reject}
                    </Button>
                  </div>
                </>
              )}
            </div>
          );
        })}
      </div>
    </div>
  );
}
