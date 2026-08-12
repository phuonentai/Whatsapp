"use client";

import { useState, useEffect, useCallback, useRef } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import Link from "next/link";
import { toast } from "sonner";
import { ConversationList } from "./components/conversation-list";
import { InboxMetrics } from "./components/inbox-metrics";
import { MessageThread } from "./components/message-thread";
import { ConversationHeader } from "./components/conversation-header";
import { ConversationContextPanel } from "@/components/agent/conversation-context-panel";
import { ReplyInput } from "./components/reply-input";
import { QuickReplies } from "./components/quick-replies";
import { EmptyState } from "./components/empty-state";
import { AgentSuggestionsPanel } from "./components/agent-suggestions-panel";
import { useConversationsQuery } from "@/lib/hooks/queries/use-conversations-query";
import { useMessagesQuery } from "@/lib/hooks/queries/use-messages-query";
import { useSendMessage } from "@/lib/hooks/mutations/use-send-message";
import { useUpdateConversationStatus } from "@/lib/hooks/mutations/use-update-conversation-status";
import { useWhatsAppConfigQuery } from "@/lib/hooks/queries/use-whatsapp-config-query";
import { useInstagramConfigQuery } from "@/lib/hooks/queries/use-instagram-config-query";
import { isConversationUnread } from "@/lib/inbox/unread";
import { usePermissions } from "@/lib/hooks/use-permissions";
import { PERMISSIONS } from "@/lib/auth/permissions";
import { useFeature } from "@/lib/hooks/use-entitlement";
import { usePendingSuggestionsQuery } from "@/lib/hooks/queries/use-pending-suggestions-query";
import { useSequence } from "@/lib/hooks/use-sequence";
import { useInboxStore } from "@/lib/stores/use-inbox-store";
import { markInboxVisited } from "@/lib/onboarding/storage";
import { ui, tpl } from "@/lib/copy/ui";
import type { Conversation, ConversationStatus, Channel, ConversationScopeView } from "@/lib/models/conversation.model";
import type { PlaybookGuionDto } from "@/lib/api/api/dto/playbook.dto";

const CHANNEL_VALUES: Array<Channel | "all"> = ["all", "whatsapp", "instagram"];
const SCOPE_VALUES: ConversationScopeView[] = ["mine", "queue", "all"];

export default function InboxPage() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const { hasPermission, isInitialized } = usePermissions();
  const [selectedConv, setSelectedConv] = useState<Conversation | null>(null);
  const [statusFilter, setStatusFilter] = useState<string>("");
  const [draft, setDraft] = useState<string>("");
  const lastSeenAt = useInboxStore((s) => s.lastSeenAt);
  const markSeen = useInboxStore((s) => s.markSeen);

  // Inbox tier: members with inbox:view can read + reply manually; admin
  // controls (close/reopen, quick replies, suggestions, writing assist) are
  // hidden for them (and rejected server-side with 403).
  const canManage = hasPermission(PERMISSIONS.ORG_MANAGE);

  // conversation-row-scoping: la capa de scope solo se activa en planes pagos
  // (flag de entitlement); en free tier la bandeja se comporta org-scope.
  const rowScopingEnabled = useFeature("conversation_row_scoping");
  const canViewAll =
    hasPermission(PERMISSIONS.INBOX_VIEW_ALL) || hasPermission(PERMISSIONS.ORG_MANAGE);
  const canViewUnassigned = hasPermission(PERMISSIONS.INBOX_VIEW_UNASSIGNED);
  const canReassign =
    hasPermission(PERMISSIONS.INBOX_REASSIGN) || hasPermission(PERMISSIONS.ORG_MANAGE);

  const rawChannel = searchParams.get("channel");
  const channelFilter: Channel | "all" = CHANNEL_VALUES.includes(rawChannel as Channel)
    ? (rawChannel as Channel)
    : "all";

  const rawScope = searchParams.get("scope");
  const scopeView: ConversationScopeView = (SCOPE_VALUES as string[]).includes(rawScope ?? "")
    ? (rawScope as ConversationScopeView)
    : "";

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

  // conversation-row-scoping: las vistas de scope (Mis chats / Cola / Todos)
  // se derivan del MISMO predicado del backend (param `scope`), nunca de
  // filtrado client-side que difiera. Free tier (flag off) → sin capa de scope.
  const handleScopeChange = useCallback(
    (scope: ConversationScopeView) => {
      const next = new URLSearchParams(searchParams.toString());
      if (!scope) {
        next.delete("scope");
      } else {
        next.set("scope", scope);
      }
      const qs = next.toString();
      router.replace(qs ? `/dashboard/inbox?${qs}` : "/dashboard/inbox");
    },
    [router, searchParams]
  );

  useEffect(() => {
    if (
      isInitialized &&
      !hasPermission(PERMISSIONS.ORG_MANAGE) &&
      !hasPermission(PERMISSIONS.INBOX_VIEW)
    ) {
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
    // Paridad unread: el badge y el poll 5s consultan con el mismo scope que
    // la lista (sin contadores fantasma de conversaciones invisibles).
    ...(rowScopingEnabled && scopeView ? { scope: scopeView } : {}),
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
  const { data: whatsappConfig } = useWhatsAppConfigQuery();
  const { data: instagramConfig } = useInstagramConfigQuery();

  // Channel-disconnected banner over the composer (per selected channel; for
  // "all", the selected conversation's channel decides).
  const bannerChannel: Channel | null =
    channelFilter !== "all"
      ? channelFilter
      : selectedConv
        ? selectedConv.channel
        : null;
  const channelDisconnected =
    bannerChannel === "whatsapp"
      ? whatsappConfig?.isActive !== true
      : bannerChannel === "instagram"
        ? instagramConfig?.isActive !== true
        : false;

  // Unread live-region: announce ONLY increments detected by the 5s poll,
  // never the full state (no a11y spam).
  const unreadCount = (conversations ?? []).filter((c) =>
    isConversationUnread(c, lastSeenAt ?? {})
  ).length;
  const [announcement, setAnnouncement] = useState<string | null>(null);
  const prevUnreadRef = useRef<number | null>(null);
  useEffect(() => {
    if (prevUnreadRef.current === null) {
      prevUnreadRef.current = unreadCount;
      return;
    }
    const increment = unreadCount - prevUnreadRef.current;
    prevUnreadRef.current = unreadCount;
    if (increment > 0) {
      const label = tpl(ui.inbox.newMessagesLiveRegion, { n: increment });
      setAnnouncement(label);
      const timer = window.setTimeout(() => setAnnouncement(null), 4000);
      return () => window.clearTimeout(timer);
    }
  }, [unreadCount]);

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
    <div className="space-y-6">
      {/* Header mensajes-view (plantilla Verifika) */}
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="font-heading text-2xl font-bold tracking-tight text-foreground sm:text-3xl">
            {ui.inbox.headerTitle}
          </h1>
          <p className="mt-1 text-slate-600">{ui.inbox.headerSubtitle}</p>
        </div>
        <div className="flex items-center gap-3">
          <span className="inline-flex items-center gap-2 rounded-lg border border-slate-200 bg-white px-4 py-2 text-sm font-medium text-slate-700">
            <span className="h-2 w-2 rounded-full bg-emerald-500" aria-hidden="true" />
            {ui.inbox.connected}
          </span>
          <Link
            href="/dashboard/crm?view=campa%C3%B1as"
            className="inline-flex items-center gap-2 rounded-lg bg-emerald-500 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-emerald-600"
          >
            <span className="text-lg leading-none" aria-hidden="true">+</span>
            {ui.inbox.newCampaign}
          </Link>
        </div>
      </div>

      {/* messages-view composition: 4 metric cards over the real inbox data */}
      <InboxMetrics
        conversations={conversations ?? []}
        lastSeenAt={lastSeenAt}
        pendingCounts={pendingCounts}
      />

      {/* conversation-row-scoping: tabs de visibilidad (Mis chats / Cola /
          Todos), visibles solo en planes pagos y según permisos de scope;
          free tier (flag off) no muestra la capa de scope. La vista "Todos"
          requiere view_all u org:manage; "Cola sin asignar" requiere
          view_unassigned. */}
      {rowScopingEnabled && (canViewAll || canViewUnassigned) && (
        <div
          className="flex items-center gap-1 border-b border-slate-200"
          role="tablist"
          aria-label={ui.inbox.scopeTabsAria}
        >
          <button
            type="button"
            role="tab"
            aria-selected={scopeView === "mine" || scopeView === ""}
            onClick={() => handleScopeChange("mine")}
            className={`rounded-t-lg px-4 py-2 text-sm font-medium transition-colors ${
              scopeView === "mine" || scopeView === ""
                ? "border-b-2 border-emerald-500 text-emerald-600"
                : "text-slate-500 hover:text-slate-800"
            }`}
          >
            {ui.inbox.scopeMine}
          </button>
          {canViewUnassigned && (
            <button
              type="button"
              role="tab"
              aria-selected={scopeView === "queue"}
              onClick={() => handleScopeChange("queue")}
              className={`rounded-t-lg px-4 py-2 text-sm font-medium transition-colors ${
                scopeView === "queue"
                  ? "border-b-2 border-emerald-500 text-emerald-600"
                  : "text-slate-500 hover:text-slate-800"
              }`}
            >
              {ui.inbox.scopeQueue}
            </button>
          )}
          {canViewAll && (
            <button
              type="button"
              role="tab"
              aria-selected={scopeView === "all"}
              onClick={() => handleScopeChange("all")}
              className={`rounded-t-lg px-4 py-2 text-sm font-medium transition-colors ${
                scopeView === "all"
                  ? "border-b-2 border-emerald-500 text-emerald-600"
                  : "text-slate-500 hover:text-slate-800"
              }`}
            >
              {ui.inbox.scopeAll}
            </button>
          )}
        </div>
      )}

      {/* Unread live-region: discreet announcements only on increments. */}
      <div aria-live="polite" className="sr-only">
        {announcement}
      </div>

      <div className="flex h-[calc(100vh-15rem)] gap-0 overflow-hidden rounded-2xl border border-slate-200 bg-white">
        <div className="w-full max-w-sm shrink-0 border-r border-slate-200 lg:w-96">
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
                canManage={canManage}
                canReassign={canReassign}
                rowScopingEnabled={rowScopingEnabled}
              />
              <ConversationContextPanel conversationId={selectedConv.id} contactId={selectedConv.contactId} />
              <MessageThread
                messages={messages ?? []}
                isLoading={isMsgsLoading}
                isError={isMsgsError}
                onRetry={() => refetchMsgs()}
                isRetrying={isMsgsRefetching}
              />
              {canManage && (
                <AgentSuggestionsPanel
                  conversationId={selectedConv.id}
                  onApproveAsDraft={(body) => {
                    setDraft(body);
                  }}
                />
              )}
              {channelDisconnected && bannerChannel && (
                <div
                  role="status"
                  className="border-t border-red-200 bg-red-50 px-4 py-2 text-xs font-medium text-red-700"
                >
                  {bannerChannel === "whatsapp"
                    ? ui.inbox.channelDisconnectedBanner
                    : ui.inbox.channelDisconnectedBannerIG}
                </div>
              )}
              {canManage && (
                <QuickReplies
                  conversationId={selectedConv.id}
                  onSelect={handleSelectGuion}
                  sequenceActive={sequence.active}
                  sequenceStep={sequence.stepIndex}
                  sequenceTotal={sequence.totalSteps}
                />
              )}
              <ReplyInput
                onSend={handleSendMessage}
                isSending={sendMsgMutation.isPending}
                conversationId={selectedConv.id}
                value={draft}
                onChange={setDraft}
                onSent={handleSequenceSent}
                showWritingAssist={canManage}
              />
            </>
          ) : (
            <EmptyState channel={channelFilter} />
          )}
        </div>
      </div>
    </div>
  );
}
