"use client";

import { useEffect, useMemo, useRef, useState } from "react";
import Link from "next/link";
import { useQuery } from "@tanstack/react-query";

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import {
  ArrowRight,
  BarChart as BarChartIcon,
  Bot,
  ChevronDown,
  Clock,
  Download,
  FileText,
  Inbox,
  Megaphone,
  MessageSquare,
  Rocket,
  Sparkles,
  TrendingUp,
  UserPlus,
  Users,
} from "lucide-react";
import {
  Bar,
  BarChart,
  CartesianGrid,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";

import { useConversationsQuery } from "@/lib/hooks/queries/use-conversations-query";
import { useMembersQuery } from "@/lib/hooks/queries/use-members-query";
import { useAgentSettingsQuery } from "@/lib/hooks/queries/use-agent-settings-query";
import { useModule } from "@/lib/hooks/use-entitlement";
import { usePermissions } from "@/lib/hooks/use-permissions";
import { PERMISSIONS } from "@/lib/auth/permissions";
import { useInboxStore } from "@/lib/stores/use-inbox-store";
import { isConversationUnread } from "@/lib/inbox/unread";
import { analyticsRepository } from "@/lib/api/api/repositories/analytics-repository";
import type { RevenueParams } from "@/lib/api/api/repositories/analytics-repository";
import { queryKeys } from "@/lib/hooks/queries/query-keys";
import { ui, tpl } from "@/lib/copy/ui";
import { FirstRunChecklist } from "@/components/onboarding/first-run-checklist";
import { AssistantIntro } from "@/components/onboarding/assistant-intro";
import type { Conversation } from "@/lib/models/conversation.model";
import { cn } from "@/lib/utils";

function formatCOP(value: number): string {
  return new Intl.NumberFormat("es-CO", {
    style: "currency",
    currency: "COP",
    maximumFractionDigits: 0,
  }).format(value);
}

function formatDay(periodo: string): string {
  const [y, m, d] = periodo.split("-");
  if (!y || !m || !d) return periodo;
  return `${d}/${m}`;
}

function timeAgo(dateStr?: string): string {
  if (!dateStr) return "";
  const date = new Date(dateStr);
  if (Number.isNaN(date.getTime())) return "";
  const mins = Math.floor((Date.now() - date.getTime()) / 60000);
  if (mins < 1) return ui.inbox.timeJustNow;
  if (mins < 60) return tpl(ui.inbox.timeMin, { n: mins });
  const hrs = Math.floor(mins / 60);
  if (hrs < 24) return tpl(ui.inbox.timeHour, { n: hrs });
  const days = Math.floor(hrs / 24);
  return tpl(ui.inbox.timeDay, { n: days });
}

function getInitials(displayName: string): string {
  const parts = displayName.trim().split(/\s+/).slice(0, 2);
  return parts.map((part) => part[0] ?? "").join("").toUpperCase();
}

function conversationDisplayName(conversation: Conversation): string {
  return (
    conversation.contactDisplayName ||
    conversation.contactInstagramUsername ||
    conversation.contactPhone ||
    ui.inbox.unknownContact
  );
}

interface KpiDelta {
  direction: "up" | "down";
  value: number;
}

function KpiCard({
  label,
  value,
  isLoading,
  hint,
  delta,
  icon: Icon,
  iconClass,
}: {
  label: string;
  value: string;
  isLoading?: boolean;
  hint?: string;
  delta?: KpiDelta;
  icon: React.ComponentType<{ className?: string }>;
  iconClass?: string;
}) {
  return (
    <div className="rounded-2xl border border-border bg-card p-6 shadow-sm">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <div className="flex items-center gap-2">
            <p className="text-sm font-medium text-muted-foreground">{label}</p>
            {delta ? (
              <span
                className={cn(
                  "inline-flex flex-none items-center rounded-full px-2 py-0.5 text-[11px] font-semibold",
                  delta.direction === "up"
                    ? "bg-emerald-50 text-emerald-600"
                    : "bg-red-50 text-red-600"
                )}
                aria-label={ui.dashboard.kpiDeltaAria}
              >
                {tpl(
                  delta.direction === "up"
                    ? ui.dashboard.kpiDeltaUp
                    : ui.dashboard.kpiDeltaDown,
                  { delta: delta.value }
                )}
              </span>
            ) : null}
          </div>
          {isLoading ? (
            <Skeleton className="mt-2 h-8 w-16" />
          ) : (
            <p className="mt-1.5 text-2xl font-bold tracking-tight text-foreground">
              {value}
            </p>
          )}
          {hint ? <p className="mt-1 text-xs text-muted-foreground">{hint}</p> : null}
        </div>
        <div
          className={cn(
            "flex h-11 w-11 flex-none items-center justify-center rounded-xl",
            iconClass ?? "bg-primary/10 text-primary"
          )}
        >
          <Icon className="h-5 w-5" aria-hidden />
        </div>
      </div>
    </div>
  );
}

const COPILOT_INSIGHTS = [
  { text: ui.dashboard.copilotInsight1, dotClass: "bg-emerald-400" },
  { text: ui.dashboard.copilotInsight2, dotClass: "bg-amber-400" },
  { text: ui.dashboard.copilotInsight3, dotClass: "bg-blue-400" },
];

type SalesPeriod = "7d" | "30d" | "month" | "quarter";

const SALES_PERIODS: Array<{ key: SalesPeriod; label: string }> = [
  { key: "7d", label: ui.dashboard.salesPeriod7d },
  { key: "30d", label: ui.dashboard.salesPeriod30d },
  { key: "month", label: ui.dashboard.salesPeriodMonth },
  { key: "quarter", label: ui.dashboard.salesPeriodQuarter },
];

function isoDate(daysAgo: number): string {
  const d = new Date();
  d.setDate(d.getDate() - daysAgo);
  return d.toISOString().slice(0, 10);
}

function firstOfMonthIso(): string {
  const d = new Date();
  d.setDate(1);
  return d.toISOString().slice(0, 10);
}

function timeGreeting(): string {
  const hour = new Date().getHours();
  if (hour < 12) return ui.layout.greetingMorning;
  if (hour < 19) return ui.layout.greetingAfternoon;
  return ui.layout.greetingEvening;
}

/** Days spanned by a sales period (used to compute the previous-period comparison window). */
function salesPeriodLength(period: SalesPeriod): number {
  switch (period) {
    case "7d":
      return 7;
    case "30d":
      return 30;
    case "quarter":
      return 90;
    case "month": {
      const now = new Date();
      return new Date(now.getFullYear(), now.getMonth() + 1, 0).getDate();
    }
  }
}

/** Previous equal-length window for a sales period, expressed as explicit dates. */
function previousSalesParams(period: SalesPeriod): RevenueParams {
  const length = salesPeriodLength(period);
  const today = isoDate(0);
  return { period: "week", from: isoDate(length * 2), to: isoDate(length) };
}

// ---- Widgets de la home recompuesta (dashboard-home-redesign) ----

/** Panel "Conversaciones Recientes": misma fuente que la bandeja, solo snippets. */
function RecentConversationsPanel({
  conversations,
  isLoading,
}: {
  conversations: Conversation[];
  isLoading: boolean;
}) {
  const lastSeenAt = useInboxStore((s) => s.lastSeenAt);

  const recent = useMemo(
    () =>
      [...conversations]
        .sort((a, b) =>
          (b.lastMessageAt ?? b.createdAt).localeCompare(a.lastMessageAt ?? a.createdAt)
        )
        .slice(0, 5),
    [conversations]
  );

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-lg text-foreground">{ui.dashboard.recentConversationsTitle}</CardTitle>
        <CardDescription>{ui.dashboard.recentConversationsDesc}</CardDescription>
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <div className="space-y-3">
            <Skeleton className="h-12 w-full" />
            <Skeleton className="h-12 w-full" />
            <Skeleton className="h-12 w-3/4" />
          </div>
        ) : recent.length === 0 ? (
          <div className="flex flex-col items-center justify-center rounded-xl border border-dashed border-border py-10 text-center">
            <MessageSquare className="h-8 w-8 text-muted-foreground/50" aria-hidden />
            <p className="mt-3 max-w-xs text-sm text-muted-foreground">
              {ui.dashboard.recentConversationsEmpty}
            </p>
            <Link
              href="/dashboard/inbox"
              className="mt-4 inline-flex items-center gap-1.5 rounded-lg bg-primary px-3.5 py-2 text-xs font-semibold text-primary-foreground transition-colors hover:bg-primary/90"
            >
              {ui.dashboard.recentConversationsCta}
              <ArrowRight className="h-3.5 w-3.5" aria-hidden />
            </Link>
          </div>
        ) : (
          <ul className="divide-y divide-border">
            {recent.map((conversation) => {
              const unread = isConversationUnread(conversation, lastSeenAt);
              return (
                <li key={conversation.id}>
                  <Link
                    href="/dashboard/inbox"
                    className="flex items-center gap-3 py-3 transition-colors hover:bg-accent/50"
                  >
                    <div className="flex h-9 w-9 flex-none items-center justify-center rounded-full bg-muted text-xs font-semibold text-muted-foreground">
                      {getInitials(conversationDisplayName(conversation))}
                    </div>
                    <div className="min-w-0 flex-1">
                      <p
                        className={cn(
                          "truncate text-sm text-foreground",
                          unread && "font-semibold"
                        )}
                      >
                        {conversationDisplayName(conversation)}
                      </p>
                      <p className="truncate text-xs text-muted-foreground">
                        {conversation.channel === "instagram"
                          ? conversation.contactInstagramUsername &&
                            conversation.contactInstagramUsername !==
                              conversation.contactDisplayName
                            ? `@${conversation.contactInstagramUsername}`
                            : "Instagram"
                          : conversation.contactPhone &&
                              conversation.contactPhone !== conversation.contactDisplayName
                            ? conversation.contactPhone
                            : "WhatsApp"}
                      </p>
                    </div>
                    <span className="flex flex-none items-center gap-1.5 text-xs text-muted-foreground">
                      {unread && (
                        <span
                          aria-label={ui.inbox.unreadAria}
                          className="inline-block h-2 w-2 rounded-full bg-primary"
                        />
                      )}
                      {timeAgo(conversation.lastMessageAt)}
                    </span>
                  </Link>
                </li>
              );
            })}
          </ul>
        )}
      </CardContent>
    </Card>
  );
}

/** Panel "Rendimiento del Equipo": gate = superficie de miembros (ORG_MANAGE); sin % inventados. */
function TeamPerformancePanel({
  organizationId,
  canManageMembers,
}: {
  organizationId?: string;
  canManageMembers: boolean;
}) {
  const membersQuery = useMembersQuery({
    organizationId,
    page: 1,
    pageSize: 50,
    enabled: canManageMembers && Boolean(organizationId),
  });

  const activeMembers = useMemo(
    () => (membersQuery.data?.members ?? []).filter((member) => member.status === "active"),
    [membersQuery.data]
  );

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-lg text-foreground">{ui.dashboard.teamPerformanceTitle}</CardTitle>
        <CardDescription>{ui.dashboard.teamPerformanceDesc}</CardDescription>
      </CardHeader>
      <CardContent>
        {!canManageMembers ? (
          <p className="py-6 text-center text-sm text-muted-foreground">
            {ui.dashboard.teamPerformanceEmpty}
          </p>
        ) : membersQuery.isLoading ? (
          <div className="space-y-3">
            <Skeleton className="h-10 w-full" />
            <Skeleton className="h-10 w-full" />
            <Skeleton className="h-10 w-3/4" />
          </div>
        ) : activeMembers.length === 0 ? (
          <div className="flex flex-col items-center justify-center rounded-xl border border-dashed border-border py-10 text-center">
            <Users className="h-8 w-8 text-muted-foreground/50" aria-hidden />
            <p className="mt-3 max-w-xs text-sm text-muted-foreground">
              {ui.dashboard.teamPerformanceEmpty}
            </p>
            <Link
              href="/dashboard/settings?view=members"
              className="mt-4 inline-flex items-center gap-1.5 rounded-lg bg-primary px-3.5 py-2 text-xs font-semibold text-primary-foreground transition-colors hover:bg-primary/90"
            >
              {ui.dashboard.teamPerformanceCta}
              <ArrowRight className="h-3.5 w-3.5" aria-hidden />
            </Link>
          </div>
        ) : (
          <ul className="divide-y divide-border">
            {activeMembers.slice(0, 5).map((member) => (
              <li key={member.id} className="flex items-center gap-3 py-3">
                <div className="flex h-9 w-9 flex-none items-center justify-center rounded-full bg-muted text-xs font-semibold text-muted-foreground">
                  {getInitials(member.name || member.email)}
                </div>
                <div className="min-w-0 flex-1">
                  <p className="truncate text-sm font-medium text-foreground">
                    {member.name || member.email}
                  </p>
                  <p className="truncate text-xs text-muted-foreground">{member.email}</p>
                </div>
                <span className="flex-none rounded-full bg-muted px-2 py-0.5 text-[10px] font-medium capitalize text-muted-foreground">
                  {member.role}
                </span>
              </li>
            ))}
          </ul>
        )}
      </CardContent>
    </Card>
  );
}

/** Panel "Facturas Siigo": sin endpoint de lista hoy → estado vacío honesto + CTA (gate invoice:view). */
function SiigoInvoicesPanel({ canViewInvoices }: { canViewInvoices: boolean }) {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-lg text-foreground">{ui.dashboard.siigoInvoicesTitle}</CardTitle>
        <CardDescription>{ui.dashboard.siigoInvoicesDesc}</CardDescription>
      </CardHeader>
      <CardContent>
        <div className="flex flex-col items-center justify-center rounded-xl border border-dashed border-border py-10 text-center">
          <FileText className="h-8 w-8 text-muted-foreground/50" aria-hidden />
          <p className="mt-3 max-w-xs text-sm text-muted-foreground">
            {ui.dashboard.siigoInvoicesEmpty}
          </p>
          {canViewInvoices && (
            <Link
              href="/dashboard/settings?view=siigo"
              className="mt-4 inline-flex items-center gap-1.5 rounded-lg bg-primary px-3.5 py-2 text-xs font-semibold text-primary-foreground transition-colors hover:bg-primary/90"
            >
              {ui.dashboard.siigoInvoicesCta}
              <ArrowRight className="h-3.5 w-3.5" aria-hidden />
            </Link>
          )}
        </div>
      </CardContent>
    </Card>
  );
}

/**
 * Banner "Auto-Piloto": refleja `mode` de agent-settings SOLO cuando está confirmado por el modelo;
 * sin dato → sugerencia estática sin afirmar modo (gate: misma condición que settings?view=ai).
 */
function AutoPilotBanner({ canManageMembers }: { canManageMembers: boolean }) {
  const agentSettingsQuery = useAgentSettingsQuery({ enabled: canManageMembers });
  const settings = agentSettingsQuery.data;

  let body: string;
  if (settings?.mode === "autopilot") {
    body = settings.kill_switch ? ui.dashboard.autopilotPaused : ui.dashboard.autopilotAutopilot;
  } else if (settings?.mode === "copilot") {
    body = ui.dashboard.autopilotCopilot;
  } else {
    body = ui.dashboard.autopilotStatic;
  }

  return (
    <div className="flex flex-col gap-4 rounded-2xl border border-primary/20 bg-gradient-to-r from-primary/5 to-background p-6 sm:flex-row sm:items-center sm:justify-between">
      <div className="flex items-start gap-3">
        <div className="flex h-10 w-10 flex-none items-center justify-center rounded-xl bg-primary/10">
          <Rocket className="h-5 w-5 text-primary" aria-hidden />
        </div>
        <div className="space-y-1">
          <h2 className="font-semibold text-foreground">{ui.dashboard.autopilotTitle}</h2>
          <p className="max-w-xl text-sm text-muted-foreground">{body}</p>
        </div>
      </div>
      <Link
        href="/dashboard/settings?view=ai"
        className="inline-flex flex-none items-center gap-2 rounded-xl bg-primary px-4 py-2.5 text-sm font-semibold text-primary-foreground transition-colors hover:bg-primary/90"
      >
        <Sparkles className="h-4 w-4" aria-hidden />
        {ui.dashboard.autopilotCta}
      </Link>
    </div>
  );
}

const ONBOARDING_FOLD_KEY = "dashboard-home.onboarding-checklist-folded";

function readFoldPreference(): boolean {
  if (typeof window === "undefined") return false;
  try {
    return window.localStorage.getItem(ONBOARDING_FOLD_KEY) === "1";
  } catch {
    // localStorage no disponible (privacidad/cuota): por defecto expandido.
    return false;
  }
}

/**
 * Envoltorio colapsable del checklist de primer uso: plegado manual persistido en localStorage,
 * visible por defecto en primer uso; cuando el checklist se completa se auto-oculta (contrato
 * ai-onboarding: "disappear once all steps are complete") y el envoltorio desaparece con él.
 */
function OnboardingChecklistSection() {
  const bodyRef = useRef<HTMLDivElement>(null);
  const [folded, setFolded] = useState<boolean>(readFoldPreference);
  const [hasContent, setHasContent] = useState(true);

  useEffect(() => {
    const el = bodyRef.current;
    if (!el) return;
    const update = () => setHasContent(el.childElementCount > 0);
    update();
    const observer = new MutationObserver(update);
    observer.observe(el, { childList: true });
    return () => observer.disconnect();
  }, []);

  const toggle = () => {
    setFolded((current) => {
      const next = !current;
      try {
        localStorage.setItem(ONBOARDING_FOLD_KEY, next ? "1" : "0");
      } catch {
        // preferencia no persistible: el estado en memoria sigue funcionando.
      }
      return next;
    });
  };

  if (!hasContent) {
    return null;
  }

  return (
    <div>
      <div className="flex items-center justify-between gap-3">
        <h2 className="text-lg font-semibold text-foreground">
          {ui.dashboard.onboardingSectionTitle}
        </h2>
        <button
          type="button"
          onClick={toggle}
          aria-expanded={!folded}
          aria-label={ui.dashboard.onboardingChecklistToggle}
          className="rounded-lg p-1.5 text-muted-foreground transition-colors hover:bg-accent hover:text-accent-foreground"
        >
          <ChevronDown
            className={cn("h-4 w-4 transition-transform", folded && "-rotate-90")}
            aria-hidden
          />
        </button>
      </div>
      {!folded && (
        <div ref={bodyRef} className="mt-3">
          <FirstRunChecklist />
        </div>
      )}
    </div>
  );
}

const QUICK_ACTIONS = [
  // Broadcast → la superficie real de campañas es la pestaña "Campañas" del CRM.
  {
    title: ui.dashboard.qaBroadcast,
    description: ui.dashboard.qaBroadcastDesc,
    href: "/dashboard/crm?view=campa%C3%B1as",
    icon: Megaphone,
  },
  {
    title: ui.dashboard.qaNewInvoice,
    description: ui.dashboard.qaNewInvoiceDesc,
    href: "/dashboard/settings?view=siigo",
    icon: FileText,
  },
  {
    title: ui.dashboard.qaNewContact,
    description: ui.dashboard.qaNewContactDesc,
    href: "/dashboard/crm",
    icon: UserPlus,
  },
  {
    title: ui.dashboard.qaExport,
    description: ui.dashboard.qaExportDesc,
    href: "/dashboard/reportes",
    icon: Download,
  },
];

export function DashboardHome() {
  const { profile, hasPermission } = usePermissions();
  const conversationsQuery = useConversationsQuery();

  const analyticsModule = useModule("analytics");
  const canViewInvoices = hasPermission(PERMISSIONS.INVOICE_VIEW);
  const canManageMembers = hasPermission(PERMISSIONS.ORG_MANAGE);
  const [salesPeriod, setSalesPeriod] = useState<SalesPeriod>("7d");

  // Period → analytics params. The backend accepts period "week"|"month" plus
  // optional from/to (YYYY-MM-DD); the 7d/30d/quarter options are expressed as
  // explicit date ranges (frontend-only, no new endpoints).
  const salesParams = useMemo<RevenueParams>(() => {
    const today = isoDate(0);
    switch (salesPeriod) {
      case "7d":
        return { period: "week", from: isoDate(7), to: today };
      case "30d":
        return { period: "week", from: isoDate(30), to: today };
      case "quarter":
        return { period: "week", from: isoDate(90), to: today };
      default:
        return { period: "month", from: firstOfMonthIso(), to: today };
    }
  }, [salesPeriod]);

  const revenueQuery = useQuery({
    queryKey: [
      ...queryKeys.modules.all,
      "analytics",
      "revenue",
      salesParams.period,
      salesParams.from,
      salesParams.to,
    ],
    queryFn: () => analyticsRepository.revenue(salesParams),
    enabled: analyticsModule.enabled && canViewInvoices,
    staleTime: 60 * 1000,
  });

  // Ventana anterior de igual longitud para el badge de delta (mismo endpoint, sin fan-out nuevo).
  const prevSalesParams = useMemo(() => previousSalesParams(salesPeriod), [salesPeriod]);
  const prevRevenueQuery = useQuery({
    queryKey: [
      ...queryKeys.modules.all,
      "analytics",
      "revenue",
      "previous",
      prevSalesParams.period,
      prevSalesParams.from,
      prevSalesParams.to,
    ],
    queryFn: () => analyticsRepository.revenue(prevSalesParams),
    enabled: analyticsModule.enabled && canViewInvoices,
    staleTime: 60 * 1000,
  });

  const salesAvailable = analyticsModule.enabled && canViewInvoices;

  const openConversations = useMemo(
    () =>
      (conversationsQuery.data ?? []).filter(
        (conversation) => conversation.status !== "closed"
      ).length,
    [conversationsQuery.data]
  );

  const weekSales = useMemo(
    () => (revenueQuery.data ?? []).reduce((sum, p) => sum + p.monto_total, 0),
    [revenueQuery.data]
  );

  const chartData = useMemo(
    () =>
      (revenueQuery.data ?? []).map((p) => ({
        name: formatDay(p.periodo),
        ventas: p.monto_total,
      })),
    [revenueQuery.data]
  );

  // Solo existe la serie real hoy; la leyenda "Predicción IA" se renderiza únicamente
  // si una segunda serie existiera (nunca una línea vacía ni valores del mockup).
  const hasPredictionSeries = useMemo(
    () => chartData.some((point) => "prediccion" in point),
    [chartData]
  );

  // Badge de delta solo cuando hay comparación de periodo calculable (ventas reales vs anterior).
  const salesDelta = useMemo<KpiDelta | undefined>(() => {
    if (!salesAvailable || revenueQuery.isLoading || prevRevenueQuery.isLoading) return undefined;
    const current = (revenueQuery.data ?? []).reduce((sum, p) => sum + p.monto_total, 0);
    const previous = (prevRevenueQuery.data ?? []).reduce((sum, p) => sum + p.monto_total, 0);
    if (previous <= 0) return undefined;
    const pct = Math.round(((current - previous) / previous) * 100);
    if (pct === 0) return undefined;
    return { direction: pct > 0 ? "up" : "down", value: Math.abs(pct) };
  }, [revenueQuery.data, prevRevenueQuery.data, revenueQuery.isLoading, prevRevenueQuery.isLoading, salesAvailable]);

  const isLoadingKpis = conversationsQuery.isLoading;

  const firstName = useMemo(() => {
    const name = profile?.name?.trim();
    if (!name) return undefined;
    return name.split(/\s+/)[0];
  }, [profile?.name]);

  const greeting = useMemo(() => {
    if (!firstName) return ui.dashboard.title;
    return `${timeGreeting()}, ${firstName} ${ui.layout.greetingSuffix}`;
  }, [firstName]);

  const todayLabel = useMemo(() => {
    const label = new Intl.DateTimeFormat("es-CO", {
      weekday: "long",
      day: "numeric",
      month: "long",
    }).format(new Date());
    return label.charAt(0).toUpperCase() + label.slice(1);
  }, []);

  return (
    <div className="space-y-8">
      {/* Saludo + fecha + selector de periodo + CTA "Nueva Conversación" */}
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div className="space-y-1">
          <h1 className="text-2xl font-bold tracking-tight text-foreground sm:text-3xl">
            {greeting}
          </h1>
          <p className="text-sm text-muted-foreground">{todayLabel}</p>
        </div>
        <div className="flex items-center gap-2">
          {salesAvailable && (
            <div className="flex flex-wrap items-center gap-1 rounded-xl border border-border bg-card p-1">
              {SALES_PERIODS.map((p) => (
                <button
                  key={p.key}
                  onClick={() => setSalesPeriod(p.key)}
                  className={cn(
                    "rounded-lg px-3 py-1.5 text-xs font-medium transition-colors",
                    salesPeriod === p.key
                      ? "bg-primary text-primary-foreground"
                      : "text-muted-foreground hover:bg-accent hover:text-accent-foreground"
                  )}
                >
                  {p.label}
                </button>
              ))}
            </div>
          )}
          <Link
            href="/dashboard/inbox"
            className="inline-flex items-center gap-2 rounded-xl bg-primary px-4 py-2.5 text-sm font-semibold text-primary-foreground transition-colors hover:bg-primary/90"
          >
            <MessageSquare className="h-4 w-4" aria-hidden />
            {ui.layout.newConversation}
          </Link>
        </div>
      </div>

      {/* Fila de 4 KPIs (datos reales o "—", nunca cifras inventadas) */}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <KpiCard
          label={ui.dashboard.kpiActiveConversations}
          value={String(openConversations)}
          isLoading={isLoadingKpis}
          hint={ui.dashboard.kpiActiveConversationsHint}
          icon={MessageSquare}
        />
        <KpiCard
          label={ui.dashboard.kpiWeekSales}
          value={
            !salesAvailable
              ? ui.dashboard.noData
              : revenueQuery.isLoading
                ? ui.dashboard.noData
                : formatCOP(weekSales)
          }
          hint={ui.dashboard.kpiWeekSalesHint}
          delta={salesDelta}
          icon={TrendingUp}
          iconClass="bg-primary/10 text-primary"
        />
        <KpiCard
          label={ui.dashboard.kpiInvoicesIssued}
          value={ui.dashboard.noData}
          hint={ui.dashboard.kpiInvoicesIssuedHint}
          icon={FileText}
          iconClass="bg-blue-50 text-blue-600"
        />
        <KpiCard
          label={ui.dashboard.kpiAiResponseTime}
          value={ui.dashboard.noData}
          hint={ui.dashboard.kpiAiResponseTimeHint}
          icon={Clock}
          iconClass="bg-amber-50 text-amber-600"
        />
      </div>

      {/* Chart "Rendimiento de Ventas" + panel Copiloto IA */}
      <div className="grid gap-6 lg:grid-cols-5">
        <Card className="lg:col-span-3">
          <CardHeader>
            <CardTitle className="text-lg text-foreground">{ui.dashboard.salesChartTitle}</CardTitle>
            <CardDescription>{ui.dashboard.salesChartDesc}</CardDescription>
          </CardHeader>
          <CardContent>
            {!salesAvailable ? (
              <div className="flex flex-col items-center justify-center rounded-xl border border-dashed border-border py-14 text-center">
                <BarChartIcon className="h-8 w-8 text-muted-foreground/50" aria-hidden />
                <p className="mt-3 max-w-xs text-sm text-muted-foreground">
                  {ui.dashboard.salesChartEmpty}
                </p>
                <Link
                  href="/dashboard/reportes"
                  className="mt-4 inline-flex items-center gap-1.5 rounded-lg bg-primary px-3.5 py-2 text-xs font-semibold text-primary-foreground transition-colors hover:bg-primary/90"
                >
                  {ui.dashboard.salesChartCta}
                  <ArrowRight className="h-3.5 w-3.5" aria-hidden />
                </Link>
              </div>
            ) : revenueQuery.isLoading ? (
              <div className="h-64">
                <Skeleton className="h-full w-full" />
              </div>
            ) : chartData.length === 0 ? (
              <div className="flex flex-col items-center justify-center rounded-xl border border-dashed border-border py-14 text-center">
                <p className="max-w-xs text-sm text-muted-foreground">
                  {ui.dashboard.salesChartEmpty}
                </p>
                <Link
                  href="/dashboard/reportes"
                  className="mt-4 inline-flex items-center gap-1.5 rounded-lg bg-primary px-3.5 py-2 text-xs font-semibold text-primary-foreground transition-colors hover:bg-primary/90"
                >
                  {ui.dashboard.salesChartCta}
                  <ArrowRight className="h-3.5 w-3.5" aria-hidden />
                </Link>
              </div>
            ) : (
              <div>
                <div className="mb-3 flex items-center gap-4 text-xs text-muted-foreground">
                  <span className="flex items-center gap-1.5">
                    <span className="h-2 w-2 rounded-full bg-emerald-500" aria-hidden />
                    {ui.dashboard.salesLegendReal}
                  </span>
                  {hasPredictionSeries && (
                    <span className="flex items-center gap-1.5">
                      <span className="h-2 w-2 rounded-full bg-blue-500" aria-hidden />
                      {ui.dashboard.salesLegendPrediction}
                    </span>
                  )}
                </div>
                <div className="h-64 w-full">
                  <ResponsiveContainer width="100%" height="100%">
                    <BarChart data={chartData} margin={{ top: 8, right: 8, left: 0, bottom: 0 }}>
                      <CartesianGrid strokeDasharray="3 3" stroke="var(--border)" vertical={false} />
                      <XAxis
                        dataKey="name"
                        tickLine={false}
                        axisLine={false}
                        tick={{ fontSize: 12, fill: "var(--muted-foreground)" }}
                      />
                      <YAxis
                        tickLine={false}
                        axisLine={false}
                        width={64}
                        tick={{ fontSize: 11, fill: "var(--muted-foreground)" }}
                        tickFormatter={(value: number) =>
                          value >= 1000 ? `${Math.round(value / 1000)}k` : String(value)
                        }
                      />
                      <Tooltip
                        cursor={{ fill: "var(--muted)" }}
                        formatter={(value: number) => [formatCOP(value), "Ventas"]}
                        contentStyle={{
                          background: "var(--popover)",
                          border: "1px solid var(--border)",
                          borderRadius: 12,
                          fontSize: 12,
                        }}
                      />
                      <Bar dataKey="ventas" fill="#10b981" radius={[6, 6, 0, 0]} maxBarSize={28} />
                    </BarChart>
                  </ResponsiveContainer>
                </div>
              </div>
            )}
          </CardContent>
        </Card>

        <div className="rounded-2xl bg-primary p-6 text-primary-foreground lg:col-span-2">
          <div className="flex items-center gap-2.5">
            <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-primary-foreground/10">
              <Bot className="h-5 w-5 text-primary-foreground" aria-hidden />
            </div>
            <div>
              <h2 className="font-semibold">{ui.dashboard.copilotTitle}</h2>
              <p className="text-xs text-primary-foreground/70">{ui.dashboard.copilotSubtitle}</p>
            </div>
          </div>
          <ul className="mt-5 space-y-4">
            {COPILOT_INSIGHTS.map((insight, index) => (
              <li key={index} className="flex items-start gap-3">
                <span
                  className={cn("mt-1.5 h-2 w-2 flex-none rounded-full", insight.dotClass)}
                  aria-hidden="true"
                />
                <p className="text-sm leading-relaxed text-primary-foreground/80">{insight.text}</p>
              </li>
            ))}
          </ul>
          <Link
            href="/dashboard/settings?view=ai"
            className="mt-6 inline-flex w-full items-center justify-center gap-2 rounded-xl bg-white py-2.5 text-sm font-semibold text-primary transition-colors hover:bg-white/90"
          >
            <Sparkles className="h-4 w-4" aria-hidden />
            {ui.dashboard.copilotCta}
          </Link>
        </div>
      </div>

      {/* Fila de 3 paneles: Conversaciones Recientes / Rendimiento del Equipo / Facturas Siigo */}
      <div className="grid gap-6 lg:grid-cols-3">
        <RecentConversationsPanel
          conversations={conversationsQuery.data ?? []}
          isLoading={conversationsQuery.isLoading}
        />
        <TeamPerformancePanel
          organizationId={profile?.organization?.organization_id}
          canManageMembers={canManageMembers}
        />
        <SiigoInvoicesPanel canViewInvoices={canViewInvoices} />
      </div>

      {/* Banner Auto-Piloto */}
      <AutoPilotBanner canManageMembers={canManageMembers} />

      {/* Acciones Rápidas operativas */}
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

      {/* Onboarding helpers conservados: intro + checklist colapsable (contratos ai-onboarding) */}
      {/* [&>*]:min-w-0 evita que el min-content de las tarjetas ensanche el track del grid en móvil */}
      <div className="grid gap-6 lg:grid-cols-2 [&>*]:min-w-0">
        <AssistantIntro />
        <OnboardingChecklistSection />
      </div>
    </div>
  );
}
