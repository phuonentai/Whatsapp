"use client";

import { useState } from "react";
import type { Conversation } from "@/lib/models/conversation.model";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { Instagram, MessageCircle, ShieldAlert, UserRound } from "lucide-react";
import { useModule } from "@/lib/hooks/use-entitlement";
import { useCreateTicket } from "@/lib/hooks/mutations/use-tickets-mutations";
import { useConversationContextQuery } from "@/lib/hooks/queries/use-conversation-context-query";
import { useMemberDirectoryQuery } from "@/lib/hooks/queries/use-member-directory-query";
import { useUpdateConversationAssignee } from "@/lib/hooks/mutations/use-update-conversation-assignee";
import { ui } from "@/lib/copy/ui";

interface ConversationHeaderProps {
  conversation: Conversation;
  onToggleStatus: () => void;
  isUpdating: boolean;
  /** Members without org:manage cannot close/reopen conversations. */
  canManage?: boolean;
  /** conversation-row-scoping: permiso inbox:reassign (u org:manage). */
  canReassign?: boolean;
  /** conversation-row-scoping: flag de entitlement (solo planes pagos). */
  rowScopingEnabled?: boolean;
}

export function ConversationHeader({
  conversation,
  onToggleStatus,
  isUpdating,
  canManage = true,
  canReassign = false,
  rowScopingEnabled = false,
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
  const { data: context } = useConversationContextQuery(conversation.id);
  const consentGated = context?.consentGated ?? false;

  // conversation-row-scoping (task 5.4): picker de asignación gated por
  // flag + permiso. El directorio (Stytch Members API cacheada) puede estar
  // no disponible (503 member_directory_unavailable) → estado de retry sin
  // ghost; el thread y el composer permanecen operativos.
  const showPicker = rowScopingEnabled && canReassign;
  const directory = useMemberDirectoryQuery();
  const reassign = useUpdateConversationAssignee();
  const [pickerOpen, setPickerOpen] = useState(false);
  const directoryAvailable = !directory.isLoading && !directory.isError;
  const currentAssignee = conversation.assigneeStytchMemberId;

  const handleAssign = (memberId: string | null) => {
    setPickerOpen(false);
    reassign.mutate({ conversationId: conversation.id, assignee: memberId });
  };

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
          {consentGated && (
            <p className="mt-0.5 inline-flex items-center gap-1 text-[11px] text-amber-600">
              <ShieldAlert className="h-3 w-3" />
              {ui.agent.contextConsentIndicator}
            </p>
          )}
        </div>
      </div>

      <div className="flex items-center gap-2">
        {/* Assignee (conversation-row-scoping): chip informativo + picker de
            re-asignación. Oculto en free tier y sin permiso (hide, no ghost). */}
        {showPicker && (
          <div className="relative">
            <button
              type="button"
              onClick={() => setPickerOpen((o) => !o)}
              className="inline-flex items-center gap-1.5 rounded-full border border-slate-200 bg-white px-2.5 py-1 text-xs font-medium text-slate-700 hover:bg-slate-50"
              aria-expanded={pickerOpen}
              aria-haspopup="listbox"
            >
              <UserRound className="h-3.5 w-3.5 text-slate-400" />
              {currentAssignee ? (
                <span className="max-w-[10rem] truncate">{currentAssignee.slice(0, 8)}</span>
              ) : (
                ui.inbox.unassigned
              )}
            </button>
            {pickerOpen && (
              <div
                role="listbox"
                aria-label={ui.inbox.assigneePickerAria}
                className="absolute right-0 top-9 z-20 w-56 rounded-lg border border-slate-200 bg-white p-2 shadow-lg"
              >
                {!directoryAvailable ? (
                  // Directorio no disponible: estado de retry (sin ghost).
                  <div className="space-y-2 px-1 py-2 text-center">
                    <p className="text-xs text-slate-500">{ui.inbox.memberDirectoryUnavailable}</p>
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => directory.refetch()}
                      disabled={directory.isRefetching}
                    >
                      {ui.inbox.retry}
                    </Button>
                  </div>
                ) : (
                  <>
                    <button
                      type="button"
                      role="option"
                      aria-selected={!currentAssignee}
                      onClick={() => handleAssign(null)}
                      className="w-full rounded px-2 py-1.5 text-left text-xs text-slate-700 hover:bg-slate-50"
                    >
                      {ui.inbox.unassign}
                    </button>
                    <div className="my-1 border-t border-slate-100" />
                    {(directory.data ?? []).map((memberId) => (
                      <button
                        key={memberId}
                        type="button"
                        role="option"
                        aria-selected={currentAssignee === memberId}
                        onClick={() => handleAssign(memberId)}
                        className="block w-full truncate rounded px-2 py-1.5 text-left text-xs text-slate-700 hover:bg-slate-50"
                      >
                        {memberId.slice(0, 12)}
                      </button>
                    ))}
                    {!currentAssignee && (
                      <button
                        type="button"
                        role="option"
                        aria-selected={false}
                        onClick={() => handleAssign(conversation.assigneeStytchMemberId ?? null)}
                        className="hidden"
                      />
                    )}
                    {(directory.data ?? []).length === 0 && (
                      <p className="px-1 py-2 text-center text-xs text-slate-400">
                        {ui.inbox.noAssignableMembers}
                      </p>
                    )}
                  </>
                )}
              </div>
            )}
          </div>
        )}
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
            conversation.status === "active" && "bg-primary/10 text-primary",
            conversation.status === "closed" && "bg-gray-100 text-gray-600",
            conversation.status === "archived" && "bg-orange-50 text-orange-600"
          )}
        >
          {conversation.status}
        </span>
        {canManage && (
          <Button
            variant="outline"
            size="sm"
            onClick={onToggleStatus}
            disabled={isUpdating}
          >
            {conversation.status === "active" ? "Close" : "Reopen"}
          </Button>
        )}
      </div>
    </div>
  );
}
