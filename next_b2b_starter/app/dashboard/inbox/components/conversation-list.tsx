"use client";

import type { Conversation } from "@/lib/models/conversation.model";
import { Skeleton } from "@/components/ui/skeleton";
import { cn } from "@/lib/utils";

interface ConversationListProps {
  conversations: Conversation[];
  selectedId?: number;
  onSelect: (conv: Conversation) => void;
  isLoading: boolean;
  statusFilter: string;
  onStatusFilterChange: (filter: string) => void;
  pendingCounts?: Record<number, number>;
}

const statusTabs = [
  { label: "All", value: "" },
  { label: "Active", value: "active" },
  { label: "Closed", value: "closed" },
  { label: "Archived", value: "archived" },
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
  pendingCounts,
}: ConversationListProps) {
  return (
    <div className="flex h-full flex-col">
      <div className="border-b border-gray-200 px-4 py-3">
        <h2 className="text-lg font-semibold text-gray-900">Inbox</h2>
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
            <p className="text-sm text-gray-500">No conversations found</p>
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
                <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-gray-200 text-sm font-medium text-gray-600">
                  {(conv.contactDisplayName || conv.contactPhone || "?").charAt(0)}
                </div>
                <div className="min-w-0 flex-1">
                  <div className="flex items-center justify-between gap-2">
                    <span className="truncate text-sm font-medium text-gray-900">
                      {conv.contactDisplayName || conv.contactPhone || "Unknown"}
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
                      {conv.contactPhone && conv.contactPhone !== conv.contactDisplayName
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
