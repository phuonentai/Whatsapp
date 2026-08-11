"use client";

import type { Conversation, Channel } from "@/lib/models/conversation.model";
import { Skeleton } from "@/components/ui/skeleton";
import { ErrorState } from "@/components/common/error-state";
import { isConversationUnread } from "@/lib/inbox/unread";
import { cn } from "@/lib/utils";
import { Instagram, MessageCircle } from "lucide-react";
import { ui, tpl } from "@/lib/copy/ui";

interface ConversationListProps {
  conversations: Conversation[];
  selectedId?: number;
  onSelect: (conv: Conversation) => void;
  isLoading: boolean;
  isError?: boolean;
  onRetry?: () => void;
  isRetrying?: boolean;
  statusFilter: string;
  onStatusFilterChange: (filter: string) => void;
  channelFilter: Channel | "all";
  onChannelFilterChange: (channel: Channel | "all") => void;
  pendingCounts?: Record<number, number>;
  lastSeenAt?: Record<number, number>;
}

const statusTabs = [
  { label: ui.inbox.tabAll, value: "" },
  { label: ui.inbox.tabActive, value: "active" },
  { label: ui.inbox.tabClosed, value: "closed" },
  { label: ui.inbox.tabArchived, value: "archived" },
];

const channelTabs: Array<{ label: string; value: Channel | "all" }> = [
  { label: ui.inbox.channelAll, value: "all" },
  { label: ui.inbox.channelWhatsapp, value: "whatsapp" },
  { label: ui.inbox.channelInstagram, value: "instagram" },
];

function timeAgo(dateStr?: string): string {
  if (!dateStr) return "";
  const date = new Date(dateStr);
  const diff = Date.now() - date.getTime();
  const mins = Math.floor(diff / 60000);
  if (mins < 1) return ui.inbox.timeJustNow;
  if (mins < 60) return tpl(ui.inbox.timeMin, { n: mins });
  const hrs = Math.floor(mins / 60);
  if (hrs < 24) return tpl(ui.inbox.timeHour, { n: hrs });
  const days = Math.floor(hrs / 24);
  return tpl(ui.inbox.timeDay, { n: days });
}

export function ConversationList({
  conversations,
  selectedId,
  onSelect,
  isLoading,
  isError = false,
  onRetry,
  isRetrying = false,
  statusFilter,
  onStatusFilterChange,
  channelFilter,
  onChannelFilterChange,
  pendingCounts,
  lastSeenAt,
}: ConversationListProps) {
  return (
    <div className="flex h-full flex-col">
      <div className="border-b border-gray-200 px-4 py-3">
        <h2 className="text-lg font-semibold text-gray-900">{ui.inbox.title}</h2>
        <div className="mt-2 flex gap-1">
          {channelTabs.map((tab) => (
            <button
              key={tab.value}
              onClick={() => onChannelFilterChange(tab.value)}
              className={cn(
                "rounded-md px-2.5 py-1 text-xs font-medium transition-colors",
                channelFilter === tab.value
                  ? "bg-blue-600 text-white"
                  : "text-gray-600 hover:bg-gray-100"
              )}
            >
              {tab.label}
            </button>
          ))}
        </div>
        <div className="mt-2 flex gap-1">
          {statusTabs.map((tab) => (
            <button
              key={tab.value}
              onClick={() => onStatusFilterChange(tab.value)}
              className={cn(
                "rounded-md px-2.5 py-1 text-xs font-medium transition-colors",
                statusFilter === tab.value
                  ? "bg-gray-900 text-white"
                  : "text-gray-600 hover:bg-gray-100"
              )}
            >
              {tab.label}
            </button>
          ))}
        </div>
      </div>

      <div className="flex-1 overflow-y-auto">
        {isError ? (
          <div className="p-4">
            <ErrorState
              title={ui.inbox.listErrorTitle}
              description={ui.inbox.listErrorDesc}
              onRetry={onRetry}
              isRetrying={isRetrying}
            />
          </div>
        ) : isLoading ? (
          <div className="space-y-2 p-4">
            {Array.from({ length: 5 }).map((_, i) => (
              <div key={i} className="flex items-center gap-3 rounded-lg p-3">
                <Skeleton className="h-10 w-10 rounded-full" />
                <div className="flex-1 space-y-2">
                  <Skeleton className="h-4 w-32" />
                  <Skeleton className="h-3 w-48" />
                </div>
              </div>
            ))}
          </div>
        ) : conversations.length === 0 ? (
          <div className="flex flex-col items-center justify-center px-4 py-12 text-center">
            <p className="text-sm text-gray-500">
              {channelFilter === "instagram"
                ? ui.inbox.emptyInstagram
                : channelFilter === "whatsapp"
                  ? ui.inbox.emptyWhatsapp
                  : ui.inbox.emptyAll}
            </p>
            <a
              href={
                channelFilter === "instagram"
                  ? "/dashboard/settings?view=instagram"
                  : "/dashboard/settings?view=whatsapp"
              }
              className="mt-2 text-xs font-medium text-blue-600 hover:text-blue-700"
            >
              {ui.inbox.goToSettings}
            </a>
          </div>
        ) : (
          <div className="divide-y divide-gray-100">
            {conversations.map((conv) => (
              <button
                key={conv.id}
                onClick={() => onSelect(conv)}
                className={cn(
                  "flex w-full items-start gap-3 px-4 py-3 text-left transition-colors hover:bg-gray-50",
                  selectedId === conv.id && "bg-gray-100"
                )}
              >
                <div className="relative">
                  <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-gray-200 text-sm font-medium text-gray-600">
                    {conv.contactAvatarUrl ? (
                      // eslint-disable-next-line @next/next/no-img-element
                      <img
                        src={conv.contactAvatarUrl}
                        alt={conv.contactDisplayName || conv.contactInstagramUsername || ""}
                        className="h-10 w-10 rounded-full object-cover"
                      />
                    ) : (
                      (conv.contactDisplayName || conv.contactInstagramUsername || conv.contactPhone || "?").charAt(0)
                    )}
                  </div>
                  <span className="absolute -bottom-1 -right-1 flex h-5 w-5 items-center justify-center rounded-full border border-white bg-white">
                    {conv.channel === "instagram" ? (
                      <Instagram className="h-3.5 w-3.5 text-pink-600" />
                    ) : (
                      <MessageCircle className="h-3.5 w-3.5 text-green-600" />
                    )}
                  </span>
                </div>
                <div className="min-w-0 flex-1">
                  <div className="flex items-center justify-between gap-2">
                    <span className="truncate text-sm font-medium text-gray-900">
                      {conv.contactDisplayName || conv.contactInstagramUsername || conv.contactPhone || ui.inbox.unknownContact}
                    </span>
                    <span className="flex items-center gap-1.5 shrink-0">
                      {pendingCounts && pendingCounts[conv.id] > 0 && (
                        <span className="inline-flex h-5 min-w-5 items-center justify-center rounded-full bg-violet-600 px-1.5 text-[10px] font-semibold text-white">
                          {pendingCounts[conv.id]}
                        </span>
                      )}
                      {isConversationUnread(conv, lastSeenAt ?? {}) && (
                        <span
                          aria-label={ui.inbox.unreadAria}
                          className="inline-block h-2.5 w-2.5 shrink-0 rounded-full bg-blue-500"
                        />
                      )}
                      <span className="text-xs text-gray-400">
                        {timeAgo(conv.lastMessageAt)}
                      </span>
                    </span>
                  </div>
                  <div className="flex items-center gap-2 mt-0.5">
                    <span className="truncate text-xs text-gray-500">
                      {conv.channel === "instagram"
                        ? conv.contactInstagramUsername && conv.contactInstagramUsername !== conv.contactDisplayName
                          ? `@${conv.contactInstagramUsername}`
                          : ""
                        : conv.contactPhone && conv.contactPhone !== conv.contactDisplayName
                          ? conv.contactPhone
                          : ""}
                    </span>
                    {conv.status !== "active" && (
                      <span className={cn(
                        "shrink-0 rounded-full px-1.5 py-0.5 text-[10px] font-medium uppercase",
                        conv.status === "closed" && "bg-gray-100 text-gray-600",
                        conv.status === "archived" && "bg-orange-50 text-orange-600",
                      )}>
                        {conv.status}
                      </span>
                    )}
                  </div>
                </div>
              </button>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
