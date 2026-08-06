"use client";

import type { Conversation } from "@/lib/models/conversation.model";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

interface ConversationHeaderProps {
  conversation: Conversation;
  onToggleStatus: () => void;
  isUpdating: boolean;
}

export function ConversationHeader({
  conversation,
  onToggleStatus,
  isUpdating,
}: ConversationHeaderProps) {
  const displayName = conversation.contactDisplayName || conversation.contactPhone || "Unknown";
  const initial = displayName.charAt(0);

  return (
    <div className="flex items-center justify-between border-b border-gray-200 bg-white px-4 py-3">
      <div className="flex items-center gap-3">
        <div className="flex h-10 w-10 items-center justify-center rounded-full bg-gray-200 text-sm font-medium text-gray-600">
          {initial}
        </div>
        <div>
          <p className="text-sm font-medium text-gray-900">{displayName}</p>
          {conversation.contactPhone && conversation.contactPhone !== displayName && (
            <p className="text-xs text-gray-500">{conversation.contactPhone}</p>
          )}
        </div>
      </div>

      <div className="flex items-center gap-2">
        <span
          className={cn(
            "rounded-full px-2 py-0.5 text-xs font-medium",
            conversation.status === "active" && "bg-emerald-50 text-emerald-700",
            conversation.status === "closed" && "bg-gray-100 text-gray-600",
            conversation.status === "archived" && "bg-orange-50 text-orange-600"
          )}
        >
          {conversation.status}
        </span>
        <Button
          variant="outline"
          size="sm"
          onClick={onToggleStatus}
          disabled={isUpdating}
        >
          {conversation.status === "active" ? "Close" : "Reopen"}
        </Button>
      </div>
    </div>
  );
}
