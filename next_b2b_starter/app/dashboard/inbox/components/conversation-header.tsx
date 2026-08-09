"use client";

import { useState } from "react";
import type { Conversation } from "@/lib/models/conversation.model";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { useModule } from "@/lib/hooks/use-entitlement";
import { useCreateTicket } from "@/lib/hooks/mutations/use-tickets-mutations";

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
  const ticketsModule = useModule("tickets");
  const createTicket = useCreateTicket();
  const [ticketTitle, setTicketTitle] = useState("");

  const handleCreateTicket = () => {
    createTicket.mutate({
      contact_id: conversation.contactId,
      conversation_id: conversation.id,
      title: ticketTitle.trim() || `Ticket de ${displayName}`,
    });
    setTicketTitle("");
  };

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
        {ticketsModule.enabled && (
          <input
            value={ticketTitle}
            onChange={(e) => setTicketTitle(e.target.value)}
            placeholder="Título del ticket..."
            className="w-40 rounded border px-2 py-1 text-xs"
          />
        )}
        {ticketsModule.enabled && (
          <Button variant="outline" size="sm" onClick={handleCreateTicket} disabled={createTicket.isPending}>
            Crear ticket
          </Button>
        )}
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
