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

const statusLabels: Record<string, string> = {
  active: ui.inbox.labelActive,
  closed: ui.inbox.labelClosed,
  archived: ui.inbox.labelArchived,
};

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

function getInitials(displayName: string): string {
  const parts = displayName.trim().split(/\s+/).slice(0, 2);
  return parts.map((p) => p[0] ?? "").join("").toUpperCase() || "?";
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
      <div className="border-b border-border p-4">
        <h2 className="text-lg font-semibold text-foreground">{ui.inbox.title}</h2>
        <div className="mt-3 flex gap-1.5">
          {channelTabs.map((tab) => (
            <button
              key={tab.value}
              onClick={() => onChannelFilterChange(tab.value)}
              className={cn(
                "rounded-lg px-2.5 py-1 text-xs font-medium transition-colors",
                channelFilter === tab.value
                  ? "bg-primary text-primary-foreground"
                  : "text-muted-foreground hover:bg-slate-100"
              )}
            >
              {tab.label}
            </button>
          ))}
        </div>
        <div className="mt-1.5 flex gap-1.5">
          {statusTabs.map((tab) => (
            <button
              key={tab.value}
              onClick={() => onStatusFilterChange(tab.value)}
              className={cn(
                "rounded-lg px-2.5 py-1 text-xs font-medium transition-colors",
                statusFilter === tab.value
                  ? "bg-primary text-primary-foreground"
                  : "text-muted-foreground hover:bg-slate-100"
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
            <p className="text-sm text-muted-foreground">
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
              className="mt-2 text-xs font-medium text-primary hover:text-primary"
            >
              {ui.inbox.goToSettings}
            </a>
          </div>
        ) : (
          <div className="divide-y divide-slate-100">
            {conversations.map((conv) => {
              const displayName =
                conv.contactDisplayName ||
                conv.contactInstagramUsername ||
                conv.contactPhone ||
                ui.inbox.unknownContact;
              const subtitle =
                conv.channel === "instagram"
                  ? conv.contactInstagramUsername && conv.contactInstagramUsername !== conv.contactDisplayName
                    ? `@${conv.contactInstagramUsername}`
                    : ""
                  : conv.contactPhone && conv.contactPhone !== conv.contactDisplayName
                    ? conv.contactPhone
                    : "";
              const unread = isConversationUnread(conv, lastSeenAt ?? {});
              return (
                <button
                  key={conv.id}
                  onClick={() => onSelect(conv)}
                  className={cn(
                    "group flex w-full items-start gap-3 p-4 text-left transition-colors hover:bg-slate-50",
                    selectedId === conv.id && "bg-slate-50"
                  )}
                >
                  <div className="relative">
                    <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-slate-200 text-sm font-semibold text-muted-foreground">
                      {conv.contactAvatarUrl ? (
                        // eslint-disable-next-line @next/next/no-img-element
                        <img
                          src={conv.contactAvatarUrl}
                          alt={displayName}
                          className="h-10 w-10 rounded-full object-cover"
                        />
                      ) : (
                        getInitials(displayName)
                      )}
                    </div>
                    <span className="absolute -bottom-0.5 -right-0.5 flex h-5 w-5 items-center justify-center rounded-full border-2 border-white bg-white">
                      {conv.channel === "instagram" ? (
                        <Instagram className="h-3 w-3 text-pink-600" />
                      ) : (
                        <MessageCircle className="h-3 w-3 text-green-600" />
                      )}
                    </span>
                  </div>
                  <div className="min-w-0 flex-1">
                    <div className="flex items-center justify-between gap-2">
                      <span
                        className={cn(
                          "truncate text-sm font-medium text-foreground",
                          unread && "font-semibold"
                        )}
                      >
                        {displayName}
                      </span>
                      <span className="flex items-center gap-1.5 shrink-0">
                        {pendingCounts && pendingCounts[conv.id] > 0 && (
                          <span className="inline-flex h-5 min-w-5 items-center justify-center rounded-full bg-emerald-500 px-1.5 text-[10px] font-semibold text-white">
                            {pendingCounts[conv.id]}
                          </span>
                        )}
                        {unread && (
                          <span
                            aria-label={ui.inbox.unreadAria}
                            className="inline-block h-2.5 w-2.5 shrink-0 rounded-full bg-primary"
                          />
                        )}
                        <span className="text-xs text-muted-foreground">
                          {timeAgo(conv.lastMessageAt)}
                        </span>
                      </span>
                    </div>
                    <div className="mt-0.5 flex items-center gap-2">
                      {subtitle && (
                        <span className="truncate text-xs text-muted-foreground">{subtitle}</span>
                      )}
                      {conv.status !== "active" && (
                        <span
                          className={cn(
                            "shrink-0 rounded-full px-2 py-0.5 text-[10px] font-medium",
                            conv.status === "closed" && "bg-slate-100 text-muted-foreground",
                            conv.status === "archived" && "bg-orange-50 text-orange-600"
                          )}
                        >
                          {statusLabels[conv.status] ?? conv.status}
                        </span>
                      )}
                    </div>
                  </div>
                </button>
              );
            })}
          </div>
        )}
      </div>
    </div>
  );
}
