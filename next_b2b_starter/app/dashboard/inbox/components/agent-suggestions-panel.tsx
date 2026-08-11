"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import { ErrorState } from "@/components/common/error-state";
import { Loader2, Sparkles } from "lucide-react";
import { usePendingSuggestionsQuery } from "@/lib/hooks/queries/use-pending-suggestions-query";
import { useApproveSuggestion, useRejectSuggestion } from "@/lib/hooks/mutations/use-agent-suggestion-mutations";
import type { AgentSuggestion } from "@/lib/models/agent.model";
import { ui } from "@/lib/copy/ui";

interface AgentSuggestionsPanelProps {
  conversationId?: number;
}

export function AgentSuggestionsPanel({ conversationId }: AgentSuggestionsPanelProps) {
  const { data: suggestions, isLoading, isError, refetch, isRefetching } = usePendingSuggestionsQuery({
    enabled: Boolean(conversationId),
  });
  const approveMutation = useApproveSuggestion();
  const rejectMutation = useRejectSuggestion();

  const [editingId, setEditingId] = useState<number | null>(null);
  const [editedBody, setEditedBody] = useState("");

  if (isLoading) return null;

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

  const handleApprove = async (suggestion: AgentSuggestion) => {
    try {
      await approveMutation.mutateAsync({ suggestionId: suggestion.id });
    } catch {
      // toast handled by mutation
    }
  };

  const handleApproveEdited = async (suggestion: AgentSuggestion) => {
    try {
      await approveMutation.mutateAsync({ suggestionId: suggestion.id, editedBody });
      setEditingId(null);
      setEditedBody("");
    } catch {
      // toast handled by mutation
    }
  };

  const handleReject = async (suggestion: AgentSuggestion) => {
    try {
      await rejectMutation.mutateAsync(suggestion.id);
    } catch {
      // toast handled by mutation
    }
  };

  return (
    <div
      role="status"
      aria-live="polite"
      className="border-t border-gray-200 bg-violet-50/60 px-4 py-3"
    >
      <div className="mb-2 flex items-center gap-2 text-xs font-semibold uppercase tracking-wider text-violet-700">
        <Sparkles className="h-3.5 w-3.5" />
        {ui.agent.panelTitle}
      </div>
      <div className="space-y-2">
        {conversationSuggestions.map((suggestion) => (
          <div key={suggestion.id} className="rounded-lg border border-violet-200 bg-white p-3 shadow-sm">
            {suggestion.type === "escalation" ? (
              <p className="text-sm font-medium text-amber-700">
                ⚠️ {suggestion.body}
              </p>
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
                    className="bg-violet-600 text-white hover:bg-violet-700"
                    onClick={() => handleApproveEdited(suggestion)}
                    disabled={approveMutation.isPending}
                  >
                    {approveMutation.isPending && <Loader2 className="mr-1 h-3 w-3 animate-spin" />}
                    {ui.agent.sendEdited}
                  </Button>
                  <Button size="sm" variant="outline" onClick={() => { setEditingId(null); setEditedBody(""); }}>
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
                    className="bg-violet-600 text-white hover:bg-violet-700"
                    onClick={() => handleApprove(suggestion)}
                    disabled={approveMutation.isPending}
                  >
                    {approveMutation.isPending && <Loader2 className="mr-1 h-3 w-3 animate-spin" />}
                    {ui.agent.approveSend}
                  </Button>
                  <Button
                    size="sm"
                    variant="outline"
                    onClick={() => { setEditingId(suggestion.id); setEditedBody(suggestion.body); }}
                    disabled={approveMutation.isPending}
                  >
                    {ui.agent.edit}
                  </Button>
                  <Button
                    size="sm"
                    variant="ghost"
                    className="text-red-600 hover:bg-red-50"
                    onClick={() => handleReject(suggestion)}
                    disabled={rejectMutation.isPending}
                  >
                    {ui.agent.reject}
                  </Button>
                </div>
              </>
            )}
          </div>
        ))}
      </div>
    </div>
  );
}
