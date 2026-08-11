"use client";

import { useState, useEffect, useCallback } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { toast } from "sonner";
import { ConversationList } from "./components/conversation-list";
import { MessageThread } from "./components/message-thread";
import { ConversationHeader } from "./components/conversation-header";
import { ReplyInput } from "./components/reply-input";
import { QuickReplies } from "./components/quick-replies";
import { EmptyState } from "./components/empty-state";
import { AgentSuggestionsPanel } from "./components/agent-suggestions-panel";
import { useConversationsQuery } from "@/lib/hooks/queries/use-conversations-query";
import { useMessagesQuery } from "@/lib/hooks/queries/use-messages-query";
import { useSendMessage } from "@/lib/hooks/mutations/use-send-message";
import { useUpdateConversationStatus } from "@/lib/hooks/mutations/use-update-conversation-status";
import { usePermissions } from "@/lib/hooks/use-permissions";
import { PERMISSIONS } from "@/lib/auth/permissions";
import { usePendingSuggestionsQuery } from "@/lib/hooks/queries/use-pending-suggestions-query";
import { useSequence } from "@/lib/hooks/use-sequence";
import { useInboxStore } from "@/lib/stores/use-inbox-store";
import { markInboxVisited } from "@/lib/onboarding/storage";
import type { Conversation, ConversationStatus, Channel } from "@/lib/models/conversation.model";
import type { PlaybookGuionDto } from "@/lib/api/api/dto/playbook.dto";

const CHANNEL_VALUES: Array<Channel | "all"> = ["all", "whatsapp", "instagram"];

export default function InboxPage() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const { hasPermission, isInitialized } = usePermissions();
  const [selectedConv, setSelectedConv] = useState<Conversation | null>(null);
  const [statusFilter, setStatusFilter] = useState<string>("");
  const [draft, setDraft] = useState<string>("");
  const lastSeenAt = useInboxStore((s) => s.lastSeenAt);
  const markSeen = useInboxStore((s) => s.markSeen);

  const rawChannel = searchParams.get("channel");
  const channelFilter: Channel | "all" = CHANNEL_VALUES.includes(rawChannel as Channel)
    ? (rawChannel as Channel)
    : "all";

  const handleChannelChange = useCallback(
    (channel: Channel | "all") => {
      const next = new URLSearchParams(searchParams.toString());
      if (channel === "all") {
        next.delete("channel");
      } else {
        next.set("channel", channel);
      }
      const qs = next.toString();
      router.replace(qs ? `/dashboard/inbox?${qs}` : "/dashboard/inbox");
    },
    [router, searchParams]
  );

  useEffect(() => {
    if (isInitialized && !hasPermission(PERMISSIONS.ORG_MANAGE)) {
      router.replace("/dashboard");
    }
  }, [isInitialized, hasPermission, router]);

  useEffect(() => {
    markInboxVisited();
  }, []);

  const {
    data: conversations,
    isLoading: isConvsLoading,
    isError: isConvsError,
    refetch: refetchConvs,
    isRefetching: isConvsRefetching,
  } = useConversationsQuery({
    ...(statusFilter ? { status: statusFilter } : {}),
    ...(channelFilter !== "all" ? { channel: channelFilter } : {}),
  });
  const {
    data: messages,
    isLoading: isMsgsLoading,
    isError: isMsgsError,
    refetch: refetchMsgs,
    isRefetching: isMsgsRefetching,
  } = useMessagesQuery(selectedConv?.id);
  const { data: pendingSuggestions } = usePendingSuggestionsQuery();
  const sendMsgMutation = useSendMessage(selectedConv?.id ?? 0);
  const updateStatusMutation = useUpdateConversationStatus();
  const sequence = useSequence(selectedConv?.id ?? 0);

  const pendingCounts = (pendingSuggestions ?? []).reduce<Record<number, number>>((acc, s) => {
    if (s.status !== "pending") return acc;
    acc[s.conversation_id] = (acc[s.conversation_id] ?? 0) + 1;
    return acc;
  }, {});

  if (!isInitialized) {
    return (
      <div className="flex h-[calc(100vh-8rem)] items-center justify-center">
        <div className="h-8 w-8 animate-spin rounded-full border-2 border-gray-200 border-t-gray-900" />
      </div>
    );
  }

  const handleSelectConversation = (conv: Conversation) => {
    setSelectedConv(conv);
    sequence.reset();
    setDraft("");
    markSeen(conv.id);
  };

  const handleSendMessage = async (content: string) => {
    if (!selectedConv) return;
    await sendMsgMutation.mutateAsync(content);
    markSeen(selectedConv.id);
  };

  const handleSelectGuion = (guion: PlaybookGuionDto) => {
    const firstStep = sequence.start(guion);
    setDraft(firstStep ?? guion.mensaje ?? "");
  };

  const handleSequenceSent = () => {
    const nextStep = sequence.advance();
    if (nextStep !== null) {
      setDraft(nextStep);
    }
  };

  const handleToggleStatus = async () => {
    if (!selectedConv) return;
    const newStatus: ConversationStatus = selectedConv.status === "active" ? "closed" : "active";
    try {
      await updateStatusMutation.mutateAsync({
        conversationId: selectedConv.id,
        status: newStatus,
      });
      setSelectedConv({ ...selectedConv, status: newStatus });
    } catch {
      toast.error("No se pudo actualizar el estado de la conversación. Inténtalo de nuevo.");
    }
  };

  return (
    <div className="flex h-[calc(100vh-8rem)] gap-0 overflow-hidden rounded-2xl border border-gray-200 bg-white">
      <div className="w-full max-w-sm shrink-0 border-r border-gray-200 lg:w-96">
        <ConversationList
          conversations={conversations ?? []}
          selectedId={selectedConv?.id}
          onSelect={handleSelectConversation}
          isLoading={isConvsLoading}
          isError={isConvsError}
          onRetry={() => refetchConvs()}
          isRetrying={isConvsRefetching}
          statusFilter={statusFilter}
          onStatusFilterChange={setStatusFilter}
          channelFilter={channelFilter}
          onChannelFilterChange={handleChannelChange}
          pendingCounts={pendingCounts}
          lastSeenAt={lastSeenAt}
        />
      </div>

      <div className="flex flex-1 flex-col overflow-hidden">
        {selectedConv ? (
          <>
            <ConversationHeader
              conversation={selectedConv}
              onToggleStatus={handleToggleStatus}
              isUpdating={updateStatusMutation.isPending}
            />
            <MessageThread
              messages={messages ?? []}
              isLoading={isMsgsLoading}
              isError={isMsgsError}
              onRetry={() => refetchMsgs()}
              isRetrying={isMsgsRefetching}
            />
            <AgentSuggestionsPanel conversationId={selectedConv.id} />
            <QuickReplies
              conversationId={selectedConv.id}
              onSelect={handleSelectGuion}
              sequenceActive={sequence.active}
              sequenceStep={sequence.stepIndex}
              sequenceTotal={sequence.totalSteps}
            />
            <ReplyInput
              onSend={handleSendMessage}
              isSending={sendMsgMutation.isPending}
              conversationId={selectedConv.id}
              value={draft}
              onChange={setDraft}
              onSent={handleSequenceSent}
            />
          </>
        ) : (
          <EmptyState channel={channelFilter} />
        )}
      </div>
    </div>
  );
}
