"use client";

import { useMemo } from "react";
import { Clock, MessageSquare, Reply, TrendingUp } from "lucide-react";
import type { Conversation } from "@/lib/models/conversation.model";
import { isConversationUnread } from "@/lib/inbox/unread";
import { ui } from "@/lib/copy/ui";
import { cn } from "@/lib/utils";

interface InboxMetricsProps {
  conversations: Conversation[];
  lastSeenAt?: Record<number, number>;
  pendingCounts?: Record<number, number>;
}

function isToday(dateStr?: string): boolean {
  if (!dateStr) return false;
  const date = new Date(dateStr);
  if (Number.isNaN(date.getTime())) return false;
  const now = new Date();
  return (
    date.getFullYear() === now.getFullYear() &&
    date.getMonth() === now.getMonth() &&
    date.getDate() === now.getDate()
  );
}

/**
 * Metric cards per messages-view composition. Only values computable from the
 * existing inbox data are shown; missing data sources render "—" (spec rule).
 */
export function InboxMetrics({
  conversations,
  lastSeenAt = {},
  pendingCounts,
}: InboxMetricsProps) {
  const metrics = useMemo(() => {
    const conversationsToday = conversations.filter((c) => isToday(c.lastMessageAt)).length;

    // "Por responder": active conversations with an unread (inbound) message
    // or a pending agent suggestion awaiting approval.
    const pendingReplies = conversations.filter(
      (c) =>
        c.status === "active" &&
        (isConversationUnread(c, lastSeenAt) || (pendingCounts?.[c.id] ?? 0) > 0)
    ).length;

    return {
      conversationsToday,
      pendingReplies,
    };
  }, [conversations, lastSeenAt, pendingCounts]);

  const cards = [
    {
      label: ui.inbox.metricConversationsToday,
      value: String(metrics.conversationsToday),
      icon: MessageSquare,
      iconClass: "bg-primary/10 text-primary",
    },
    {
      label: ui.inbox.metricPendingReplies,
      value: String(metrics.pendingReplies),
      icon: Reply,
      iconClass: "bg-amber-50 text-amber-600",
    },
    {
      label: ui.inbox.metricResponseRate,
      value: ui.dashboard.noData,
      icon: TrendingUp,
      iconClass: "bg-blue-50 text-blue-600",
    },
    {
      label: ui.inbox.metricAvgTime,
      value: ui.dashboard.noData,
      icon: Clock,
      iconClass: "bg-violet-50 text-violet-600",
    },
  ];

  return (
    <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
      {cards.map((card) => {
        const Icon = card.icon;
        return (
          <div
            key={card.label}
            className="rounded-2xl border border-border bg-white p-5 shadow-sm"
          >
            <div className="flex items-start justify-between gap-3">
              <div className="min-w-0">
                <p className="text-sm text-muted-foreground">{card.label}</p>
                <p className="mt-1.5 text-2xl font-bold tracking-tight text-foreground">
                  {card.value}
                </p>
              </div>
              <div
                className={cn(
                  "flex h-9 w-9 flex-none items-center justify-center rounded-lg",
                  card.iconClass
                )}
              >
                <Icon className="h-4 w-4" aria-hidden />
              </div>
            </div>
          </div>
        );
      })}
    </div>
  );
}
