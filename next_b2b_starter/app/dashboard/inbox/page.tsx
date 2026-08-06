"use client";

import { useState, useEffect } from "react";
import { useRouter } from "next/navigation";
import { ConversationList } from "./components/conversation-list";
import { MessageThread } from "./components/message-thread";
import { ConversationHeader } from "./components/conversation-header";
import { ReplyInput } from "./components/reply-input";
import { EmptyState } from "./components/empty-state";
import { useConversationsQuery } from "@/lib/hooks/queries/use-conversations-query";
import { useMessagesQuery } from "@/lib/hooks/queries/use-messages-query";
import { useSendMessage } from "@/lib/hooks/mutations/use-send-message";
import { useUpdateConversationStatus } from "@/lib/hooks/mutations/use-update-conversation-status";
import { usePermissions } from "@/lib/hooks/use-permissions";
import { PERMISSIONS } from "@/lib/auth/permissions";
import type { Conversation, ConversationStatus } from "@/lib/models/conversation.model";

export default function InboxPage() {
  const router = useRouter();
  const { hasPermission, isInitialized } = usePermissions();
  const [selectedConv, setSelectedConv] = useState<Conversation | null>(null);
  const [statusFilter, setStatusFilter] = useState<string>("");

  useEffect(() => {
    if (isInitialized && !hasPermission(PERMISSIONS.ORG_MANAGE)) {
      router.replace("/dashboard");
    }
  }, [isInitialized, hasPermission, router]);

  if (!isInitialized) {
    return (
      <div className="flex h-[calc(100vh-8rem)] items-center justify-center">
        <div className="h-8 w-8 animate-spin rounded-full border-2 border-gray-200 border-t-gray-900" />
      </div>
    );
  }

  const { data: conversations, isLoading: isConvsLoading } = useConversationsQuery(
    statusFilter ? { status: statusFilter } : undefined
  );
  const { data: messages, isLoading: isMsgsLoading } = useMessagesQuery(selectedConv?.id);
  const sendMsgMutation = useSendMessage(selectedConv?.id ?? 0);
  const updateStatusMutation = useUpdateConversationStatus();

  const handleSelectConversation = (conv: Conversation) => {
    setSelectedConv(conv);
  };

  const handleSendMessage = async (content: string) => {
    if (!selectedConv) return;
    await sendMsgMutation.mutateAsync(content);
  };

  const handleToggleStatus = async () => {
    if (!selectedConv) return;
    const newStatus: ConversationStatus = selectedConv.status === "active" ? "closed" : "active";
    await updateStatusMutation.mutateAsync({
      conversationId: selectedConv.id,
      status: newStatus,
    });
    setSelectedConv({ ...selectedConv, status: newStatus });
  };

  return (
    <div className="flex h-[calc(100vh-8rem)] gap-0 overflow-hidden rounded-2xl border border-gray-200 bg-white">
      <div className="w-full max-w-sm shrink-0 border-r border-gray-200 lg:w-96">
        <ConversationList
          conversations={conversations ?? []}
          selectedId={selectedConv?.id}
          onSelect={handleSelectConversation}
          isLoading={isConvsLoading}
          statusFilter={statusFilter}
          onStatusFilterChange={setStatusFilter}
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
            <MessageThread messages={messages ?? []} isLoading={isMsgsLoading} />
            <ReplyInput
              onSend={handleSendMessage}
              isSending={sendMsgMutation.isPending}
              conversationId={selectedConv.id}
            />
          </>
        ) : (
          <EmptyState />
        )}
      </div>
    </div>
  );
}
