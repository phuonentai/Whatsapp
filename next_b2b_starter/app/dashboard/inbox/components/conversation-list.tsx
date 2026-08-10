"use client";

import type { Conversation, Channel } from "@/lib/models/conversation.model";
import { Skeleton } from "@/components/ui/skeleton";
import { cn } from "@/lib/utils";
import { Instagram, MessageCircle } from "lucide-react";

interface ConversationListProps {
  conversations: Conversation[];
  selectedId?: number;
  onSelect: (conv: Conversation) => void;
  isLoading: boolean;
  statusFilter: string;
  onStatusFilterChange: (filter: string) => void;
  channelFilter: Channel | "all";
  onChannelFilterChange: (channel: Channel | "all") => void;
  pendingCounts?: Record<number, number>;
}

const statusTabs = [
  { label: "All", value: "" },
  { label: "Active", value: "active" },
  { label: "Closed", value: "closed" },
  { label: "Archived", value: "archived" },
];

const channelTabs: Array<{ label: string; value: Channel | "all" }> = [
  { label: "All", value: "all" },
  { label: "WhatsApp", value: "whatsapp" },
  { label: "Instagram", value: "instagram" },
];

function timeAgo(dateStr?: string): string {
  if (!dateStr) return "";
  const date = new Date(dateStr);
  const diff = Date.now() - date.getTime();
  const mins = Math.floor(diff / 60000);
  if (mins < 1) return "just now";
  if (mins < 60) return `${mins}m ago`;
  const hrs = Math.floor(mins / 60);
  if (hrs < 24) return `${hrs}h ago`;
  const days = Math.floor(hrs / 24);
  return `${days}d ago`;
}

export function ConversationList({
  conversations,
  selectedId,
  onSelect,
  isLoading,
  statusFilter,
  onStatusFilterChange,
  channelFilter,
  onChannelFilterChange,
  pendingCounts,
}: ConversationListProps) {
  return (
    <div className="flex h-full flex-col">
      <div className="border-b border-gray-200 px-4 py-3">
        <h2 className="text-lg font-semibold text-gray-900">Inbox</h2>
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
        {isLoading ? (
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
                ? "No Instagram messages yet — connect Instagram in Settings to get started"
                : channelFilter === "whatsapp"
                  ? "No WhatsApp messages yet — connect WhatsApp in Settings to get started"
                  : "No conversations found"}
            </p>
            <a
              href={
                channelFilter === "instagram"
                  ? "/dashboard/settings?view=instagram"
                  : "/dashboard/settings?view=whatsapp"
              }
              className="mt-2 text-xs font-medium text-blue-600 hover:text-blue-700"
            >
              Go to settings →
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
                      {conv.contactDisplayName || conv.contactInstagramUsername || conv.contactPhone || "Unknown"}
                    </span>
                    <span className="flex items-center gap-1.5 shrink-0">
                      {pendingCounts && pendingCounts[conv.id] > 0 && (
                        <span className="inline-flex h-5 min-w-5 items-center justify-center rounded-full bg-violet-600 px-1.5 text-[10px] font-semibold text-white">
                          {pendingCounts[conv.id]}
                        </span>
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
