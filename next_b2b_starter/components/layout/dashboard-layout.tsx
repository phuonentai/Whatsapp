// components/layout/dashboard-layout.tsx
"use client";

import { useCallback, useEffect, useMemo, useState, createContext, useContext } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";

import { Sidebar } from "./sidebar";
import { Header } from "./header";
import { PlansModal } from "@/components/billing/plans-modal";
import {
  SubscriptionAlerts,
  type SubscriptionAlertDescriptor,
  type SubscriptionAlertAction,
} from "@/components/billing/subscription-alerts";
import { cn } from "@/lib/utils";
import { useSidebarStore } from "@/lib/stores/sidebar-store";
import type { ServerPermissions } from "@/lib/auth/server-permissions";
import { useAuthContext } from "@/lib/contexts/auth-context";
import { PERMISSIONS } from "@/lib/auth/permissions";
import { useSubscriptionQuery } from "@/lib/hooks/queries/use-subscription-query";
import { queryKeys } from "@/lib/hooks/queries/query-keys";
import type { SubscriptionGateState } from "@/lib/polar/current-subscription";
import { useIsPlansModalOpen, useUIStore } from "@/stores/ui-store";

interface DashboardLayoutProps {
  children: React.ReactNode;
  initialSubscription: SubscriptionGateState;
}

export function DashboardLayout({
  children,
  initialSubscription,
}: DashboardLayoutProps) {
  const isSidebarCollapsed = useSidebarStore((state) => state.isCollapsed);
  const setAutoCollapsed = useSidebarStore((state) => state.setAutoCollapsed);
  const auth = useAuthContext();
  const queryClient = useQueryClient();
  const isPlansModalOpen = useIsPlansModalOpen();
  const setPlansModalOpen = useUIStore((state) => state.setPlansModalOpen);
  const openPlansModal = useCallback(() => setPlansModalOpen(true), [setPlansModalOpen]);
  const handlePlansModalOpenChange = useCallback(
    (open: boolean) => setPlansModalOpen(open),
    [setPlansModalOpen]
  );

  useEffect(() => {
    queryClient.setQueryData(
      queryKeys.subscription.status(),
      initialSubscription
    );
  }, [initialSubscription, queryClient]);

  // Show toast notification when server is unreachable
  useEffect(() => {
    if (!initialSubscription.backendAvailable) {
      toast.error("Server Unreachable", {
        description: "Unable to connect to the server. Some features may be unavailable.",
        duration: 5000,
      });
    }
  }, [initialSubscription.backendAvailable]);

  const permissions: ServerPermissions | null = useMemo(() => {
    if (!auth) {
      return null;
    }

    const profile = auth.profile;
    const roles = auth.roles;
    const granted = auth.permissions;

    return {
      profile,
      roles,
      permissions: granted,
      canViewInvoices: granted.includes(PERMISSIONS.INVOICE_VIEW),
      canCreateInvoices: granted.includes(PERMISSIONS.INVOICE_CREATE),
      canUploadInvoices: granted.includes(PERMISSIONS.INVOICE_UPLOAD),
      canDeleteInvoices: granted.includes(PERMISSIONS.INVOICE_DELETE),
      canViewApprovals: granted.includes(PERMISSIONS.APPROVALS_VIEW),
      canApproveInvoices: granted.includes(PERMISSIONS.APPROVALS_APPROVE),
      canViewDuplicates: granted.includes(PERMISSIONS.DUPLICATES_VIEW),
      canResolveDuplicates: granted.includes(PERMISSIONS.DUPLICATES_RESOLVE),
      canSchedulePayments: granted.includes(PERMISSIONS.PAYMENT_OPTIMIZATION_SCHEDULE),
      canExportPayments: granted.includes(PERMISSIONS.PAYMENT_OPTIMIZATION_EXPORT),
      canExecutePayments: granted.includes(PERMISSIONS.PAYMENT_OPTIMIZATION_EXECUTE),
      canViewAudit: granted.includes(PERMISSIONS.AUDIT_VIEW),
      canViewOrganization: granted.includes(PERMISSIONS.ORG_VIEW),
      canManageOrganization: granted.includes(PERMISSIONS.ORG_MANAGE),
      canManageSubscriptions: granted.includes(PERMISSIONS.ORG_MANAGE),
      backendAvailable: true,
      backendError: null,
    };
  }, [auth]);

  const shouldLoadSubscription = Boolean(
    permissions?.canManageSubscriptions
  );

  const { data: subscriptionData } = useSubscriptionQuery({
    enabled: shouldLoadSubscription,
    initialData: initialSubscription,
  });

  const subscriptionState =
    subscriptionData ?? initialSubscription ?? null;

  const {
    alerts,
  } = useMemo(
    () => deriveSubscriptionUiState(subscriptionState, openPlansModal),
    [subscriptionState, openPlansModal]
  );

  useEffect(() => {
    const mediaQuery = window.matchMedia("(max-width: 1279px)");

    const applyMatch = (matches: boolean) => {
      setAutoCollapsed(matches);
    };

    const handleChange = (event: MediaQueryListEvent) => {
      applyMatch(event.matches);
    };

    applyMatch(mediaQuery.matches);

    if (typeof mediaQuery.addEventListener === "function") {
      mediaQuery.addEventListener("change", handleChange);
      return () => mediaQuery.removeEventListener("change", handleChange);
    }

    const legacyHandler = () => applyMatch(mediaQuery.matches);
    mediaQuery.addListener(legacyHandler);
    return () => mediaQuery.removeListener(legacyHandler);
  }, [setAutoCollapsed]);

  const isReady = Boolean(auth?.isInitialized && permissions);

  if (!isReady || !permissions) {
    return (
      <div className="min-h-screen bg-white flex items-center justify-center">
        <div className="text-center">
          <div className="h-8 w-8 border-4 border-gray-200 border-t-gray-900 rounded-full animate-spin mx-auto mb-4" />
          <p className="text-gray-600">Loading...</p>
        </div>
      </div>
    );
  }

  return (
      <div className="min-h-screen bg-white">
        <Sidebar
          permissions={permissions}
        />
        <Header />

        <main
          className={cn(
            "p-6 transition-[padding] duration-200",
            isSidebarCollapsed ? "lg:pl-24" : "lg:pl-64"
          )}
        >
          <div className="mx-auto max-w-7xl space-y-6">
            <SubscriptionAlerts alerts={alerts} />
            {children}
          </div>
        </main>

        <PlansModal
          open={isPlansModalOpen}
          onOpenChange={handlePlansModalOpenChange}
          subscriptionState={subscriptionState}
        />
      </div>
  );
}

function deriveSubscriptionUiState(
  state: SubscriptionGateState | null | undefined,
  openPlansModal: () => void
): {
  alerts: SubscriptionAlertDescriptor[];
} {
  if (!state) {
    return {
      alerts: [],
    };
  }

  const alerts: SubscriptionAlertDescriptor[] = [];
  const settingsHref = "/dashboard/settings?view=subscription";
  const reason = state.reason ?? null;

  const pushAlert = (alert: SubscriptionAlertDescriptor) => {
    alerts.push(alert);
  };

  if (!state.isActive) {
    if (reason === "INSUFFICIENT_PERMISSIONS") {
      alerts.push({
        id: "subscription-permissions",
        variant: "info",
        title: "Limited billing visibility",
        description:
          "You don't have permission to view subscription details.",
      });
      return {
        alerts,
      };
    }

    if (reason === "POLAR_UNCONFIGURED") {
      alerts.push({
        id: "subscription-unconfigured",
        variant: "info",
        title: "Billing configuration required",
        description:
          "We couldn't verify your subscription because Polar credentials are missing. Add them in settings to enable monitoring.",
        actions: [
          {
            label: "Open billing settings",
            href: settingsHref,
            priority: "secondary",
          },
        ],
      });
      return {
        alerts,
      };
    }

    pushAlert({
      id: "subscription-inactive",
      variant: "critical",
      title: "Subscription inactive",
      description: getInactiveDescription(state),
      actions: [
        {
          label: "Subscribe now",
          onClick: openPlansModal,
          priority: "primary",
        },
        {
          label: "Manage subscription",
          href: settingsHref,
          priority: "secondary",
        },
      ],
    });
  }

  const usage = state.usage;
  if (usage && usage.included > 0) {
    const { included, used, remaining } = usage;
    const usageRatio = included > 0 ? used / included : 0;

    if (remaining <= 0) {
      pushAlert({
        id: "subscription-usage-max",
        variant: "critical",
        title: "Usage limit reached",
        description: `You've used ${formatNumber(
          used
        )} of ${formatNumber(
          included
        )} units this billing period. Upgrade or extend your plan to continue.`,
        actions: [
          {
            label: "Upgrade plan",
            onClick: openPlansModal,
            priority: "primary",
          },
          {
            label: "Manage billing",
            href: settingsHref,
            priority: "secondary",
          },
        ],
      });
    } else {
      const thresholdPercent = 0.85;
      const thresholdRemaining = Math.max(5, Math.ceil(included * 0.05));
      if (
        usageRatio >= thresholdPercent ||
        remaining <= thresholdRemaining
      ) {
        alerts.push({
          id: "subscription-usage-warning",
          variant: "warning",
          title: "You're nearing your usage limit",
          description: `Only ${formatNumber(
            remaining
          )} unit${remaining === 1 ? "" : "s"} remain in this billing period.`,
          actions: [
            {
              label: "Review plans",
              onClick: openPlansModal,
              priority: "primary",
            },
            {
              label: "Track usage",
              href: settingsHref,
              priority: "secondary",
            },
          ],
        });
      }
    }
  }

  const subscription = state.subscription;
  if (subscription?.cancelAtPeriodEnd) {
    const cancelDate = subscription.currentPeriodEnd
      ? formatDateString(subscription.currentPeriodEnd)
      : "the end of this billing period";
    alerts.push({
      id: "subscription-cancelled",
      variant: "warning",
      title: "Subscription scheduled to cancel",
      description: `Your current plan will end on ${cancelDate}. Resume the subscription to maintain uninterrupted access.`,
      actions: [
        {
          label: "Resume subscription",
          href: settingsHref,
          priority: "primary",
        },
      ],
    });
  }

  const trialEnd = subscription?.trialEnd;
  if (trialEnd) {
    const daysLeft = daysUntil(trialEnd);
    if (daysLeft !== null && daysLeft <= 7) {
      alerts.push({
        id: "subscription-trial-ending",
        variant: daysLeft <= 2 ? "critical" : "warning",
        title: "Trial ending soon",
        description: `Your trial ends on ${formatDateString(
          trialEnd
        )}. Add a payment method to stay active.`,
        actions: [
          {
            label: "Secure your plan",
            onClick: openPlansModal,
            priority: "primary",
          },
          {
            label: "Update billing details",
            href: settingsHref,
            priority: "secondary",
          },
        ],
      });
    }
  }

  if (reason === "UNKNOWN_ERROR") {
    alerts.push({
      id: "subscription-unknown-error",
      variant: "info",
      title: "Subscription status unavailable",
      description:
        "We couldn't refresh your subscription details. We'll retry automatically, or you can refresh the page.",
    });
  }

  return {
    alerts,
  };
}

function getInactiveDescription(state: SubscriptionGateState): string {
  const reason = state.reason ?? null;
  const status = state.status ?? null;

  switch (reason) {
    case "CUSTOMER_NOT_FOUND":
      return "We couldn't match your workspace to an active billing account. Start a subscription to unlock premium features.";
    case "NO_ACTIVE_SUBSCRIPTION":
      if (status === "past_due") {
        return "We couldn't process your latest payment. Update billing details to resume service.";
      }
      return "Your subscription has ended. Restart your plan to continue using the service.";
    case "PROFILE_UNAVAILABLE":
      return "We couldn't load your profile to verify billing status. Please refresh the page or contact support.";
    case "UNKNOWN_ERROR":
      return "We couldn't verify your subscription. Try again shortly or contact support if this persists.";
    default:
      return "We could not confirm an active subscription for this workspace. Update your plan to continue.";
  }
}

function formatNumber(value: number): string {
  return new Intl.NumberFormat().format(value);
}

function formatDateString(value: string): string {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }

  return date.toLocaleDateString(undefined, {
    month: "short",
    day: "numeric",
    year: "numeric",
  });
}

function daysUntil(value: string): number | null {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return null;
  }

  const diff = date.getTime() - Date.now();
  return Math.ceil(diff / (1000 * 60 * 60 * 24));
}
