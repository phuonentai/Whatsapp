"use client";

import { useState } from "react";
import type { Conversation } from "@/lib/models/conversation.model";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { Instagram, MessageCircle } from "lucide-react";
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
  const displayName =
    conversation.contactDisplayName ||
    conversation.contactInstagramUsername ||
    conversation.contactPhone ||
    "Unknown";
  const isInstagram = conversation.channel === "instagram";
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
        <div className="relative">
          <div className="flex h-10 w-10 items-center justify-center rounded-full bg-gray-200 text-sm font-medium text-gray-600">
            {conversation.contactAvatarUrl ? (
              // eslint-disable-next-line @next/next/no-img-element
              <img
                src={conversation.contactAvatarUrl}
                alt={displayName}
                className="h-10 w-10 rounded-full object-cover"
              />
            ) : (
              initial
            )}
          </div>
          <span className="absolute -bottom-1 -right-1 flex h-5 w-5 items-center justify-center rounded-full border border-white bg-white">
            {isInstagram ? (
              <Instagram className="h-3.5 w-3.5 text-pink-600" />
            ) : (
              <MessageCircle className="h-3.5 w-3.5 text-green-600" />
            )}
          </span>
        </div>
        <div>
          <div className="flex items-center gap-2">
            <p className="text-sm font-medium text-gray-900">{displayName}</p>
            <span
              className={cn(
                "shrink-0 rounded-full px-1.5 py-0.5 text-[10px] font-semibold uppercase",
                isInstagram
                  ? "bg-pink-50 text-pink-600"
                  : "bg-green-50 text-green-600"
              )}
            >
              {isInstagram ? "Instagram" : "WhatsApp"}
            </span>
          </div>
          {!isInstagram && conversation.contactPhone && conversation.contactPhone !== displayName && (
            <p className="text-xs text-gray-500">{conversation.contactPhone}</p>
          )}
          {isInstagram && conversation.contactInstagramUsername && (
            <p className="text-xs text-gray-500">@{conversation.contactInstagramUsername}</p>
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
