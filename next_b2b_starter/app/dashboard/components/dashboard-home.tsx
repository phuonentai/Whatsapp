"use client";

import { useMemo } from "react";
import Link from "next/link";

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { Badge } from "@/components/ui/badge";
import { ArrowRight, BookOpen, Inbox, Settings, Users } from "lucide-react";
import { format } from "date-fns";

import { useConversationsQuery } from "@/lib/hooks/queries/use-conversations-query";
import {
  useActivitiesQuery,
  useContactsQuery,
  useDealsQuery,
  usePipelinesQuery,
} from "@/lib/hooks/queries/use-crm-queries";
import { ui } from "@/lib/copy/ui";
import { FirstRunChecklist } from "@/components/onboarding/first-run-checklist";
import { AssistantIntro } from "@/components/onboarding/assistant-intro";

function KpiCard({
  label,
  value,
  isLoading,
  hint,
}: {
  label: string;
  value: string;
  isLoading?: boolean;
  hint?: string;
}) {
  return (
    <Card>
      <CardHeader className="pb-2">
        <CardDescription className="text-sm font-medium text-muted-foreground">
          {label}
        </CardDescription>
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <Skeleton className="h-8 w-16" />
        ) : (
          <p className="text-3xl font-semibold tracking-tight text-foreground">{value}</p>
        )}
        {hint ? <p className="mt-1 text-xs text-muted-foreground">{hint}</p> : null}
      </CardContent>
    </Card>
  );
}

const QUICK_ACTIONS = [
  {
    title: ui.dashboard.qaOpenInbox,
    description: ui.dashboard.qaOpenInboxDesc,
    href: "/dashboard/inbox",
    icon: Inbox,
  },
  {
    title: ui.dashboard.qaManageCrm,
    description: ui.dashboard.qaManageCrmDesc,
    href: "/dashboard/crm",
    icon: Users,
  },
  {
    title: ui.dashboard.qaKnowledgeBase,
    description: ui.dashboard.qaKnowledgeBaseDesc,
    href: "/dashboard/knowledge",
    icon: BookOpen,
  },
  {
    title: ui.dashboard.qaWorkspaceSettings,
    description: ui.dashboard.qaWorkspaceSettingsDesc,
    href: "/dashboard/settings",
    icon: Settings,
  },
];

export function DashboardHome() {
  const conversationsQuery = useConversationsQuery();
  const contactsQuery = useContactsQuery();
  const dealsQuery = useDealsQuery();
  const pipelinesQuery = usePipelinesQuery();
  const activitiesQuery = useActivitiesQuery({ limit: 8 });

  const openConversations = useMemo(
    () =>
      (conversationsQuery.data ?? []).filter(
        (conversation) => conversation.status !== "closed"
      ).length,
    [conversationsQuery.data]
  );

  const stageNameById = useMemo(() => {
    const map = new Map<number, string>();
    for (const pipeline of pipelinesQuery.data ?? []) {
      for (const stage of pipeline.etapas) {
        map.set(stage.id, stage.nombre);
      }
    }
    return map;
  }, [pipelinesQuery.data]);

  const dealsByStage = useMemo(() => {
    const counts = new Map<string, number>();
    for (const deal of dealsQuery.data ?? []) {
      const stageName = deal.stage_id != null
        ? (stageNameById.get(deal.stage_id) ?? ui.dashboard.noStage)
        : ui.dashboard.noStage;
      counts.set(stageName, (counts.get(stageName) ?? 0) + 1);
    }
    return Array.from(counts.entries())
      .sort((a, b) => b[1] - a[1])
      .slice(0, 3);
  }, [dealsQuery.data, stageNameById]);

  const isLoadingKpis =
    conversationsQuery.isLoading || contactsQuery.isLoading || dealsQuery.isLoading;

  return (
    <div className="space-y-8">
      <div className="space-y-1">
        <h1 className="text-3xl font-semibold text-foreground">{ui.dashboard.title}</h1>
        <p className="text-sm text-muted-foreground">
          {ui.dashboard.subtitle}
        </p>
      </div>

      <div className="grid gap-6 lg:grid-cols-2">
        <AssistantIntro />
        <FirstRunChecklist />
      </div>

      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        <KpiCard
          label={ui.dashboard.kpiOpenConversations}
          value={String(openConversations)}
          isLoading={conversationsQuery.isLoading}
          hint={ui.dashboard.kpiOpenConversationsHint}
        />
        <KpiCard
          label={ui.dashboard.kpiContacts}
          value={String(contactsQuery.data?.length ?? 0)}
          isLoading={contactsQuery.isLoading}
          hint={ui.dashboard.kpiContactsHint}
        />
        <KpiCard
          label={ui.dashboard.kpiDeals}
          value={String(dealsQuery.data?.length ?? 0)}
          isLoading={dealsQuery.isLoading}
          hint={ui.dashboard.kpiDealsHint}
        />
      </div>

      <div className="grid gap-6 lg:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle className="text-lg text-foreground">{ui.dashboard.dealsByStage}</CardTitle>
            <CardDescription>{ui.dashboard.dealsByStageDesc}</CardDescription>
          </CardHeader>
          <CardContent>
            {dealsQuery.isLoading ? (
              <div className="space-y-2">
                <Skeleton className="h-5 w-full" />
                <Skeleton className="h-5 w-full" />
                <Skeleton className="h-5 w-3/4" />
              </div>
            ) : dealsByStage.length === 0 ? (
              <p className="text-sm text-muted-foreground">
                {ui.dashboard.noDeals}
              </p>
            ) : (
              <ul className="space-y-3">
                {dealsByStage.map(([stage, count]) => (
                  <li
                    key={stage}
                    className="flex items-center justify-between gap-4 text-sm"
                  >
                    <span className="text-foreground">{stage}</span>
                    <Badge variant="secondary">{count}</Badge>
                  </li>
                ))}
              </ul>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="text-lg text-foreground">{ui.dashboard.recentActivity}</CardTitle>
            <CardDescription>{ui.dashboard.recentActivityDesc}</CardDescription>
          </CardHeader>
          <CardContent>
            {activitiesQuery.isLoading ? (
              <div className="space-y-2">
                <Skeleton className="h-5 w-full" />
                <Skeleton className="h-5 w-full" />
                <Skeleton className="h-5 w-3/4" />
              </div>
            ) : (activitiesQuery.data ?? []).length === 0 ? (
              <p className="text-sm text-muted-foreground">
                {ui.dashboard.noActivity}
              </p>
            ) : (
              <ul className="space-y-3">
                {(activitiesQuery.data ?? []).map((activity) => (
                  <li key={activity.id} className="flex items-start justify-between gap-4 text-sm">
                    <div className="min-w-0">
                      <p className="truncate font-medium text-foreground">
                        {activity.asunto || activity.tipo}
                      </p>
                      <p className="text-xs text-muted-foreground">
                        {activity.tipo}
                      </p>
                    </div>
                    <span className="flex-none text-xs text-muted-foreground">
                      {activity.realizada_en
                        ? format(new Date(activity.realizada_en), "MMM d")
                        : ""}
                    </span>
                  </li>
                ))}
              </ul>
            )}
          </CardContent>
        </Card>
      </div>

      <div>
        <h2 className="mb-3 text-lg font-semibold text-foreground">{ui.dashboard.quickActions}</h2>
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          {QUICK_ACTIONS.map((action) => {
            const Icon = action.icon;
            return (
              <Link
                key={action.href}
                href={action.href}
                className="group rounded-xl border border-border bg-card p-5 shadow-sm transition-colors hover:bg-accent"
              >
                <Icon className="h-5 w-5 text-muted-foreground" aria-hidden />
                <p className="mt-3 flex items-center gap-1 font-medium text-foreground">
                  {action.title}
                  <ArrowRight
                    className="h-3.5 w-3.5 text-muted-foreground transition-transform group-hover:translate-x-0.5"
                    aria-hidden
                  />
                </p>
                <p className="mt-1 text-sm text-muted-foreground">
                  {action.description}
                </p>
              </Link>
            );
          })}
        </div>
      </div>
    </div>
  );
}
